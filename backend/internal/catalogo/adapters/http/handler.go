// Package http expone el catálogo por HTTP (todas las rutas requieren sesión).
//
//	GET    /empresas                          ?q=&activas=true&page=
//	POST   /empresas                          {nombre}
//	PUT    /empresas/{id}                     {nombre, activo, version}
//	GET    /categorias                        árbol completo
//	POST   /categorias                        {nombre, padreId}
//	PUT    /categorias/{id}                   {nombre, padreId, version}
//	DELETE /categorias/{id}
//	POST   /categorias/{id}/precios/vista-previa  {tipo, valor, redondeo, incluirSubcategorias}
//	POST   /categorias/{id}/precios               {..., motivo}
//	GET    /productos                         ?q=&categoriaId=&empresaId=&tipo=&estado=&stockBajo=&orden=&page=
//	POST   /productos                         producto simple o combo
//	GET    /productos/codigo/{codigo}         búsqueda exacta por SKU o código de barras
//	GET    /productos/{id}
//	PUT    /productos/{id}
//	PATCH  /productos/{id}/estado             {activo}
//	GET    /productos/{id}/precios            historial de precios
package http

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"magui/internal/catalogo/application"
	"magui/internal/catalogo/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/platform/httpx"
)

// Handler implementa httpserver.Module.
type Handler struct {
	svc *application.Servicio
	log *slog.Logger
}

func NuevoHandler(svc *application.Servicio, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Routes(r chi.Router) {
	r.Route("/empresas", func(r chi.Router) {
		r.Get("/", h.listarEmpresas)
		r.Post("/", h.crearEmpresa)
		r.Put("/{id}", h.actualizarEmpresa)
	})
	r.Route("/categorias", func(r chi.Router) {
		r.Get("/", h.arbolCategorias)
		r.Post("/", h.crearCategoria)
		r.Put("/{id}", h.actualizarCategoria)
		r.Delete("/{id}", h.eliminarCategoria)
		r.Post("/{id}/precios/vista-previa", h.ajustePrecios(false))
		r.Post("/{id}/precios", h.ajustePrecios(true))
	})
	r.Route("/productos", func(r chi.Router) {
		r.Get("/", h.listarProductos)
		r.Post("/", h.crearProducto)
		r.Get("/codigo/{codigo}", h.buscarPorCodigo)
		r.Get("/{id}", h.obtenerProducto)
		r.Put("/{id}", h.actualizarProducto)
		r.Patch("/{id}/estado", h.cambiarEstado)
		r.Get("/{id}/precios", h.historialPrecios)
	})
}

// responder escribe el resultado o el error de un caso de uso.
func responder[T any](h *Handler, w http.ResponseWriter, r *http.Request, status int, v T, err error) {
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, status, v)
}

// --- Empresas ---

type empresaReq struct {
	Nombre  string `json:"nombre"`
	Activo  *bool  `json:"activo"`
	Version int    `json:"version"`
}

func (h *Handler) listarEmpresas(w http.ResponseWriter, r *http.Request) {
	f := application.FiltroEmpresas{Texto: r.URL.Query().Get("q"), SoloActivas: r.URL.Query().Get("activas") == "true"}
	res, err := h.svc.ListarEmpresas(r.Context(), f, httpx.Paginacion(r))
	responder(h, w, r, http.StatusOK, res, err)
}

func (h *Handler) crearEmpresa(w http.ResponseWriter, r *http.Request) {
	var req empresaReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.CrearEmpresa(r.Context(), req.Nombre)
	responder(h, w, r, http.StatusCreated, res, err)
}

func (h *Handler) actualizarEmpresa(w http.ResponseWriter, r *http.Request) {
	var req empresaReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	activo := req.Activo == nil || *req.Activo
	res, err := h.svc.ActualizarEmpresa(r.Context(), chi.URLParam(r, "id"), req.Nombre, activo, req.Version)
	responder(h, w, r, http.StatusOK, res, err)
}

// --- Categorías ---

