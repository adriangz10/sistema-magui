// Package http expone la sincronización del escritorio.
//
//	GET  /sync/catalogo   copia completa del catálogo (productos con stock y componentes de combos)
//	POST /sync/ventas     {ventas: [{id, fecha, items: [{productoId, cantidad, precio}], pagos: [{metodo, monto}]}]}
//	                      -> {resultados: [{id, estado: registrada|ya_existia|rechazada, error?}]}
package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"magui/internal/platform/httpx"
	"magui/internal/sync/application"
)

type Handler struct {
	svc *application.Servicio
	log *slog.Logger
}

func NuevoHandler(svc *application.Servicio, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Routes(r chi.Router) {
	r.Route("/sync", func(r chi.Router) {
		r.Get("/catalogo", h.catalogo)
		r.Post("/ventas", h.ventas)
	})
}

func (h *Handler) catalogo(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.CopiaCatalogo(r.Context())
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) ventas(w http.ResponseWriter, r *http.Request) {
	var req application.EnvioVentas
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.RecibirVentas(r.Context(), req.Ventas)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
