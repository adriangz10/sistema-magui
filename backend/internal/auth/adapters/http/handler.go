// Package http expone el acceso por HTTP y provee el middleware que protege el resto de la API.
package http

import (
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"magui/internal/auth/application"
	"magui/internal/auth/domain"
	"magui/internal/platform/httpx"
)

// Handler implementa httpserver.Module (rutas protegidas) y httpserver.PublicModule (login).
type Handler struct {
	svc *application.Servicio
	log *slog.Logger
}

func NuevoHandler(svc *application.Servicio, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// PublicRoutes: lo único accesible sin sesión.
func (h *Handler) PublicRoutes(r chi.Router) {
	r.Post("/auth/login", h.login)
}

// Routes: requieren sesión.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/auth/sesion", h.sesion)
	r.Post("/auth/cambiar-clave", h.cambiarClave)
}

type loginReq struct {
	Clave string `json:"clave"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	ses, err := h.svc.IniciarSesion(r.Context(), origen(r), req.Clave)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ses)
}

func (h *Handler) sesion(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]bool{"activa": true})
}

type cambiarClaveReq struct {
	Actual string `json:"actual"`
	Nueva  string `json:"nueva"`
}

func (h *Handler) cambiarClave(w http.ResponseWriter, r *http.Request) {
	var req cambiarClaveReq
	if err := httpx.LeerJSON(w, r, &req); err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	ses, err := h.svc.CambiarClave(r.Context(), req.Actual, req.Nueva)
	if err != nil {
		httpx.Error(w, r, h.log, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ses)
}

// RequerirSesion bloquea las solicitudes sin un token válido ("Authorization: Bearer <token>").
func (h *Handler) RequerirSesion(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || strings.TrimSpace(token) == "" {
			httpx.Error(w, r, h.log, domain.ErrSesionInvalida)
			return
		}
		if err := h.svc.ValidarSesion(r.Context(), strings.TrimSpace(token)); err != nil {
			httpx.Error(w, r, h.log, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// origen identifica al cliente para el limitador de intentos.
// Usa la IP de la conexión: no confía en X-Forwarded-For (se puede falsificar).
func origen(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
