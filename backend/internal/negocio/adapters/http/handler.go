// Package http expone los datos del negocio (US-003).
//
// Públicas (se ven en la pantalla de ingreso):
//
//	GET    /negocio/marca     {nombre, tieneLogo, logoVersion}
//	GET    /negocio/logo      la imagen (?v=logoVersion para la caché)
//
// Con sesión:
//
//	GET    /negocio           marca + configuración del ticket
//	PUT    /negocio           {nombre, ticket: {habilitado, imprimirAlCobrar, anchoPapel, mostrarLogo, datos, pie}}
//	PUT    /negocio/logo      cuerpo = la imagen (PNG, JPG o WEBP, máximo 512 KB)
//	DELETE /negocio/logo
package http

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"magui/internal/negocio/application"
	"magui/internal/negocio/domain"
	"magui/internal/platform/httpx"
)

type Handler struct {
	svc *application.Servicio
	log *slog.Logger
}

func NuevoHandler(svc *application.Servicio, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) PublicRoutes(r chi.Router) {
	r.Get("/negocio/marca", h.marca)
	r.Get("/negocio/logo", h.logo)
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/negocio", h.obtener)
	r.Put("/negocio", h.guardar)
	r.Put("/negocio/logo", h.subirLogo)
	r.Delete("/negocio/logo", h.quitarLogo)
}

func (h *Handler) responder(w http.ResponseWriter, r *http.Request, v any, err error) {
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}

func (h *Handler) marca(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Marca(r.Context())
	h.responder(w, r, res, err)
}

func (h *Handler) logo(w http.ResponseWriter, r *http.Request) {
	datos, tipo, err := h.svc.Logo(r.Context())
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	w.Header().Set("Content-Type", tipo)
	if r.URL.Query().Get("v") != "" {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // la versión cambia con cada logo
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	_, _ = w.Write(datos)
}

func (h *Handler) obtener(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Obtener(r.Context())
	h.responder(w, r, res, err)
}

type guardarReq struct {
	Nombre string                `json:"nombre"`
	Ticket application.TicketDTO `json:"ticket"`
}

func (h *Handler) guardar(w http.ResponseWriter, r *http.Request) {
	var req guardarReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	res, err := h.svc.Guardar(r.Context(), application.ComandoNegocio{Nombre: req.Nombre, Ticket: req.Ticket})
	h.responder(w, r, res, err)
}

func (h *Handler) subirLogo(w http.ResponseWriter, r *http.Request) {
	datos, err := io.ReadAll(http.MaxBytesReader(w, r.Body, domain.TamanoMaximoLogo+1))
	if _, grande := errors.AsType[*http.MaxBytesError](err); grande {
		httpx.Error(w, r, h.log, domain.ErrLogoGrande)
		return
	}
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	if datos == nil {
		datos = []byte{} // cuerpo vacío: lo rechaza la validación (nil significaría "quitar")
	}
	res, err := h.svc.CambiarLogo(r.Context(), datos)
	h.responder(w, r, res, err)
}

func (h *Handler) quitarLogo(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.CambiarLogo(r.Context(), nil)
	h.responder(w, r, res, err)
}