type categoriaReq struct {
	Nombre  string  `json:"nombre"`
	PadreID *string `json:"padreId"`
	Version int     `json:"version"`
}

func (h *Handler) arbolCategorias(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.ArbolCategorias(r.Context())
	responder(h, w, r, http.StatusOK, res, err)
}

func (h *Handler) crearCategoria(w http.ResponseWriter, r *http.Request) {
	var req categoriaReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.CrearCategoria(r.Context(), req.Nombre, req.PadreID)
	responder(h, w, r, http.StatusCreated, res, err)
}

func (h *Handler) actualizarCategoria(w http.ResponseWriter, r *http.Request) {
	var req categoriaReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.ActualizarCategoria(r.Context(), chi.URLParam(r, "id"), req.Nombre, req.PadreID, req.Version)
	responder(h, w, r, http.StatusOK, res, err)
}

func (h *Handler) eliminarCategoria(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.EliminarCategoria(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.SinContenido(w)
}

type ajusteReq struct {
	Tipo                 string         `json:"tipo"`
	Valor                string         `json:"valor"` // "10.5" (%) o "500" ($)
	Redondeo             *dinero.Dinero `json:"redondeo"`
	IncluirSubcategorias *bool          `json:"incluirSubcategorias"`
	Motivo               string         `json:"motivo"`
}

func (h *Handler) ajustePrecios(aplicar bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ajusteReq
		if err := httpx.LeerJSON(w, r, &req); err != nil {
			httpx.Error(w, r, h.log, err)
			return
		}
		cmd, err := req.comando(chi.URLParam(r, "id"))
		if err != nil {
			httpx.Error(w, r, h.log, err)
			return
		}
		var res application.ResultadoAjuste
		if aplicar {
			res, err = h.svc.AplicarAjuste(r.Context(), cmd)
		} else {
			res, err = h.svc.VistaPreviaAjuste(r.Context(), cmd)
		}
		responder(h, w, r, http.StatusOK, res, err)
	}
}

func (req ajusteReq) comando(categoriaID string) (application.ComandoAjuste, error) {
	// El valor admite hasta 2 decimales: se reutiliza el parser de dinero
	// (para porcentajes, "10.5" -> 1050 centésimas de punto).
	valor, err := dinero.Parse(req.Valor)
	if err != nil {
		var v errores.Validacion
		v.Agregar("valor", "Ingresá un número con hasta 2 decimales")
		return application.ComandoAjuste{}, v.Err()
	}
	aj := domain.AjustePrecio{Tipo: domain.TipoAjuste(strings.ToUpper(req.Tipo))}
	if aj.Tipo == domain.AjustePorcentaje {
		aj.Porcentaje = valor.Centavos()
	} else {
		aj.Monto = valor
	}
	if req.Redondeo != nil {
		aj.Redondeo = *req.Redondeo
	}
	return application.ComandoAjuste{
		CategoriaID: categoriaID, Ajuste: aj, Motivo: req.Motivo,
		IncluirSubcategorias: req.IncluirSubcategorias == nil || *req.IncluirSubcategorias,
	}, nil
}

// --- Productos ---

type comboItemReq struct {
	ProductoID string `json:"productoId"`
	Cantidad   int    `json:"cantidad"`
}

type productoReq struct {
	Tipo            string         `json:"tipo"`
	SKU             string         `json:"sku"`
	CodigoBarras    string         `json:"codigoBarras"`
	Nombre          string         `json:"nombre"`
	Descripcion     string         `json:"descripcion"`
	EmpresaID       string         `json:"empresaId"`
	PrecioVenta     dinero.Dinero  `json:"precioVenta"`
	Costo           *dinero.Dinero `json:"costo"`
	StockMinimo     int            `json:"stockMinimo"`
	MostrarEnTienda *bool          `json:"mostrarEnTienda"`
	CategoriaIDs    []string       `json:"categoriaIds"`
	Items           []comboItemReq `json:"items"`
	Version         int            `json:"version"`
}

