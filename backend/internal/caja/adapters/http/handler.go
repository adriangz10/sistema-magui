// Package http expone los gastos de caja por HTTP.
//
//	GET  /caja/gastos              ?desde=&hasta=&categoria=&incluirAnulados=true&page=
//	POST /caja/gastos              {fecha?, monto, motivo, categoria?, metodo?}
//	POST /caja/gastos/{id}/anular  {motivo}
//	GET  /caja/categorias
//
// Apertura y cierre de caja (US-002):
//
//	GET   /caja/sesiones                 ?page=
//	GET   /caja/sesiones/actual          caja abierta con arqueo en vivo, o null
//	GET   /caja/sesiones/{id}            detalle + historial de ediciones
//	POST  /caja/sesiones                 {montoInicial, notas?}
//	POST  /caja/sesiones/{id}/cerrar     {version, contado: {EFECTIVO, TRANSFERENCIA, QR, TARJETA}, notas?}
//	PATCH /caja/sesiones/{id}            {version, motivo, montoInicial, contado?, notas?}
package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"magui/internal/caja/application"
	"magui/internal/caja/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/platform/httpx"
)

type Handler struct {
	svc      *application.Servicio
	sesiones *application.ServicioSesiones
	log      *slog.Logger
}

func NuevoHandler(svc *application.Servicio, sesiones *application.ServicioSesiones, log *slog.Logger) *Handler {
	return &Handler{svc: svc, sesiones: sesiones, log: log}
}

func (h *Handler) Routes(r chi.Router) {
	r.Route("/caja", func(r chi.Router) {
		r.Get("/gastos", h.listar)
		r.Post("/gastos", h.registrar)
		r.Post("/gastos/{id}/anular", h.anular)
		r.Get("/categorias", h.categorias)
		r.Get("/sesiones", h.listarSesiones)
		r.Post("/sesiones", h.abrir)
		r.Get("/sesiones/actual", h.actual)
		r.Get("/sesiones/{id}", h.detalle)
		r.Post("/sesiones/{id}/cerrar", h.cerrar)
		r.Patch("/sesiones/{id}", h.editar)
	})
}

func (h *Handler) responder(w http.ResponseWriter, r *http.Request, status int, v any, err error) {
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, status, v)
}

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) {
	desde, hasta, err := httpx.RangoFechas(r)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	q := r.URL.Query()
	res, err := h.svc.ListarGastos(r.Context(), application.Filtro{Desde: desde, Hasta: hasta,
		Categoria: q.Get("categoria"), IncluirAnulados: q.Get("incluirAnulados") == "true"}, httpx.Paginacion(r))
	h.responder(w, r, http.StatusOK, res, err)
}

type gastoReq struct {
	Fecha     string        `json:"fecha"`
	Monto     dinero.Dinero `json:"monto"`
	Motivo    string        `json:"motivo"`
	Categoria string        `json:"categoria"`
	Metodo    string        `json:"metodo"`
}

func (h *Handler) registrar(w http.ResponseWriter, r *http.Request) {
	var req gastoReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	f, err := httpx.FechaOInstante("fecha", req.Fecha)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	metodo, ok := domain.ParseMetodo(req.Metodo)
	if !ok {
		var v errores.Validacion
		v.Agregar("metodo", "Use EFECTIVO, TRANSFERENCIA, QR o TARJETA")
		httpx.Error(w, r, h.log, v.Err())
		return
	}
	res, err := h.svc.RegistrarGasto(r.Context(), application.ComandoGasto{
		Fecha: f, Monto: req.Monto, Motivo: req.Motivo, Categoria: req.Categoria, Metodo: metodo})
	h.responder(w, r, http.StatusCreated, res, err)
}

type anularReq struct {
	Motivo string `json:"motivo"`
}

func (h *Handler) anular(w http.ResponseWriter, r *http.Request) {
	var req anularReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.AnularGasto(r.Context(), chi.URLParam(r, "id"), req.Motivo)
	h.responder(w, r, http.StatusOK, res, err)
}

func (h *Handler) categorias(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Categorias(r.Context())
	h.responder(w, r, http.StatusOK, res, err)
}
