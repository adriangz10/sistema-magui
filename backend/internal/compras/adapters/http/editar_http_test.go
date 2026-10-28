package http_test

import (
	"context"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	comhttp "magui/internal/compras/adapters/http"
	"magui/internal/compras/application"
	"magui/internal/compras/domain"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/id"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/kernel/tx"
)

type repo struct{ c domain.Compra }

func (r *repo) Guardar(context.Context, domain.Compra) error { return nil }
func (r *repo) Obtener(context.Context, string) (application.CompraDetalle, error) {
	return application.CompraDetalle{CompraResumen: application.CompraResumen{ID: r.c.ID, Version: r.c.Version}, Items: []application.ItemDTO{}}, nil
}
func (r *repo) Listar(context.Context, application.Filtro, paginacion.Solicitud) ([]application.CompraResumen, int, error) {
	return nil, 0, nil
}
func (r *repo) ObtenerParaEditar(context.Context, string) (domain.Compra, error) { return r.c, nil }
func (r *repo) ActualizarCompra(_ context.Context, c domain.Compra, _ int) error { r.c = c; return nil }
func (r *repo) GuardarEdicion(context.Context, domain.Edicion) error             { return nil }
func (r *repo) Ediciones(context.Context, string) ([]domain.Edicion, error)      { return nil, nil }

type cat struct{}

func (cat) Productos(context.Context, []string) (map[string]application.ProductoCompra, error) {
	return nil, nil
}
func (cat) ValidarEmpresa(context.Context, string) error { return nil }
func (cat) NombreEmpresa(context.Context, string) (string, error) {
	return "", errores.NoEncontrado("empresa_no_encontrada", "x")
}
func (cat) RegistrarCostos(context.Context, []application.CostoProducto, string) error { return nil }

type inv struct{}

func (inv) Ingresar(context.Context, application.IngresoCompra) error { return nil }

func servidor() *httptest.Server {
	ahora := time.Date(2026, 10, 27, 16, 0, 0, 0, time.UTC)
	r := &repo{c: domain.Compra{ID: "c1", Fecha: time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC), Version: 1}}
	svc := application.NuevoServicio(application.Dependencias{Repo: r, Catalogo: cat{}, Inventario: inv{},
		Tx: tx.SinTransaccion{}, IDs: id.UUIDv7{}, Reloj: reloj.Fijo{T: ahora}})
	router := chi.NewRouter()
	comhttp.NuevoHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil))).Routes(router)
	return httptest.NewServer(router)
}

func put(t *testing.T, url, body string) (int, string) {
	t.Helper()
	req, _ := nethttp.NewRequest(nethttp.MethodPut, url+"/compras/c1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := nethttp.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

func TestPutCompra(t *testing.T) {
	casos := []struct {
		nombre, body, codigo string
		status               int
	}{
		{"ok", `{"fecha":"2026-10-19","notas":"x","version":1}`, "", 200},
		{"sin cambios", `{"fecha":"2026-10-20","version":1}`, "", 200},
		{"version vieja", `{"fecha":"2026-10-19","notas":"y","version":7}`, "version_desactualizada", 409},
		{"items", `{"fecha":"2026-10-19","version":1,"items":[]}`, "renglones_no_editables", 422},
		{"fecha vacía", `{"version":1}`, "datos_invalidos", 422},
	}
	for _, c := range casos {
		srv := servidor() // uno por caso: el fake guarda estado
		status, body := put(t, srv.URL, c.body)
		srv.Close()
		if status != c.status || !strings.Contains(body, c.codigo) {
			t.Errorf("%s: %d %s", c.nombre, status, body)
		}
	}
	srv := servidor()
	defer srv.Close()
	if _, b := put(t, srv.URL, `{"version":1}`); !strings.Contains(b, `"fecha"`) {
		t.Errorf("debería indicar el campo fecha: %s", b)
	}
}

// Reposición mínima de TestEditarCompraEstaEnrutado: el PUT no puede dar 404/405.
func TestEditarCompraEstaEnrutado(t *testing.T) {
	srv := servidor()
	defer srv.Close()
	if status, body := put(t, srv.URL, `{"fecha":"2026-10-19","version":1}`); status == 404 || status == 405 {
		t.Errorf("PUT /compras/{id} no está enrutado: %d %s", status, body)
	}
}
