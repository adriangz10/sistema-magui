package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeModule struct{}

func (fakeModule) Routes(r chi.Router) {
	r.Get("/privado", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("secreto")) })
}

func (fakeModule) PublicRoutes(r chi.Router) {
	r.Get("/publico", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("hola")) })
}

// soloConLlave simula el middleware de sesión.
func soloConLlave(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newTestRouter(checks map[string]Check) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(log, Options{
		AllowedOrigins: []string{"http://localhost:4200"},
		Version:        "test",
		Checks:         checks,
		Autenticacion:  soloConLlave,
	}, fakeModule{})
}

func get(h http.Handler, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	rec := get(newTestRouter(map[string]Check{"db": func(context.Context) error { return nil }}), "/api/health", "")
	var body map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if rec.Code != 200 || body["status"] != "ok" || body["version"] != "test" {
		t.Fatalf("status=%d body=%v", rec.Code, body)
	}
}

func TestHealthDegradado(t *testing.T) {
	rec := get(newTestRouter(map[string]Check{"db": func(context.Context) error { return errors.New("caída") }}), "/api/health", "")
	var body map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if rec.Code != http.StatusServiceUnavailable || body["status"] != "degradado" {
		t.Fatalf("status=%d body=%v", rec.Code, body)
	}
}

func TestRutasPublicasYProtegidas(t *testing.T) {
	h := newTestRouter(nil)
	if rec := get(h, "/api/publico", ""); rec.Code != 200 || rec.Body.String() != "hola" {
		t.Errorf("pública: %d %q", rec.Code, rec.Body.String())
	}
	if rec := get(h, "/api/privado", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("protegida sin sesión: %d", rec.Code)
	}
	if rec := get(h, "/api/privado", "Bearer ok"); rec.Code != 200 || rec.Body.String() != "secreto" {
		t.Errorf("protegida con sesión: %d %q", rec.Code, rec.Body.String())
	}
}

func TestRutaInexistenteDevuelveJSON(t *testing.T) {
	rec := get(newTestRouter(nil), "/api/no-existe", "Bearer ok")
	if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("status=%d ct=%q", rec.Code, rec.Header().Get("Content-Type"))
	}
}

func TestCORSAllowsFrontend(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Header.Set("Origin", "http://localhost:4200")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	newTestRouter(nil).ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:4200" {
		t.Fatalf("Allow-Origin = %q", got)
	}
}
