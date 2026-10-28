package http

import (
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// US-001a: PUT /compras/{id} tiene que estar enrutado. Con un cuerpo que no es JSON el handler
// responde 422 antes de tocar el servicio, así que alcanza con un Handler sin servicio.
func TestRutaPutComprasRegistrada(t *testing.T) {
	r := chi.NewRouter()
	NuevoHandler(nil, slog.New(slog.NewTextHandler(io.Discard, nil))).Routes(r)

	req := httptest.NewRequest(nethttp.MethodPut, "/compras/abc", strings.NewReader("no es json"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code == nethttp.StatusMethodNotAllowed || rec.Code == nethttp.StatusNotFound {
		t.Fatalf("PUT /compras/{id} no está registrado en Routes: HTTP %d", rec.Code)
	}
	if rec.Code != nethttp.StatusUnprocessableEntity {
		t.Errorf("esperaba 422 json_invalido, obtuve %d: %s", rec.Code, rec.Body.String())
	}
}
