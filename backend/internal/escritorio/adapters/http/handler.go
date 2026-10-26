// Package http es el handler local del escritorio, el "servidor" que ve la pantalla.
//
//	POST /api/ventas                   se cobra siempre en local y se sube enseguida
//	GET  /api/productos                del servidor; sin conexión, de la copia del catálogo
//	GET  /api/productos/codigo/{c}     ídem
//	POST /api/auth/login               en el servidor; sin conexión, con la última contraseña usada en esta PC
//	GET  /api/escritorio/estado        conexión y ventas pendientes de subir
//	/api/*                             se reenvía al servidor; sin conexión responde 503 sin_conexion
//	/*                                 la app (si se configuró)
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"magui/internal/escritorio/application"
	"magui/internal/kernel/reloj"
	"magui/internal/platform/httpx"
	ventashttp "magui/internal/ventas/adapters/http"
)

// VentanaSinConexion: después de una falla, durante este tiempo las búsquedas van directo a la copia
// (no se espera al servidor en cada tecla).
const VentanaSinConexion = 20 * time.Second

// Opciones del handler local.
type Opciones struct {
	Servidor *url.URL // servidor central, ej. https://magui.ejemplo.com
	Ventas   *ventashttp.Handler
	Catalogo *application.CatalogoLocal
	Sincro   *application.Sincronizador
	Ingreso  *application.Ingreso
	Conexion *application.Conexion
	Reloj    reloj.Reloj
	Log      *slog.Logger
	Frontend http.Handler // opcional: la app compilada
	// Transporte hacia el servidor (en tests, uno falso).
	Transporte http.RoundTripper
}

type handler struct {
	o       Opciones
	proxy   *httputil.ReverseProxy
	cliente *http.Client

	mu          sync.Mutex
	ultimoToken string
}

var errServidorCaido = errors.New("el servidor no responde")

// NuevoHandler arma el router local.
func NuevoHandler(o Opciones) http.Handler {
	if o.Transporte == nil {
		o.Transporte = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		}
	}
	h := &handler{o: o, cliente: &http.Client{Transport: o.Transporte, Timeout: 30 * time.Second}}
	h.proxy = &httputil.ReverseProxy{
		Transport: o.Transporte,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(o.Servidor)
			pr.Out.Host = o.Servidor.Host
			pr.Out.Header.Del("Origin") // el servidor ve un pedido directo, no uno de otro sitio
			pr.Out.Header.Del("Referer")
		},
		ModifyResponse: h.alResponder,
		ErrorHandler:   h.alFallar,
	}

	r := chi.NewRouter()
	r.Get("/api/escritorio/estado", h.estado)
	r.Post("/api/ventas", h.vender)
	r.Get("/api/productos", h.conCopia(h.buscarLocal))
	r.Get("/api/productos/codigo/{codigo}", h.conCopia(h.porCodigoLocal))
	r.Post("/api/auth/login", h.ingresar)
	r.Post("/api/auth/cambiar-clave", h.cambiarClave)
	r.Handle("/api/*", h.proxy)
	if o.Frontend != nil {
		r.Handle("/*", o.Frontend)
	}
	return r
}

// --- Reenvío al servidor ---

type claveRespaldo struct{}

// conCopia reenvía al servidor y, si no responde (o hace poco que no responde), contesta desde la copia.
func (h *handler) conCopia(local http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.o.Conexion.SinRespuestaReciente(h.o.Reloj.Ahora(), VentanaSinConexion) {
			local(w, r)
			return
		}
		h.proxy.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claveRespaldo{}, local)))
	}
}

func (h *handler) alResponder(res *http.Response) error {
	switch res.StatusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return errServidorCaido // lo atiende alFallar
	}
	h.o.Conexion.Informar(true, h.o.Reloj.Ahora())
	if res.StatusCode < 300 {
		h.recordarToken(res.Request)
	}
	return nil
}

func (h *handler) alFallar(w http.ResponseWriter, r *http.Request, err error) {
	if r.Context().Err() != nil {
		return // la pantalla canceló el pedido
	}
	h.o.Conexion.Informar(false, h.o.Reloj.Ahora())
	if local, ok := r.Context().Value(claveRespaldo{}).(http.HandlerFunc); ok {
		local(w, r)
		return
	}
	sinConexion(w)
}

func sinConexion(w http.ResponseWriter) {
	httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{"error": map[string]string{
		"code":    "sin_conexion",
		"message": "Sin conexión con el servidor. Esta pantalla necesita internet; Vender sigue funcionando.",
	}})
}

// recordarToken guarda el token que el servidor aceptó, para que la sincronización lo use.
func (h *handler) recordarToken(r *http.Request) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" || strings.HasPrefix(token, application.PrefijoTokenLocal) {
		return
	}
	h.mu.Lock()
	nuevo := token != h.ultimoToken
	h.ultimoToken = token
	h.mu.Unlock()
	if nuevo {
		if err := h.o.Ingreso.ActualizarToken(context.WithoutCancel(r.Context()), token); err != nil {
			h.o.Log.Error("guardar sesión", "error", err)
		}
	}
}

// --- Venta local ---

