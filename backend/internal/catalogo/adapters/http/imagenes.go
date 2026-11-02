package http

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"magui/internal/catalogo/domain"
	"magui/internal/platform/httpx"
)

// PublicRoutes: la foto de un producto se pide sin sesión (US-004).
func (h *Handler) PublicRoutes(r chi.Router) {
	r.Get("/imagenes/productos/{id}", h.imagen)
}

func (h *Handler) imagen(w http.ResponseWriter, r *http.Request) {
	datos, tipo, err := h.svc.Imagen(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	w.Header().Set("Content-Type", tipo)
	if r.URL.Query().Get("v") != "" {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // la versión cambia con cada foto
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	_, _ = w.Write(datos)
}

func (h *Handler) subirImagen(w http.ResponseWriter, r *http.Request) {
	datos, err := io.ReadAll(http.MaxBytesReader(w, r.Body, domain.TamanoMaximoImagen+1))
	if _, grande := errors.AsType[*http.MaxBytesError](err); grande {
		httpx.Error(w, r, h.log, domain.ErrImagenGrande)
		return
	}
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	if datos == nil {
		datos = []byte{} // cuerpo vacío: lo rechaza la validación (nil significaría "quitar")
	}
	res, err := h.svc.CambiarImagen(r.Context(), chi.URLParam(r, "id"), datos)
	responder(h, w, r, http.StatusOK, res, err)
}

func (h *Handler) quitarImagen(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.CambiarImagen(r.Context(), chi.URLParam(r, "id"), nil)
	responder(h, w, r, http.StatusOK, res, err)
}
