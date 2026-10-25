// Package http expone el inventario por HTTP.
//
//	GET  /inventario/productos/{id}/stock   stock total y por lote
//	GET  /inventario/movimientos            ?productoId=&tipo=&desde=&hasta=&page=
//	POST /inventario/ajustes                {productoId, cantidad, motivo, loteId?, vencimiento?, costo?}
//	GET  /inventario/vencimientos           ?estado=vencidos|por_vencer|todos&page=
//	GET  /inventario/alertas
//	GET  /inventario/configuracion
//	PUT  /inventario/configuracion          {diasAlertaVencimiento}
package http

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"magui/internal/inventario/application"
	"magui/internal/inventario/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
	"magui/internal/platform/httpx"
)

type Handler struct {
	svc *application.Servicio
	log *slog.Logger
}

func NuevoHandler(svc *application.Servicio, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Routes(r chi.Router) {
	r.Route("/inventario", func(r chi.Router) {
		r.Get("/productos/{id}/stock", h.stock)
		r.Get("/movimientos", h.movimientos)
		r.Post("/ajustes", h.ajuste)
		r.Get("/vencimientos", h.vencimientos)
		r.Get("/alertas", h.alertas)
		r.Get("/configuracion", h.configuracion)
		r.Put("/configuracion", h.guardarConfiguracion)
	})
}

func (h *Handler) responder(w http.ResponseWriter, r *http.Request, v any, err error) {
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}

func (h *Handler) stock(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.StockDeProducto(r.Context(), chi.URLParam(r, "id"))
	h.responder(w, r, res, err)
}

func (h *Handler) movimientos(w http.ResponseWriter, r *http.Request) {
	desde, hasta, err := httpx.RangoFechas(r)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	f := application.FiltroMovimientos{
		ProductoID: r.URL.Query().Get("productoId"),
		Tipo:       domain.TipoMovimiento(strings.ToUpper(r.URL.Query().Get("tipo"))),
		Desde:      desde, Hasta: hasta,
	}
	res, err := h.svc.ListarMovimientos(r.Context(), f, httpx.Paginacion(r))
	h.responder(w, r, res, err)
}

type ajusteReq struct {
	ProductoID  string         `json:"productoId"`
	Cantidad    int            `json:"cantidad"`
	Motivo      string         `json:"motivo"`
	LoteID      string         `json:"loteId"`
	Vencimiento *fecha.Fecha   `json:"vencimiento"`
	Costo       *dinero.Dinero `json:"costo"`
}

func (h *Handler) ajuste(w http.ResponseWriter, r *http.Request) {
	var req ajusteReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.AjustarStock(r.Context(), application.ComandoAjuste{
		ProductoID: req.ProductoID, Cantidad: req.Cantidad, Motivo: req.Motivo,
		LoteID: req.LoteID, Vencimiento: req.Vencimiento, Costo: req.Costo,
	})
	h.responder(w, r, res, err)
}

func (h *Handler) vencimientos(w http.ResponseWriter, r *http.Request) {
	estado := application.EstadoVencimientos(strings.ToLower(r.URL.Query().Get("estado")))
	res, err := h.svc.ListarVencimientos(r.Context(), estado, httpx.Paginacion(r))
	h.responder(w, r, res, err)
}

func (h *Handler) alertas(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Alertas(r.Context())
	h.responder(w, r, res, err)
}

type configuracionDTO struct {
	DiasAlertaVencimiento int `json:"diasAlertaVencimiento"`
}

func (h *Handler) configuracion(w http.ResponseWriter, r *http.Request) {
	dias, err := h.svc.DiasAlerta(r.Context())
	h.responder(w, r, configuracionDTO{DiasAlertaVencimiento: dias}, err)
}

func (h *Handler) guardarConfiguracion(w http.ResponseWriter, r *http.Request) {
	var req configuracionDTO
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	err := h.svc.CambiarDiasAlerta(r.Context(), req.DiasAlertaVencimiento)
	h.responder(w, r, req, err)
}
