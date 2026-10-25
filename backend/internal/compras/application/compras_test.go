package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"magui/internal/compras/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/kernel/tx"
)

type repoMem struct{ compras map[string]domain.Compra }

func (r *repoMem) Guardar(_ context.Context, c domain.Compra) error { r.compras[c.ID] = c; return nil }
func (r *repoMem) Obtener(_ context.Context, id string) (CompraDetalle, error) {
	c, ok := r.compras[id]
	if !ok {
		return CompraDetalle{}, domain.ErrCompraNoEncontrada
	}
	return CompraDetalle{CompraResumen: CompraResumen{ID: c.ID, Total: c.Total, Productos: len(c.Items), Unidades: c.Unidades()}}, nil
}
func (r *repoMem) Listar(context.Context, Filtro, paginacion.Solicitud) ([]CompraResumen, int, error) {
	return nil, len(r.compras), nil
}

type catalogoMem struct {
	productos map[string]ProductoCompra
	costos    []CostoProducto
	motivo    string
}

func (c *catalogoMem) Productos(_ context.Context, ids []string) (map[string]ProductoCompra, error) {
	out := map[string]ProductoCompra{}
	for _, id := range ids {
		if p, ok := c.productos[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}
func (c *catalogoMem) ValidarEmpresa(_ context.Context, id string) error {
	if id != "natura" {
		return errores.NoEncontrado("empresa_no_encontrada", "La empresa no existe")
	}
	return nil
}
func (c *catalogoMem) RegistrarCostos(_ context.Context, cs []CostoProducto, motivo string) error {
	c.costos, c.motivo = append(c.costos, cs...), motivo
	return nil
}

type inventarioMem struct{ ingresos []IngresoCompra }

func (i *inventarioMem) Ingresar(_ context.Context, in IngresoCompra) error {
	i.ingresos = append(i.ingresos, in)
	return nil
}

type idsSec struct{ n int }

func (g *idsSec) Nuevo() string { g.n++; return fmt.Sprintf("id-%d", g.n) }

var ahora = time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)

func entorno() (*Servicio, *repoMem, *catalogoMem, *inventarioMem) {
	repo := &repoMem{compras: map[string]domain.Compra{}}
	cat := &catalogoMem{productos: map[string]ProductoCompra{"kaiak": {Nombre: "Kaiak"}, "jabon": {Nombre: "Jabón"}, "combo": {Nombre: "Combo", Combo: true}}}
	inv := &inventarioMem{}
	s := NuevoServicio(Dependencias{Repo: repo, Catalogo: cat, Inventario: inv, Tx: tx.SinTransaccion{}, IDs: &idsSec{}, Reloj: reloj.Fijo{T: ahora}})
	return s, repo, cat, inv
}

func d(s string) dinero.Dinero { return dinero.MustParse(s) }

var ctx = context.Background()

func TestRegistrarCompra(t *testing.T) {
	s, repo, cat, inv := entorno()
	precio := d("65000")
	venc, _ := fecha.Parse("2027-03-01")
	det, err := s.RegistrarCompra(ctx, ComandoCompra{EmpresaID: "natura", Notas: "Pedido ciclo 15", Items: []ItemCmd{
		{ProductoID: "kaiak", Cantidad: 3, CostoUnitario: d("35520"), PrecioVentaNuevo: &precio, Vencimiento: &venc},
		{ProductoID: "jabon", Cantidad: 6, CostoUnitario: d("2000.50")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if det.Total.String() != "118563.00" || det.Unidades != 9 || det.Productos != 2 {
		t.Errorf("detalle = %+v", det)
	}
	c := repo.compras[det.ID]
	if !c.Fecha.Equal(ahora) || *c.EmpresaID != "natura" || *c.Notas != "Pedido ciclo 15" {
		t.Errorf("compra = %+v", c)
	}
	if len(inv.ingresos) != 2 || inv.ingresos[0].CompraItemID != c.Items[0].ID || inv.ingresos[0].Vencimiento.String() != "2027-03-01" {
		t.Errorf("ingresos = %+v", inv.ingresos)
	}
	if len(cat.costos) != 2 || cat.costos[0].PrecioVenta.String() != "65000.00" || cat.costos[1].PrecioVenta != nil {
		t.Errorf("costos = %+v", cat.costos)
	}
	if cat.motivo != "Compra del 2026-10-25" {
		t.Errorf("motivo = %q", cat.motivo)
	}
}

func TestRegistrarCompraValidaciones(t *testing.T) {
	s, repo, _, inv := entorno()
	futuro := ahora.Add(48 * time.Hour)
	cero := d("0")
	casos := map[string]ComandoCompra{
		"sin items":        {},
		"fecha futura":     {Fecha: &futuro, Items: []ItemCmd{{ProductoID: "kaiak", Cantidad: 1}}},
		"cantidad cero":    {Items: []ItemCmd{{ProductoID: "kaiak", Cantidad: 0}}},
		"costo negativo":   {Items: []ItemCmd{{ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("-1")}}},
		"precio cero":      {Items: []ItemCmd{{ProductoID: "kaiak", Cantidad: 1, PrecioVentaNuevo: &cero}}},
		"producto nuevo":   {Items: []ItemCmd{{ProductoID: "nope", Cantidad: 1}}},
		"combo":            {Items: []ItemCmd{{ProductoID: "combo", Cantidad: 1}}},
		"empresa inválida": {EmpresaID: "avon", Items: []ItemCmd{{ProductoID: "kaiak", Cantidad: 1}}},
	}
	for nombre, cmd := range casos {
		if _, err := s.RegistrarCompra(ctx, cmd); err == nil {
			t.Errorf("%s: debería fallar", nombre)
		}
	}
	if len(repo.compras) != 0 || len(inv.ingresos) != 0 {
		t.Error("ninguna compra inválida debería guardarse")
	}
	_, err := s.RegistrarCompra(ctx, ComandoCompra{Items: []ItemCmd{{ProductoID: "kaiak", Cantidad: 0}}})
	if e, _ := errores.Como(err); e == nil || e.Campos["items[0].cantidad"] == "" {
		t.Errorf("el error debería indicar el renglón: %v", err)
	}
	if _, err := s.ObtenerCompra(ctx, "nope"); !errors.Is(err, domain.ErrCompraNoEncontrada) {
		t.Errorf("compra inexistente: %v", err)
	}
}