func (h *handler) vender(w http.ResponseWriter, r *http.Request) {
	h.o.Ventas.Registrar(w, r)
	h.o.Sincro.Avisar() // se sube enseguida si hay conexión
}

// --- Productos desde la copia ---

func (h *handler) buscarLocal(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := h.o.Catalogo.Buscar(r.Context(), application.ConsultaLocal{
		Texto: q.Get("q"), Estado: q.Get("estado"), Tipo: q.Get("tipo")}, httpx.Paginacion(r))
	if err != nil {
		httpx.Error(w, r, h.o.Log, err)
		return
	}
	items := make([]productoLocal, len(res.Items))
	for i, p := range res.Items {
		items[i] = local(p)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items": items, "page": res.Page, "size": res.Size, "total": res.Total, "totalPages": res.TotalPages})
}

func (h *handler) porCodigoLocal(w http.ResponseWriter, r *http.Request) {
	p, err := h.o.Catalogo.PorCodigo(r.Context(), chi.URLParam(r, "codigo"))
	if err != nil {
		httpx.Error(w, r, h.o.Log, err)
		return
	}
	httpx.JSON(w, http.StatusOK, local(p))
}

// --- Estado ---

func (h *handler) estado(w http.ResponseWriter, r *http.Request) {
	e, err := h.o.Sincro.Estado(r.Context())
	if err != nil {
		httpx.Error(w, r, h.o.Log, err)
		return
	}
	httpx.JSON(w, http.StatusOK, e)
}

// --- Ingreso ---

type sesionServidor struct {
	Token    string    `json:"token"`
	ExpiraEn time.Time `json:"expiraEn"`
}

func (h *handler) ingresar(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Clave string `json:"clave"`
	}
	cuerpo, ok := h.leer(w, r, &req)
	if !ok {
		return
	}
	status, resp, err := h.reenviar(r, cuerpo)
	if err != nil { // sin conexión: se ingresa con la contraseña guardada
		s, err := h.o.Ingreso.IngresarSinConexion(r.Context(), req.Clave)
		if err != nil {
			httpx.Error(w, r, h.o.Log, err)
			return
		}
		httpx.JSON(w, http.StatusOK, s)
		return
	}
	h.alIngresar(r.Context(), status, resp, req.Clave)
	responderCrudo(w, status, resp)
}

func (h *handler) cambiarClave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nueva string `json:"nueva"`
	}
	cuerpo, ok := h.leer(w, r, &req)
	if !ok {
		return
	}
	status, resp, err := h.reenviar(r, cuerpo)
	if err != nil {
		sinConexion(w)
		return
	}
	h.alIngresar(r.Context(), status, resp, req.Nueva)
	responderCrudo(w, status, resp)
}

// alIngresar recuerda la sesión que dio el servidor y avisa a la sincronización (puede haber ventas esperando).
func (h *handler) alIngresar(ctx context.Context, status int, resp []byte, clave string) {
	if status != http.StatusOK {
		return
	}
	var s sesionServidor
	if json.Unmarshal(resp, &s) != nil || s.Token == "" {
		return
	}
	if err := h.o.Ingreso.Recordar(context.WithoutCancel(ctx), s.Token, s.ExpiraEn, clave); err != nil {
		h.o.Log.Error("guardar sesión", "error", err)
		return
	}
	h.mu.Lock()
	h.ultimoToken = s.Token
	h.mu.Unlock()
	h.o.Sincro.Avisar()
}

// leer guarda el cuerpo (para reenviarlo tal cual) y lo decodifica.
func (h *handler) leer(w http.ResponseWriter, r *http.Request, dst any) ([]byte, bool) {
	cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, httpx.TamanoMaximoCuerpo))
	if err == nil {
		err = json.Unmarshal(cuerpo, dst)
	}
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{
			"code": "json_invalido", "message": "El cuerpo del pedido no es JSON válido"}})
		return nil, false
	}
	return cuerpo, true
}

// reenviar manda el pedido al servidor y devuelve su respuesta; err indica que no hubo conexión.
func (h *handler) reenviar(r *http.Request, cuerpo []byte) (int, []byte, error) {
	destino := *h.o.Servidor
	destino.Path = r.URL.Path
	req, err := http.NewRequestWithContext(r.Context(), r.Method, destino.String(), bytes.NewReader(cuerpo))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a := r.Header.Get("Authorization"); a != "" {
		req.Header.Set("Authorization", a)
	}
	res, err := h.cliente.Do(req)
	if err == nil && (res.StatusCode == http.StatusBadGateway || res.StatusCode == http.StatusServiceUnavailable ||
		res.StatusCode == http.StatusGatewayTimeout) {
		res.Body.Close()
		err = errServidorCaido
	}
	if err != nil {
		h.o.Conexion.Informar(false, h.o.Reloj.Ahora())
		return 0, nil, err
	}
	defer res.Body.Close()
	h.o.Conexion.Informar(true, h.o.Reloj.Ahora())
	resp, err := io.ReadAll(res.Body)
	return res.StatusCode, resp, err
}

func responderCrudo(w http.ResponseWriter, status int, cuerpo []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(cuerpo)
}
