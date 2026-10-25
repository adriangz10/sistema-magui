// Package http expone las compras por HTTP.
//
//	GET  /compras        ?desde=&hasta=&empresaId=&page=
//	GET  /compras/{id}
//	POST /compras        {fecha?, empresaId?, notas?, items: [{productoId, cantidad, costoUnitario, precioVentaNuevo?, vencimiento?}]}
package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"magui/internal/compras/application"
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
	r.Route("/compras", func(r chi.Router) {
		r.Get("/", h.listar)
		r.Post("/", h.registrar)
		r.Get("/{id}", h.obtener)
	})
}

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) {
	desde, hasta, err := httpx.RangoFechas(r)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.ListarCompras(r.Context(),
		application.Filtro{Desde: desde, Hasta: hasta, EmpresaID: r.URL.Query().Get("empresaId")}, httpx.Paginacion(r))
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) obtener(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.ObtenerCompra(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

type itemReq struct {
	ProductoID       string         `json:"productoId"`
	Cantidad         int            `json:"cantidad"`
	CostoUnitario    dinero.Dinero  `json:"costoUnitario"`
	PrecioVentaNuevo *dinero.Dinero `json:"precioVentaNuevo"`
	Vencimiento      *fecha.Fecha   `json:"vencimiento"`
}

type compraReq struct {
	Fecha     string    `json:"fecha"` // AAAA-MM-DD o ISO; vacío = ahora
	EmpresaID string    `json:"empresaId"`
	Notas     string    `json:"notas"`
	Items     []itemReq `json:"items"`
}

func (h *Handler) registrar(w http.ResponseWriter, r *http.Request) {
	var req compraReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	f, err := httpx.FechaOInstante("fecha", req.Fecha)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	items := make([]application.ItemCmd, len(req.Items))
	for i, it := range req.Items {
		items[i] = application.ItemCmd{ProductoID: it.ProductoID, Cantidad: it.Cantidad, CostoUnitario: it.CostoUnitario,
			PrecioVentaNuevo: it.PrecioVentaNuevo, Vencimiento: it.Vencimiento}
	}
	res, err := h.svc.RegistrarCompra(r.Context(), application.ComandoCompra{
		Fecha: f, EmpresaID: req.EmpresaID, Notas: req.Notas, Items: items,
	})
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}