func (req productoReq) datos() domain.DatosProducto {
	items := make([]domain.ComboItem, len(req.Items))
	for i, it := range req.Items {
		items[i] = domain.ComboItem{ProductoID: it.ProductoID, Cantidad: it.Cantidad}
	}
	return domain.DatosProducto{
		SKU: req.SKU, CodigoBarras: req.CodigoBarras, Nombre: req.Nombre, Descripcion: req.Descripcion,
		EmpresaID: req.EmpresaID, PrecioVenta: req.PrecioVenta, Costo: req.Costo, StockMinimo: req.StockMinimo,
		MostrarEnTienda: req.MostrarEnTienda == nil || *req.MostrarEnTienda,
		CategoriaIDs:    req.CategoriaIDs, Items: items,
	}
}

func (h *Handler) listarProductos(w http.ResponseWriter, r *http.Request) {
	c, err := consultaDesdeURL(r)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.ListarProductos(r.Context(), c, httpx.Paginacion(r))
	responder(h, w, r, http.StatusOK, res, err)
}

func consultaDesdeURL(r *http.Request) (application.ConsultaProductos, error) {
	q := r.URL.Query()
	c := application.ConsultaProductos{
		Texto: q.Get("q"), CategoriaID: q.Get("categoriaId"), EmpresaID: q.Get("empresaId"),
		Tipo:      domain.Tipo(strings.ToUpper(q.Get("tipo"))),
		Estado:    application.Estado(strings.ToLower(q.Get("estado"))),
		Orden:     application.Orden(strings.ToLower(q.Get("orden"))),
		StockBajo: q.Get("stockBajo") == "true",
	}
	var v errores.Validacion
	v.Si(c.Tipo != "" && c.Tipo != domain.TipoSimple && c.Tipo != domain.TipoCombo, "tipo", "Use SIMPLE o COMBO")
	switch c.Estado {
	case "", application.EstadoActivos, application.EstadoInactivos, application.EstadoTodos:
	default:
		v.Agregar("estado", "Use activos, inactivos o todos")
	}
	switch c.Orden {
	case "", application.OrdenNombre, application.OrdenRecientes, application.OrdenPrecio, application.OrdenStock:
	default:
		v.Agregar("orden", "Use nombre, recientes, precio o stock")
	}
	return c, v.Err()
}

func (h *Handler) crearProducto(w http.ResponseWriter, r *http.Request) {
	var req productoReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	tipo := domain.Tipo(strings.ToUpper(req.Tipo))
	if tipo == "" {
		tipo = domain.TipoSimple
	}
	res, err := h.svc.CrearProducto(r.Context(), application.ComandoProducto{Tipo: tipo, Datos: req.datos()})
	responder(h, w, r, http.StatusCreated, res, err)
}

func (h *Handler) obtenerProducto(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.ObtenerProducto(r.Context(), chi.URLParam(r, "id"))
	responder(h, w, r, http.StatusOK, res, err)
}

func (h *Handler) buscarPorCodigo(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.BuscarPorCodigo(r.Context(), chi.URLParam(r, "codigo"))
	responder(h, w, r, http.StatusOK, res, err)
}

func (h *Handler) actualizarProducto(w http.ResponseWriter, r *http.Request) {
	var req productoReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.ActualizarProducto(r.Context(), chi.URLParam(r, "id"), req.datos(), req.Version)
	responder(h, w, r, http.StatusOK, res, err)
}

type estadoReq struct {
	Activo bool `json:"activo"`
}

func (h *Handler) cambiarEstado(w http.ResponseWriter, r *http.Request) {
	var req estadoReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.CambiarEstadoProducto(r.Context(), chi.URLParam(r, "id"), req.Activo)
	responder(h, w, r, http.StatusOK, res, err)
}

func (h *Handler) historialPrecios(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.HistorialPrecios(r.Context(), chi.URLParam(r, "id"), httpx.Paginacion(r))
	responder(h, w, r, http.StatusOK, res, err)
}
