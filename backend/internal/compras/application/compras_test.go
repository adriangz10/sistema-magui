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

type repoMem struct {
	compras       map[string]domain.Compra
	ediciones     []domain.Edicion
	falloEdicion  error
	escrituras    int
	itemsEscritos int
}

func (r *repoMem) Guardar(_ context.Context, c domain.Compra) error {
	if c.Version == 0 {
		c.Version = 1
	}
	r.compras[c.ID] = c
	return nil
}
func (r *repoMem) ObtenerParaEditar(_ context.Context, id string) (domain.Compra, error) {
	c, ok := r.compras[id]
	if !ok {
		return domain.Compra{}, domain.ErrCompraNoEncontrada
	}
	if c.EmpresaID != nil {
		c.Empresa = "Natura"
	}
	return c, nil
}
func (r *repoMem) ActualizarCompra(_ context.Context, c domain.Compra, v int) error {
	if r.compras[c.ID].Version != v {
		return domain.ErrVersionDesactualizada
	}
	r.escrituras++
	c.Items = r.compras[c.ID].Items // los renglones se graban con ActualizarItems
	r.compras[c.ID] = c
	return nil
}
func (r *repoMem) ActualizarItems(_ context.Context, items []domain.Item) error {
	for _, nuevo := range items {
		for id, c := range r.compras {
			for i, it := range c.Items {
				if it.ID == nuevo.ID {
					c.Items = append([]domain.Item(nil), c.Items...)
					c.Items[i] = nuevo
					r.compras[id] = c
					r.itemsEscritos++
				}
			}
		}
	}
	return nil
}

// CompraMasReciente: fecha, created_at e id descendentes; costo del último renglón del producto.
func (r *repoMem) CompraMasReciente(_ context.Context, productoID string) (*CompraReciente, error) {
	var mejor *domain.Compra
	for _, c := range r.compras {
		tiene := false
		for _, it := range c.Items {
			tiene = tiene || it.ProductoID == productoID
		}
		if !tiene {
			continue
		}
		dc := fecha.DelInstante(c.Fecha, reloj.ZonaArgentina)
		if mejor == nil {
			cc := c
			mejor = &cc
			continue
		}
		dm := fecha.DelInstante(mejor.Fecha, reloj.ZonaArgentina)
		if dc.Despues(dm) || dc.String() == dm.String() &&
			(c.CreatedAt.After(mejor.CreatedAt) || c.CreatedAt.Equal(mejor.CreatedAt) && c.ID > mejor.ID) {
			cc := c
			mejor = &cc
		}
	}
	if mejor == nil {
		return nil, nil
	}
	rec := &CompraReciente{CompraID: mejor.ID}
	for _, it := range mejor.Items {
		if it.ProductoID == productoID {
			rec.Costo = it.CostoUnitario
		}
	}
	return rec, nil
}
func (r *repoMem) GuardarEdicion(_ context.Context, e domain.Edicion) error {
	if r.falloEdicion != nil {
		return r.falloEdicion
	}
	r.ediciones = append(r.ediciones, e)
	return nil
}
func (r *repoMem) Ediciones(_ context.Context, id string) ([]domain.Edicion, error) {
	var out []domain.Edicion
	for i := len(r.ediciones) - 1; i >= 0; i-- {
		if r.ediciones[i].CompraID == id {
			out = append(out, r.ediciones[i])
		}
	}
	return out, nil
}
func (r *repoMem) Obtener(_ context.Context, id string) (CompraDetalle, error) {
	c, ok := r.compras[id]
	if !ok {
		return CompraDetalle{}, domain.ErrCompraNoEncontrada
	}
	det := CompraDetalle{CompraResumen: CompraResumen{ID: c.ID, Version: c.Version, Total: c.Total, Productos: len(c.Items), Unidades: c.Unidades()}, Items: []ItemDTO{}}
	for _, it := range c.Items {
		det.Items = append(det.Items, ItemDTO{ID: it.ID, ProductoID: it.ProductoID, Cantidad: it.Cantidad, CostoUnitario: it.CostoUnitario,
			PrecioVentaNuevo: it.PrecioVentaNuevo, Vencimiento: it.Vencimiento})
	}
	return det, nil
}
func (r *repoMem) Listar(context.Context, Filtro, paginacion.Solicitud) ([]CompraResumen, int, error) {
	return nil, len(r.compras), nil
}

type catalogoMem struct {
	consultas int
	productos map[string]ProductoCompra
	costos    []CostoProducto
	motivo    string
}

func (c *catalogoMem) Productos(_ context.Context, ids []string) (map[string]ProductoCompra, error) {
	c.consultas++
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
func (c *catalogoMem) NombreEmpresa(ctx context.Context, id string) (string, error) {
	if err := c.ValidarEmpresa(ctx, id); err != nil {
		return "", err
	}
	return "Natura", nil
}
func (c *catalogoMem) RegistrarCostos(_ context.Context, cs []CostoProducto, motivo string) error {
	c.costos, c.motivo = append(c.costos, cs...), motivo
	return nil
}

type inventarioMem struct {
	ingresos []IngresoCompra
	lotes    map[string]domain.LoteRenglon // por id de renglón
	ajustes  []AjusteRenglon
	bloqueos int
	falla    error
}

func (i *inventarioMem) Ingresar(_ context.Context, in IngresoCompra) error {
	i.ingresos = append(i.ingresos, in)
	if i.lotes == nil {
		i.lotes = map[string]domain.LoteRenglon{}
	}
	i.lotes[in.CompraItemID] = domain.LoteRenglon{LoteID: "lote-" + in.CompraItemID, Stock: in.Cantidad}
	return nil
}
func (i *inventarioMem) LotesDeRenglones(_ context.Context, ids []string, bloquear bool) (map[string]domain.LoteRenglon, error) {
	if bloquear {
		i.bloqueos++
	}
	out := map[string]domain.LoteRenglon{}
	for _, id := range ids {
		if l, ok := i.lotes[id]; ok {
			out[id] = l
		}
	}
	return out, nil
}
func (i *inventarioMem) AjustarLoteDeRenglon(_ context.Context, a AjusteRenglon) error {
	if i.falla != nil {
		return i.falla
	}
	i.ajustes = append(i.ajustes, a)
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
