package escritorio_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"magui/internal/escritorio"
	"magui/internal/escritorio/application"
	"magui/internal/kernel/dinero"
	sincro "magui/internal/sync/application"
)

// servidorFalso imita el servidor central: sincronización, ingreso y una ruta cualquiera.
type servidorFalso struct {
	mu       sync.Mutex
	stock    map[string]int
	ventas   map[string]sincro.VentaOffline
	envios   int
	rechazar string // producto cuyas ventas se rechazan
	tokenOK  string
}

func nuevoServidorFalso() *servidorFalso {
	return &servidorFalso{stock: map[string]int{"kaiak": 3, "jabon": 10}, ventas: map[string]sincro.VentaOffline{},
		tokenOK: "tok-servidor", rechazar: "baja"}
}

func ptr(s string) *string { return &s }

func (f *servidorFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	responder := func(status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	if r.URL.Path == "/api/auth/login" {
		var req struct{ Clave string }
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Clave != "secreta123" {
			responder(401, map[string]any{"error": map[string]string{"code": "clave_incorrecta", "message": "La contraseña es incorrecta"}})
			return
		}
		responder(200, map[string]any{"token": f.tokenOK, "expiraEn": time.Now().Add(720 * time.Hour)})
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+f.tokenOK {
		responder(401, map[string]any{"error": map[string]string{"code": "sesion_invalida", "message": "x"}})
		return
	}
	switch r.URL.Path {
	case "/api/sync/catalogo":
		responder(200, sincro.CopiaCatalogo{GeneradaEn: time.Now(),
			Productos: []sincro.ProductoCopia{
				{ID: "kaiak", SKU: "P1", CodigoBarras: ptr("779001"), Nombre: "Kaiak Clásico", Tipo: "SIMPLE", PrecioVenta: dinero.MustParse("59200"), Stock: f.stock["kaiak"], Activo: true, Version: 1},
				{ID: "jabon", SKU: "P2", Nombre: "Jabón Ekos", Tipo: "SIMPLE", PrecioVenta: dinero.MustParse("5500"), Stock: f.stock["jabon"], Activo: true, Version: 1},
				{ID: "baja", SKU: "P3", Nombre: "Producto de baja", Tipo: "SIMPLE", PrecioVenta: dinero.MustParse("100"), Stock: 5, Activo: false, Version: 1},
				{ID: "combo", SKU: "C1", Nombre: "Combo Madre", Tipo: "COMBO", PrecioVenta: dinero.MustParse("62000"), Activo: true, Version: 1},
			},
			Componentes: []sincro.ComponenteCopia{{ComboID: "combo", ProductoID: "kaiak", Cantidad: 1}, {ComboID: "combo", ProductoID: "jabon", Cantidad: 2}},
		})
	case "/api/sync/ventas":
		f.envios++
		var req sincro.EnvioVentas
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := sincro.RespuestaEnvio{}
		for _, v := range req.Ventas {
			out := sincro.ResultadoEnvio{ID: v.ID, Estado: sincro.Registrada}
			switch {
			case f.ventas[v.ID].ID != "":
				out.Estado = sincro.YaExistia
			case v.Items[0].ProductoID == f.rechazar:
				out.Estado, out.Error = sincro.Rechazada, &sincro.ErrorEnvio{Code: "x", Message: "rechazada a propósito"}
			default:
				f.ventas[v.ID] = v
				for _, it := range v.Items {
					if it.ProductoID == "combo" { // el combo descuenta sus componentes
						f.stock["kaiak"] -= it.Cantidad
						f.stock["jabon"] -= 2 * it.Cantidad
						continue
					}
					f.stock[it.ProductoID] -= it.Cantidad
				}
			}
			res.Resultados = append(res.Resultados, out)
		}
		responder(200, res)
	default:
		responder(200, map[string]any{"desde": "servidor", "ruta": r.URL.Path})
	}
}

// red deja "desenchufar" el servidor: sin red, cada pedido falla como sin internet.
type red struct{ cortada atomic.Bool }

func (n *red) RoundTrip(r *http.Request) (*http.Response, error) {
	if n.cortada.Load() {
		return nil, errors.New("dial tcp: sin internet")
	}
	return http.DefaultTransport.RoundTrip(r)
}

type respuesta struct {
	code int
	body map[string]any
}

func pedir(t *testing.T, h http.Handler, metodo, ruta, token string, cuerpo any) respuesta {
	t.Helper()
	var body io.Reader
	if cuerpo != nil {
		b, _ := json.Marshal(cuerpo)
		body = bytes.NewReader(b)
	}
	req := httptest.NewRequest(metodo, ruta, body)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return respuesta{rec.Code, out}
}

func items(r respuesta) []map[string]any {
	lista, _ := r.body["items"].([]any)
	out := make([]map[string]any, len(lista))
	for i, x := range lista {
		out[i] = x.(map[string]any)
	}
	return out
}

func stockLocal(t *testing.T, h http.Handler, sku string) float64 {
	t.Helper()
	r := pedir(t, h, http.MethodGet, "/api/productos/codigo/"+sku, "", nil)
	if r.code != 200 {
		t.Fatalf("producto %s: %d %v", sku, r.code, r.body)
	}
	return r.body["stock"].(float64)
}

func venta(producto string, cantidad int, pagos ...map[string]any) map[string]any {
	return map[string]any{"items": []map[string]any{{"productoId": producto, "cantidad": cantidad}}, "pagos": pagos, "origen": "WEB"}
}

func pago(metodo, monto string) map[string]any {
	return map[string]any{"metodo": metodo, "monto": monto}
}

func TestEscritorioVendeSinConexionYSincroniza(t *testing.T) {
	ctx := context.Background()
	falso := nuevoServidorFalso()
	srv := httptest.NewServer(falso)
	defer srv.Close()
	conexion := &red{}
	app, err := escritorio.Abrir(ctx, escritorio.Config{Servidor: srv.URL, Datos: filepath.Join(t.TempDir(), "escritorio.db"),
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Transporte: conexion})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cerrar()
	h := app.Handler

	// Sin haber ingresado nunca no se puede sincronizar.
	if err := app.Sincro.Sincronizar(ctx); !errors.Is(err, application.ErrSesionVencida) {
		t.Fatalf("sin sesión: %v", err)
	}

	// --- Con conexión: ingreso y primera copia ---
	r := pedir(t, h, http.MethodPost, "/api/auth/login", "", map[string]string{"clave": "secreta123"})
	if r.code != 200 || r.body["token"] != "tok-servidor" {
		t.Fatalf("ingreso: %d %v", r.code, r.body)
	}
	if err := app.Sincro.Sincronizar(ctx); err != nil {
		t.Fatal(err)
	}
	// Con conexión, las búsquedas van al servidor
	if r := pedir(t, h, http.MethodGet, "/api/productos?q=kaiak", "tok-servidor", nil); r.body["desde"] != "servidor" {
		t.Errorf("búsqueda con conexión: %v", r.body)
	}

	// --- Se corta internet ---
	conexion.cortada.Store(true)
	r = pedir(t, h, http.MethodGet, "/api/productos?q=clasico&estado=activos&size=8", "tok-servidor", nil)
	if r.code != 200 || len(items(r)) != 1 || items(r)[0]["nombre"] != "Kaiak Clásico" || items(r)[0]["desdeCopia"] != true {
		t.Fatalf("búsqueda sin conexión: %d %v", r.code, r.body)
	}
	if r := pedir(t, h, http.MethodGet, "/api/productos?q=producto", "", nil); len(items(r)) != 0 {
		t.Errorf("los desactivados no aparecen por defecto: %v", items(r))
	}
	if r := pedir(t, h, http.MethodGet, "/api/productos/codigo/779001", "", nil); r.body["sku"] != "P1" {
		t.Errorf("por código de barras: %v", r.body)
	}
	if r := pedir(t, h, http.MethodGet, "/api/productos/codigo/NADA", "", nil); r.code != 404 {
		t.Errorf("código inexistente: %d", r.code)
	}
	if got := stockLocal(t, h, "C1"); got != 3 {
		t.Errorf("stock del combo = %v, want 3", got)
	}

	// Venta en efectivo con vuelto
	r = pedir(t, h, http.MethodPost, "/api/ventas", "tok-servidor", venta("kaiak", 2, pago("EFECTIVO", "120000")))
	if r.code != 201 || r.body["total"] != "118400.00" || r.body["vuelto"] != "1600.00" || r.body["origen"] != "DESKTOP" || r.body["unidades"] != 1.0*2 {
		t.Fatalf("venta 1: %d %v", r.code, r.body)
	}
	if got := stockLocal(t, h, "P1"); got != 1 {
		t.Errorf("stock local de Kaiak = %v, want 1", got)
	}
	// Combo: necesita 1 Kaiak (queda 1) y 2 jabones
	if r := pedir(t, h, http.MethodPost, "/api/ventas", "", venta("combo", 1, pago("QR", "62000"))); r.code != 201 {
		t.Fatalf("combo: %d %v", r.code, r.body)
	}
	if stockLocal(t, h, "P1") != 0 || stockLocal(t, h, "P2") != 8 || stockLocal(t, h, "C1") != 0 {
		t.Error("el combo debe descontar sus componentes del stock local")
	}
	// Sin stock: mismo aviso que el servidor y, confirmando, se vende igual
	r = pedir(t, h, http.MethodPost, "/api/ventas", "", venta("kaiak", 1, pago("TARJETA", "59200")))
	if r.code != 409 || !strings.Contains(r.body["error"].(map[string]any)["message"].(string), "Kaiak Clásico: No hay stock suficiente (disponible: 0)") {
		t.Fatalf("sin stock: %d %v", r.code, r.body)
	}
	sinStock := venta("kaiak", 1, pago("TARJETA", "59200"))
	sinStock["permitirSinStock"] = true
	if r := pedir(t, h, http.MethodPost, "/api/ventas", "", sinStock); r.code != 201 {
		t.Fatalf("vender igual: %d %v", r.code, r.body)
	}
	// Pago incompleto: mismas reglas de cobro
	if r := pedir(t, h, http.MethodPost, "/api/ventas", "", venta("jabon", 1, pago("QR", "100"))); r.code != 409 {
		t.Errorf("pago incompleto: %d %v", r.code, r.body)
	}
	// Desactivado: no se vende
	if r := pedir(t, h, http.MethodPost, "/api/ventas", "", venta("baja", 1, pago("QR", "100"))); r.code != 409 {
		t.Errorf("producto desactivado: %d %v", r.code, r.body)
	}

	// Otras pantallas necesitan internet
	if r := pedir(t, h, http.MethodGet, "/api/ventas", "tok-servidor", nil); r.code != 503 || r.body["error"].(map[string]any)["code"] != "sin_conexion" {
		t.Errorf("listado sin conexión: %d %v", r.code, r.body)
	}
	// Ingreso sin conexión con la última contraseña usada en esta PC
	if r := pedir(t, h, http.MethodPost, "/api/auth/login", "", map[string]string{"clave": "secreta123"}); r.code != 200 || r.body["token"] != "tok-servidor" {
		t.Errorf("ingreso sin conexión: %d %v", r.code, r.body)
	}
	if r := pedir(t, h, http.MethodPost, "/api/auth/login", "", map[string]string{"clave": "otra"}); r.code != 401 {
		t.Errorf("contraseña incorrecta sin conexión: %d", r.code)
	}

	if err := app.Sincro.Sincronizar(ctx); !errors.Is(err, application.ErrSinConexion) {
		t.Errorf("sincronizar sin conexión: %v", err)
	}
	r = pedir(t, h, http.MethodGet, "/api/escritorio/estado", "", nil)
	if r.body["enLinea"] != false || r.body["pendientes"] != 3.0 {
		t.Errorf("estado sin conexión: %v", r.body)
	}

	// --- Vuelve internet: sube las 3 ventas y baja la copia nueva ---
	conexion.cortada.Store(false)
	if err := app.Sincro.Sincronizar(ctx); err != nil {
		t.Fatal(err)
	}
	if len(falso.ventas) != 3 || falso.envios != 1 {
		t.Fatalf("el servidor recibió %d ventas en %d envíos", len(falso.ventas), falso.envios)
	}
	for _, v := range falso.ventas {
		if v.Items[0].ProductoID == "kaiak" && v.Items[0].Cantidad == 2 {
			if v.Pagos[0].Metodo != "EFECTIVO" || v.Pagos[0].Monto.String() != "120000.00" || v.Items[0].Precio.String() != "59200.00" {
				t.Errorf("venta enviada = %+v", v)
			}
		}
	}
	// El servidor descontó: Kaiak 3-2-1(combo)-1 = -1; la copia nueva ya lo refleja y no se resta dos veces.
	conexion.cortada.Store(true) // para mirar la copia local
	if got := stockLocal(t, h, "P1"); got != -1 {
		t.Errorf("stock después de sincronizar = %v, want -1", got)
	}
	conexion.cortada.Store(false)
	if err := app.Sincro.Sincronizar(ctx); err != nil {
		t.Fatal(err)
	}
	r = pedir(t, h, http.MethodGet, "/api/escritorio/estado", "", nil)
	if r.body["enLinea"] != true || r.body["pendientes"] != 0.0 || r.body["ultimaSincronizacion"] == nil {
		t.Errorf("estado después: %v", r.body)
	}
	// Sincronizar de nuevo no reenvió nada
	if falso.envios != 1 {
		t.Errorf("envíos = %d, want 1", falso.envios)
	}
}

func TestEscritorioVentaRechazadaYSesionVencida(t *testing.T) {
	ctx := context.Background()
	falso := nuevoServidorFalso()
	falso.rechazar = "jabon"
	srv := httptest.NewServer(falso)
	defer srv.Close()
	app, err := escritorio.Abrir(ctx, escritorio.Config{Servidor: srv.URL, Datos: filepath.Join(t.TempDir(), "e.db"),
		Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cerrar()
	h := app.Handler
	pedir(t, h, http.MethodPost, "/api/auth/login", "", map[string]string{"clave": "secreta123"})
	if err := app.Sincro.Sincronizar(ctx); err != nil {
		t.Fatal(err)
	}
	pedir(t, h, http.MethodPost, "/api/ventas", "", venta("jabon", 1, pago("QR", "5500")))
	if err := app.Sincro.Sincronizar(ctx); err != nil {
		t.Fatal(err)
	}
	r := pedir(t, h, http.MethodGet, "/api/escritorio/estado", "", nil)
	if r.body["pendientes"] != 0.0 || r.body["rechazadas"] != 1.0 {
		t.Errorf("estado con rechazada: %v", r.body)
	}

	// El servidor invalida la sesión (ej. cambiaron la contraseña desde la web)
	falso.mu.Lock()
	falso.tokenOK = "otro"
	falso.mu.Unlock()
	pedir(t, h, http.MethodPost, "/api/ventas", "", venta("kaiak", 1, pago("QR", "59200")))
	if err := app.Sincro.Sincronizar(ctx); !errors.Is(err, application.ErrSesionVencida) {
		t.Fatalf("sesión vencida: %v", err)
	}
	if r := pedir(t, h, http.MethodGet, "/api/escritorio/estado", "", nil); r.body["necesitaIngresar"] != true || r.body["pendientes"] != 1.0 {
		t.Errorf("estado: %v", r.body)
	}
}

func TestEscritorioServidorInvalido(t *testing.T) {
	_, err := escritorio.Abrir(context.Background(), escritorio.Config{Servidor: "magui.com", Datos: filepath.Join(t.TempDir(), "e.db")})
	if err == nil {
		t.Error("una dirección sin http(s) debe fallar")
	}
}
