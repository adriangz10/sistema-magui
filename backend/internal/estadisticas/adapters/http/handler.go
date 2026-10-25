// Package http expone las estadísticas por HTTP. Todas aceptan
// ?periodo=dia|semana|mes|anio|rango (por defecto dia) y, para rango, &desde=AAAA-MM-DD&hasta=AAAA-MM-DD.
//
//	GET /estadisticas/resumen
//	GET /estadisticas/ventas-por-dia
//	GET /estadisticas/ventas-por-hora
//	GET /estadisticas/mas-vendidos     &limite=10&orden=unidades|importe
package http

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"magui/internal/estadisticas/application"
	"magui/internal/estadisticas/domain"
	"magui/internal/kernel/errores"
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
	r.Route("/estadisticas", func(r chi.Router) {
		r.Get("/resumen", h.resumen)
		r.Get("/ventas-por-dia", h.porDia)
		r.Get("/ventas-por-hora", h.porHora)
		r.Get("/mas-vendidos", h.masVendidos)
	})
}

func consulta(r *http.Request) (application.ConsultaPeriodo, error) {
	q := r.URL.Query()
	c := application.ConsultaPeriodo{Tipo: domain.TipoPeriodo(strings.ToLower(q.Get("periodo")))}
	var v errores.Validacion
	for campo, dst := range map[string]**fecha.Fecha{"desde": &c.Desde, "hasta": &c.Hasta} {
		if s := q.Get(campo); s != "" {
			f, err := fecha.Parse(s)
			if err != nil {
				v.Agregar(campo, "Use el formato AAAA-MM-DD")
				continue
			}
			*dst = &f
		}
	}
	return c, v.Err()
}

func (h *Handler) atender(w http.ResponseWriter, r *http.Request, fn func(application.ConsultaPeriodo) (any, error)) {
	c, err := consulta(r)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := fn(c)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) resumen(w http.ResponseWriter, r *http.Request) {
	h.atender(w, r, func(c application.ConsultaPeriodo) (any, error) { return h.svc.Resumen(r.Context(), c) })
}

func (h *Handler) porDia(w http.ResponseWriter, r *http.Request) {
	h.atender(w, r, func(c application.ConsultaPeriodo) (any, error) { return h.svc.VentasPorDia(r.Context(), c) })
}

func (h *Handler) porHora(w http.ResponseWriter, r *http.Request) {
	h.atender(w, r, func(c application.ConsultaPeriodo) (any, error) { return h.svc.VentasPorHora(r.Context(), c) })
}

func (h *Handler) masVendidos(w http.ResponseWriter, r *http.Request) {
	limite, _ := strconv.Atoi(r.URL.Query().Get("limite"))
	orden := application.OrdenRanking(strings.ToLower(r.URL.Query().Get("orden")))
	h.atender(w, r, func(c application.ConsultaPeriodo) (any, error) {
		return h.svc.MasVendidos(r.Context(), c, limite, orden)
	})
}
