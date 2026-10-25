package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"magui/internal/kernel/errores"
)

var logMudo = slog.New(slog.NewTextHandler(io.Discard, nil))

func decodificarError(t *testing.T, rec *httptest.ResponseRecorder) detalleError {
	t.Helper()
	var c cuerpoError
	if err := json.NewDecoder(rec.Body).Decode(&c); err != nil {
		t.Fatal(err)
	}
	return c.Error
}

func TestErrorTraduceTipos(t *testing.T) {
	casos := []struct {
		err    error
		status int
		code   string
	}{
		{errores.Invalido("x", "m"), 422, "x"},
		{errores.NoEncontrado("producto_no_encontrado", "m"), 404, "producto_no_encontrado"},
		{errores.Conflicto("stock_insuficiente", "m"), 409, "stock_insuficiente"},
		{errores.NoAutorizado("no_autorizado", "m"), 401, "no_autorizado"},
		{errores.Demasiados("demasiados_intentos", "m"), 429, "demasiados_intentos"},
		{errors.New("se cayó la base"), 500, "error_interno"},
	}
	for _, c := range casos {
		rec := httptest.NewRecorder()
		Error(rec, httptest.NewRequest("GET", "/", nil), logMudo, c.err)
		if rec.Code != c.status {
			t.Errorf("%v: status %d, want %d", c.err, rec.Code, c.status)
		}
		if d := decodificarError(t, rec); d.Code != c.code || strings.Contains(d.Message, "base") {
			t.Errorf("%v: cuerpo %+v", c.err, d)
		}
	}
}

func TestErrorConCampos(t *testing.T) {
	var v errores.Validacion
	v.Agregar("precio", "Debe ser mayor a cero")
	rec := httptest.NewRecorder()
	Error(rec, httptest.NewRequest("GET", "/", nil), logMudo, v.Err())
	if d := decodificarError(t, rec); d.Fields["precio"] != "Debe ser mayor a cero" {
		t.Errorf("campos = %v", d.Fields)
	}
}

func TestLeerJSON(t *testing.T) {
	type in struct {
		Nombre string `json:"nombre"`
	}
	casos := map[string]string{
		`{"nombre":"ok"}`:           "",
		`{"nombre":"ok","extra":1}`: "json_invalido",
		``:                          "cuerpo_vacio",
		`{"nombre":`:                "json_invalido",
		`{"nombre":"a"}{"x":1}`:     "json_invalido",
		`{"nombre":"` + strings.Repeat("a", TamanoMaximoCuerpo) + `"}`: "cuerpo_muy_grande",
	}
	for body, wantCode := range casos {
		var dst in
		err := LeerJSON(httptest.NewRecorder(), httptest.NewRequest("POST", "/", strings.NewReader(body)), &dst)
		e, _ := errores.Como(err)
		got := ""
		if e != nil {
			got = e.Codigo
		}
		if got != wantCode {
			t.Errorf("cuerpo %.30q: código %q, want %q", body, got, wantCode)
		}
	}
}

func TestPaginacion(t *testing.T) {
	s := Paginacion(httptest.NewRequest(http.MethodGet, "/?page=3&size=100", nil))
	if s.Pagina != 3 || s.Tamano != 10 {
		t.Errorf("solicitud = %+v", s)
	}
}
