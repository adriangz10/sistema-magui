// Package http expone las ventas por HTTP.
//
//	POST /ventas               {items: [{productoId, cantidad}], pagos: [{metodo, monto}], origen?, permitirSinStock?}
//	GET  /ventas               ?desde=&hasta=&estado=&metodo=&page=
//	GET  /ventas/{id}
//	POST /ventas/{id}/anular   {motivo}
package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/platform/httpx"
	"magui/internal/ventas/application"
	"magui/internal/ventas/domain"
)

type Handler struct {
	svc *application.Servicio
	log *slog.Logger
}

func NuevoHandler(svc *application.Servicio, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Routes(r chi.Router) {
	r.Route("/ventas", func(r chi.Router) {
		r.Get("/", h.listar)
		r.Post("/", h.registrar)
		r.Get("/{id}", h.obtener)
		r.Post("/{id}/anular", h.anular)
	})
}

func (h *Handler) responder(w http.ResponseWriter, r *http.Request, status int, v any, err error) {
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, status, v)
}

type ventaReq struct {
	Items []struct {
		ProductoID string `json:"productoId"`
		Cantidad   int    `json:"cantidad"`
	} `json:"items"`
	Pagos []struct {
		Metodo string        `json:"metodo"`
		Monto  dinero.Dinero `json:"monto"`
	} `json:"pagos"`
	Origen           string `json:"origen"`
	PermitirSinStock bool   `json:"permitirSinStock"`
}

func (req ventaReq) comando() (application.ComandoVenta, error) {
	var v errores.Validacion
	origen, ok := domain.ParseOrigen(req.Origen)
	v.Si(!ok, "origen", "Use WEB, MOVIL o DESKTOP")
	cmd := application.ComandoVenta{Origen: origen, PermitirSinStock: req.PermitirSinStock}
	for _, it := range req.Items {
		cmd.Items = append(cmd.Items, application.LineaCmd{ProductoID: it.ProductoID, Cantidad: it.Cantidad})
	}
	for i, p := range req.Pagos {
		m, ok := domain.ParseMetodo(p.Metodo)
		v.Si(!ok, fmt.Sprintf("pagos[%d].metodo", i), "Use EFECTIVO, TRANSFERENCIA, QR o TARJETA")
		cmd.Pagos = append(cmd.Pagos, domain.PagoIngresado{Metodo: m, Monto: p.Monto})
	}
	return cmd, v.Err()
}

func (h *Handler) registrar(w http.ResponseWriter, r *http.Request) {
	var req ventaReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	cmd, err := req.comando()
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.RegistrarVenta(r.Context(), cmd)
	h.responder(w, r, http.StatusCreated, res, err)
}

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) {
	desde, hasta, err := httpx.RangoFechas(r)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	f := application.Filtro{Desde: desde, Hasta: hasta}
	var v errores.Validacion
	if s := r.URL.Query().Get("estado"); s != "" {
		f.Estado = domain.Estado(strings.ToUpper(s))
		v.Si(f.Estado != domain.Confirmada && f.Estado != domain.Anulada, "estado", "Use CONFIRMADA o ANULADA")
	}
	if s := r.URL.Query().Get("metodo"); s != "" {
		m, ok := domain.ParseMetodo(s)
		v.Si(!ok, "metodo", "Use EFECTIVO, TRANSFERENCIA, QR o TARJETA")
		f.Metodo = m
	}
	if err := v.Err(); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.ListarVentas(r.Context(), f, httpx.Paginacion(r))
	h.responder(w, r, http.StatusOK, res, err)
}

func (h *Handler) obtener(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.ObtenerVenta(r.Context(), chi.URLParam(r, "id"))
	h.responder(w, r, http.StatusOK, res, err)
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
	res, err := h.svc.AnularVenta(r.Context(), chi.URLParam(r, "id"), req.Motivo)
	h.responder(w, r, http.StatusOK, res, err)
}
