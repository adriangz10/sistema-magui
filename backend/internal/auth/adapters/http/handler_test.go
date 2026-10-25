package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"magui/internal/auth/adapters/seguridad"
	"magui/internal/auth/application"
	"magui/internal/auth/domain"
	"magui/internal/kernel/reloj"
	"magui/internal/platform/httpserver"
)

type repoMem struct{ cred *domain.Credencial }

func (r *repoMem) Obtener(context.Context) (domain.Credencial, bool, error) {
	if r.cred == nil {
		return domain.Credencial{}, false, nil
	}
	return *r.cred, true, nil
}
func (r *repoMem) Guardar(_ context.Context, c domain.Credencial) error { r.cred = &c; return nil }

type privado struct{}

func (privado) Routes(r chi.Router) {
	r.Get("/privado", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
}

func servidor(t *testing.T) http.Handler {
	t.Helper()
	tokens, _ := seguridad.NuevoJWT(strings.Repeat("s", 32), time.Hour)
	svc := application.NuevoServicio(&repoMem{}, seguridad.Bcrypt{Costo: 4}, tokens,
		seguridad.NuevoLimitador(5, 15*time.Minute), reloj.Sistema{})
	if err := svc.EstablecerClave(context.Background(), "secreta123"); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NuevoHandler(svc, log)
	return httpserver.NewRouter(log, httpserver.Options{Autenticacion: h.RequerirSesion}, h, privado{})
}

func pedir(h http.Handler, metodo, ruta, token, cuerpo string) (*httptest.ResponseRecorder, map[string]any) {
	req := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	req.RemoteAddr = "10.0.0.1:5555"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec, body
}

func codigoError(body map[string]any) string {
	e, _ := body["error"].(map[string]any)
	c, _ := e["code"].(string)
	return c
}

func TestLoginYRutaProtegida(t *testing.T) {
	h := servidor(t)

	if rec, _ := pedir(h, "GET", "/api/privado", "", ""); rec.Code != 401 {
		t.Fatalf("sin token: %d", rec.Code)
	}
	rec, body := pedir(h, "POST", "/api/auth/login", "", `{"clave":"secreta123"}`)
	token, _ := body["token"].(string)
	if rec.Code != 200 || token == "" || body["expiraEn"] == nil {
		t.Fatalf("login: %d %v", rec.Code, body)
	}
	if rec, _ := pedir(h, "GET", "/api/privado", token, ""); rec.Code != 200 {
		t.Fatalf("con token: %d", rec.Code)
	}
	if rec, _ := pedir(h, "GET", "/api/auth/sesion", token, ""); rec.Code != 200 {
		t.Fatalf("sesión: %d", rec.Code)
	}
	if rec, body := pedir(h, "GET", "/api/privado", "token-falso", ""); rec.Code != 401 || codigoError(body) != "sesion_invalida" {
		t.Fatalf("token falso: %d %v", rec.Code, body)
	}
}

func TestLoginIncorrectoYBloqueo(t *testing.T) {
	h := servidor(t)
	for i := range 5 {
		rec, body := pedir(h, "POST", "/api/auth/login", "", `{"clave":"mal"}`)
		if rec.Code != 401 || codigoError(body) != "clave_incorrecta" {
			t.Fatalf("intento %d: %d %v", i+1, rec.Code, body)
		}
	}
	rec, body := pedir(h, "POST", "/api/auth/login", "", `{"clave":"secreta123"}`)
	if rec.Code != 429 || codigoError(body) != "demasiados_intentos" {
		t.Fatalf("debería estar bloqueado: %d %v", rec.Code, body)
	}
}

func TestCambiarClaveCierraOtrasSesiones(t *testing.T) {
	h := servidor(t)
	_, body := pedir(h, "POST", "/api/auth/login", "", `{"clave":"secreta123"}`)
	vieja := body["token"].(string)

	rec, body := pedir(h, "POST", "/api/auth/cambiar-clave", vieja, `{"actual":"secreta123","nueva":"nuevaClave789"}`)
	if rec.Code != 200 {
		t.Fatalf("cambiar: %d %v", rec.Code, body)
	}
	nueva := body["token"].(string)

	if rec, _ := pedir(h, "GET", "/api/privado", vieja, ""); rec.Code != 401 {
		t.Errorf("token viejo debería quedar inválido: %d", rec.Code)
	}
	if rec, _ := pedir(h, "GET", "/api/privado", nueva, ""); rec.Code != 200 {
		t.Errorf("token nuevo: %d", rec.Code)
	}
	if rec, _ := pedir(h, "POST", "/api/auth/login", "", `{"clave":"nuevaClave789"}`); rec.Code != 200 {
		t.Errorf("login con clave nueva: %d", rec.Code)
	}
}

func TestCambiarClaveRequiereSesion(t *testing.T) {
	h := servidor(t)
	if rec, _ := pedir(h, "POST", "/api/auth/cambiar-clave", "", `{"actual":"secreta123","nueva":"x12345678"}`); rec.Code != 401 {
		t.Fatalf("sin sesión: %d", rec.Code)
	}
}
