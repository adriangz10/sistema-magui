package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	negociohttp "magui/internal/negocio/adapters/http"
	"magui/internal/negocio/application"
	"magui/internal/negocio/domain"
	"magui/internal/platform/httpserver"
)

type repoMem struct {
	n    domain.Negocio
	logo []byte
	tipo string
}

func (r *repoMem) Obtener(context.Context) (domain.Negocio, error) {
	n := r.n
	n.TieneLogo = r.logo != nil
	return n, nil
}
func (r *repoMem) Guardar(_ context.Context, n domain.Negocio) error { r.n = n; return nil }
func (r *repoMem) GuardarLogo(_ context.Context, d []byte, tipo string) error {
	r.logo, r.tipo = d, tipo
	r.n.LogoVersion++
	return nil
}
func (r *repoMem) Logo(context.Context) ([]byte, string, error) {
	if r.logo == nil {
		return nil, "", domain.ErrSinLogo
	}
	return r.logo, r.tipo, nil
}

const tokenValido = "ok"

// Router real con una "sesión" simple: Authorization: Bearer ok.
func router(repo *repoMem) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+tokenValido {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	h := negociohttp.NuevoHandler(application.NuevoServicio(repo), log)
	return httpserver.NewRouter(log, httpserver.Options{Autenticacion: auth}, h)
}

func pedir(t *testing.T, h http.Handler, metodo, ruta string, cuerpo []byte, conSesion bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(metodo, ruta, bytes.NewReader(cuerpo))
	if conSesion {
		req.Header.Set("Authorization", "Bearer "+tokenValido)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

var png1x1 = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func TestNegocioHTTP(t *testing.T) {
	repo := &repoMem{n: domain.Negocio{Nombre: "Magui", Ticket: domain.Ticket{AnchoPapel: 80, MostrarLogo: true}}}
	h := router(repo)

	// La marca y el logo se ven sin sesión (pantalla de ingreso); el resto no.
	if rec := pedir(t, h, "GET", "/api/negocio/marca", nil, false); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"nombre":"Magui"`) {
		t.Errorf("marca: %d %s", rec.Code, rec.Body)
	}
	if rec := pedir(t, h, "GET", "/api/negocio/logo", nil, false); rec.Code != 404 {
		t.Errorf("logo inexistente: %d", rec.Code)
	}
	for _, r := range [][2]string{{"GET", "/api/negocio"}, {"PUT", "/api/negocio"}, {"PUT", "/api/negocio/logo"}, {"DELETE", "/api/negocio/logo"}} {
		if rec := pedir(t, h, r[0], r[1], nil, false); rec.Code != 401 {
			t.Errorf("%s %s sin sesión: %d", r[0], r[1], rec.Code)
		}
	}

	cuerpo := `{"nombre":"Perfumería Magui","ticket":{"habilitado":true,"imprimirAlCobrar":true,"anchoPapel":58,"mostrarLogo":false,"datos":"Tel 123","pie":null}}`
	rec := pedir(t, h, "PUT", "/api/negocio", []byte(cuerpo), true)
	var res application.NegocioDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &res); rec.Code != 200 || err != nil {
		t.Fatalf("guardar: %d %s", rec.Code, rec.Body)
	}
	if res.Nombre != "Perfumería Magui" || !res.Ticket.ImprimirAlCobrar || res.Ticket.AnchoPapel != 58 || *res.Ticket.Datos != "Tel 123" {
		t.Errorf("guardado = %+v", res)
	}
	if rec := pedir(t, h, "PUT", "/api/negocio", []byte(`{"nombre":"","ticket":{"anchoPapel":80}}`), true); rec.Code != 422 {
		t.Errorf("nombre vacío: %d", rec.Code)
	}

	// Logo: se valida por contenido.
	if rec := pedir(t, h, "PUT", "/api/negocio/logo", []byte("<svg></svg>"), true); rec.Code != 422 || !strings.Contains(rec.Body.String(), "logo_tipo") {
		t.Errorf("svg: %d %s", rec.Code, rec.Body)
	}
	if rec := pedir(t, h, "PUT", "/api/negocio/logo", nil, true); rec.Code != 422 {
		t.Errorf("vacío: %d", rec.Code)
	}
	if rec := pedir(t, h, "PUT", "/api/negocio/logo", make([]byte, domain.TamanoMaximoLogo+10), true); rec.Code != 422 || !strings.Contains(rec.Body.String(), "logo_muy_grande") {
		t.Errorf("grande: %d %s", rec.Code, rec.Body)
	}
	if rec := pedir(t, h, "PUT", "/api/negocio/logo", png1x1, true); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"tieneLogo":true`) {
		t.Errorf("png: %d %s", rec.Code, rec.Body)
	}
	rec = pedir(t, h, "GET", "/api/negocio/logo?v=1", nil, false)
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" || !bytes.Equal(rec.Body.Bytes(), png1x1) {
		t.Errorf("logo: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec := pedir(t, h, "DELETE", "/api/negocio/logo", nil, true); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"tieneLogo":false`) {
		t.Errorf("quitar: %d %s", rec.Code, rec.Body)
	}
}
