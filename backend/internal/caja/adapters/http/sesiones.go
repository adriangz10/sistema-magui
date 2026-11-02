package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"magui/internal/caja/application"
	"magui/internal/caja/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/platform/httpx"
)

func (h *Handler) listarSesiones(w http.ResponseWriter, r *http.Request) {
	res, err := h.sesiones.Listar(r.Context(), httpx.Paginacion(r))
	h.responder(w, r, http.StatusOK, res, err)
}

func (h *Handler) actual(w http.ResponseWriter, r *http.Request) {
	res, err := h.sesiones.Actual(r.Context())
	h.responder(w, r, http.StatusOK, res, err)
}

func (h *Handler) detalle(w http.ResponseWriter, r *http.Request) {
	res, err := h.sesiones.Detalle(r.Context(), chi.URLParam(r, "id"))
	h.responder(w, r, http.StatusOK, res, err)
}

type abrirReq struct {
	MontoInicial *dinero.Dinero `json:"montoInicial"`
	Notas        string         `json:"notas"`
}

func (h *Handler) abrir(w http.ResponseWriter, r *http.Request) {
	var req abrirReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	if req.MontoInicial == nil {
		httpx.Error(w, r, h.log, falta("montoInicial", "Ingresá con cuánto efectivo abrís (puede ser 0)"))
		return
	}
	res, err := h.sesiones.Abrir(r.Context(), *req.MontoInicial, req.Notas)
	h.responder(w, r, http.StatusCreated, res, err)
}

// contadoReq: {"EFECTIVO": "1000.00", ...}. Los medios ausentes o null los marca el dominio como faltantes.
type contadoReq map[string]*dinero.Dinero

func (c contadoReq) aDominio() (map[domain.Metodo]dinero.Dinero, error) {
	out := make(map[domain.Metodo]dinero.Dinero, len(c))
	var v errores.Validacion
	for k, monto := range c {
		m, ok := domain.ParseMetodo(k)
		if !ok || k == "" {
			v.Agregar("contado."+k, "Medio de pago desconocido")
			continue
		}
		if monto != nil {
			out[m] = *monto
		}
	}
	return out, v.Err()
}

type cerrarReq struct {
	Version int        `json:"version"`
	Contado contadoReq `json:"contado"`
	Notas   string     `json:"notas"`
}

func (h *Handler) cerrar(w http.ResponseWriter, r *http.Request) {
	var req cerrarReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	contado, err := req.Contado.aDominio()
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.sesiones.Cerrar(r.Context(), chi.URLParam(r, "id"),
		application.ComandoCierre{Version: req.Version, Contado: contado, Notas: req.Notas})
	h.responder(w, r, http.StatusOK, res, err)
}

type editarReq struct {
	Version      int            `json:"version"`
	Motivo       string         `json:"motivo"`
	MontoInicial *dinero.Dinero `json:"montoInicial"`
	Contado      contadoReq     `json:"contado"`
	Notas        string         `json:"notas"`
}

func (h *Handler) editar(w http.ResponseWriter, r *http.Request) {
	var req editarReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	if req.MontoInicial == nil {
		httpx.Error(w, r, h.log, falta("montoInicial", "Ingresá el monto inicial"))
		return
	}
	contado, err := req.Contado.aDominio()
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.sesiones.Editar(r.Context(), chi.URLParam(r, "id"), application.ComandoEdicion{Version: req.Version,
		MontoInicial: *req.MontoInicial, Contado: contado, Notas: req.Notas, Motivo: req.Motivo})
	h.responder(w, r, http.StatusOK, res, err)
}

func falta(campo, mensaje string) error {
	var v errores.Validacion
	v.Agregar(campo, mensaje)
	return v.Err()
}
