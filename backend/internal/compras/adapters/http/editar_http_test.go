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
	"magui/internal/kernel/dinero"
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
func (r *repo) ActualizarItems(context.Context, []domain.Item) error             { return nil }
func (r *repo) CompraMasReciente(context.Context, string) (*application.CompraReciente, error) {
	return &application.CompraReciente{CompraID: r.c.ID, Costo: dinero.MustParse("10")}, nil
}

type cat struct{}

func (cat) Productos(_ context.Context, ids []string) (map[string]application.ProductoCompra, error) {
	out := map[string]application.ProductoCompra{}
	for _, id := range ids {
		out[id] = application.ProductoCompra{Nombre: "Kaiak", PrecioVenta: dinero.MustParse("62000")}
	}
	return out, nil
}
func (cat) ValidarEmpresa(context.Context, string) error { return nil }
func (cat) NombreEmpresa(context.Context, string) (string, error) {
	return "", errores.NoEncontrado("empresa_no_encontrada", "x")
}
func (cat) RegistrarCostos(context.Context, []application.CostoProducto, string) error { return nil }

type inv struct{}

func (inv) Ingresar(context.Context, application.IngresoCompra) error { return nil }

// El lote del renglón i1 tiene stock 1 (de 4): bajar por debajo de 3 unidades está bloqueado.
func (inv) LotesDeRenglones(_ context.Context, ids []string, _ bool) (map[string]domain.LoteRenglon, error) {
	out := map[string]domain.LoteRenglon{}
	for _, id := range ids {
		out[id] = domain.LoteRenglon{LoteID: "l-" + id, Stock: 1}
	}
	return out, nil
}
func (inv) AjustarLoteDeRenglon(context.Context, application.AjusteRenglon) error { return nil }

func servidor() *httptest.Server {
	ahora := time.Date(2026, 10, 27, 16, 0, 0, 0, time.UTC)
	r := &repo{c: domain.Compra{ID: "c1", Fecha: time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC), Version: 1,
		Items: []domain.Item{{ID: "i1", ProductoID: "p1", Cantidad: 4, CostoUnitario: dinero.MustParse("100")}}}}
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
		{"items vacío", `{"fecha":"2026-10-19","version":1,"items":[]}`, "datos_invalidos", 422},
		{"items sin cambios", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100","precioVentaNuevo":null,"vencimiento":null}]}`, "", 200},
		{"subir cantidad", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":6,"costoUnitario":"100"}]}`, `"version":2`, 200},
		{"vencimiento null", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100","vencimiento":null}]}`, "", 200},
		{"consumida", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":2,"costoUnitario":"100"}]}`, "compra_consumida", 409},
		{"campo del renglón", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":0,"costoUnitario":"100"}]}`, "items[0].cantidad", 422},
		{"producto distinto", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p2","cantidad":4,"costoUnitario":"100"}]}`, "items[0].productoId", 422},
		{"renglón nuevo", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100"},{"productoId":"p1","cantidad":1,"costoUnitario":"1"}]}`, "renglones_nuevos_no_disponibles", 422},
		{"un solo renglón", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100"}]}`, "", 200},
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
