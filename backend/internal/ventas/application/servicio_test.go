package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/kernel/tx"
	"magui/internal/ventas/domain"
)

// --- dobles de prueba ---

type repoMem struct{ ventas map[string]domain.Venta }

func (r *repoMem) Guardar(_ context.Context, v domain.Venta) error { r.ventas[v.ID] = v; return nil }
func (r *repoMem) Obtener(_ context.Context, id string) (domain.Venta, error) {
	v, ok := r.ventas[id]
	if !ok {
		return v, domain.ErrVentaNoEncontrada
	}
	return v, nil
}
func (r *repoMem) MarcarAnulada(_ context.Context, v domain.Venta) error {
	r.ventas[v.ID] = v
	return nil
}
func (r *repoMem) Detalle(_ context.Context, id string) (VentaDetalle, error) {
	v, ok := r.ventas[id]
	if !ok {
		return VentaDetalle{}, domain.ErrVentaNoEncontrada
	}
	costo, completo := v.CostoTotal()
	return VentaDetalle{VentaResumen: VentaResumen{ID: v.ID, Total: v.Total, Estado: v.Estado},
		CostoTotal: costo, Ganancia: v.Total.Restar(costo), CostoCompleto: completo}, nil
}
func (r *repoMem) Listar(context.Context, Filtro, paginacion.Solicitud) ([]VentaResumen, int, error) {
	return nil, len(r.ventas), nil
}

type catalogoMem map[string]ProductoVenta

func (c catalogoMem) Productos(_ context.Context, ids []string) (map[string]ProductoVenta, error) {
	out := map[string]ProductoVenta{}
	for _, id := range ids {
		if p, ok := c[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}

// inventarioMem: stock por producto con un único costo por unidad (o sin costo).
type inventarioMem struct {
	stock     map[string]int
	costo     map[string]*dinero.Dinero
	salidas   map[string]map[string]int // venta -> producto -> unidades
	repuestas []string
}

func (i *inventarioMem) Descontar(_ context.Context, s SalidaVenta) ([]ParteSalida, error) {
	if !s.PermitirSinStock && i.stock[s.ProductoID] < s.Cantidad {
		return nil, errores.Conflicto("stock_insuficiente", fmt.Sprintf("No hay stock suficiente (disponible: %d)", i.stock[s.ProductoID]))
	}
	i.stock[s.ProductoID] -= s.Cantidad
	if i.salidas[s.VentaID] == nil {
		i.salidas[s.VentaID] = map[string]int{}
	}
	i.salidas[s.VentaID][s.ProductoID] += s.Cantidad
	return []ParteSalida{{Cantidad: s.Cantidad, Costo: i.costo[s.ProductoID]}}, nil
}
func (i *inventarioMem) Reponer(_ context.Context, ventaID, _ string) error {
	for p, n := range i.salidas[ventaID] {
		i.stock[p] += n
	}
	i.repuestas = append(i.repuestas, ventaID)
	return nil
}

type idsSec struct{ n int }

func (g *idsSec) Nuevo() string { g.n++; return fmt.Sprintf("id-%d", g.n) }

func d(s string) dinero.Dinero   { return dinero.MustParse(s) }
func pd(s string) *dinero.Dinero { v := d(s); return &v }

var ctx = context.Background()

func entorno() (*Servicio, *repoMem, *inventarioMem) {
	repo := &repoMem{ventas: map[string]domain.Venta{}}
	cat := catalogoMem{
		"kaiak":    {ID: "kaiak", Nombre: "Kaiak", Activo: true, Precio: d("59200"), CostoUltimo: pd("30000")},
		"jabon":    {ID: "jabon", Nombre: "Jabón", Activo: true, Precio: d("5500"), CostoUltimo: pd("2500")},
		"sincosto": {ID: "sincosto", Nombre: "Sin costo", Activo: true, Precio: d("1000")},
		"baja":     {ID: "baja", Nombre: "De baja", Activo: false, Precio: d("100")},
		"combo": {ID: "combo", Nombre: "Combo Madre", Combo: true, Activo: true, Precio: d("62000"),
			Componentes: []Componente{{"kaiak", 1}, {"jabon", 2}}},
	}
	inv := &inventarioMem{
		stock:   map[string]int{"kaiak": 5, "jabon": 10, "sincosto": 3},
		costo:   map[string]*dinero.Dinero{"kaiak": pd("29000")}, // jabón: lote sin costo -> usa el último costo
		salidas: map[string]map[string]int{},
	}
	s := NuevoServicio(Dependencias{Repo: repo, Catalogo: cat, Inventario: inv, Tx: tx.SinTransaccion{},
		IDs: &idsSec{}, Reloj: reloj.Fijo{T: time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)}})
	return s, repo, inv
}

func cobro(pagos ...domain.PagoIngresado) []domain.PagoIngresado { return pagos }

// --- tests ---

func TestVentaSimpleConPagoCombinado(t *testing.T) {
	s, repo, inv := entorno()
	det, err := s.RegistrarVenta(ctx, ComandoVenta{
		Items: []LineaCmd{{"kaiak", 1}, {"jabon", 2}},
		Pagos: cobro(domain.PagoIngresado{Metodo: domain.Transferencia, Monto: d("50000")}, domain.PagoIngresado{Metodo: domain.Efectivo, Monto: d("30000")}),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Total 70200; costo 29000 (lote) + 2×2500 (último costo) = 34000
	if det.Total.String() != "70200.00" || det.CostoTotal.String() != "34000.00" || det.Ganancia.String() != "36200.00" {
		t.Errorf("detalle = %+v", det)
	}
	v := repo.ventas[det.ID]
	if v.Origen != domain.OrigenWeb || len(v.Pagos) != 2 || v.Pagos[1].Vuelto.String() != "9800.00" {
		t.Errorf("venta = %+v", v)
	}
	if inv.stock["kaiak"] != 4 || inv.stock["jabon"] != 8 {
		t.Errorf("stock = %v", inv.stock)
	}
}

func TestVentaCombo(t *testing.T) {
	s, repo, inv := entorno()
	det, err := s.RegistrarVenta(ctx, ComandoVenta{Items: []LineaCmd{{"combo", 2}},
		Pagos: cobro(domain.PagoIngresado{Metodo: domain.Tarjeta, Monto: d("124000")})})
	if err != nil {
		t.Fatal(err)
	}
	// 2 combos = 2 Kaiak + 4 jabones; costo 2×29000 + 4×2500 = 68000
	if inv.stock["kaiak"] != 3 || inv.stock["jabon"] != 6 {
		t.Errorf("stock = %v", inv.stock)
	}
	if det.CostoTotal.String() != "68000.00" || repo.ventas[det.ID].Items[0].ProductoID != "combo" {
		t.Errorf("combo: costo %s items %+v", det.CostoTotal, repo.ventas[det.ID].Items)
	}
}

func TestVentaErrores(t *testing.T) {
	s, repo, inv := entorno()
	casos := map[string]struct {
		cmd    ComandoVenta
		codigo string
	}{
		"falta plata":     {ComandoVenta{Items: []LineaCmd{{"kaiak", 1}}, Pagos: cobro(domain.PagoIngresado{Metodo: domain.QR, Monto: d("50000")})}, "pagos_insuficientes"},
		"sin stock":       {ComandoVenta{Items: []LineaCmd{{"kaiak", 9}}, Pagos: cobro(domain.PagoIngresado{Metodo: domain.Efectivo, Monto: d("999999")})}, "stock_insuficiente"},
		"inexistente":     {ComandoVenta{Items: []LineaCmd{{"nope", 1}}}, "producto_no_encontrado"},
		"desactivado":     {ComandoVenta{Items: []LineaCmd{{"baja", 1}}}, "producto_inactivo"},
		"vacía":           {ComandoVenta{}, "datos_invalidos"},
		"combo sin stock": {ComandoVenta{Items: []LineaCmd{{"combo", 6}}, Pagos: cobro(domain.PagoIngresado{Metodo: domain.Efectivo, Monto: d("999999")})}, "stock_insuficiente"},
	}
	for nombre, c := range casos {
		_, err := s.RegistrarVenta(ctx, c.cmd)
		if e, _ := errores.Como(err); e == nil || e.Codigo != c.codigo {
			t.Errorf("%s: err = %v, want %s", nombre, err, c.codigo)
		}
	}
	if len(repo.ventas) != 0 {
		t.Error("no debería guardarse ninguna venta")
	}
	_, err := s.RegistrarVenta(ctx, casos["sin stock"].cmd)
	if err.Error() != "stock_insuficiente: Kaiak: No hay stock suficiente (disponible: 5)" {
		t.Errorf("el mensaje debería nombrar el producto: %v", err)
	}
	_ = inv
}

func TestVentaSinStockConfirmada(t *testing.T) {
	s, _, inv := entorno()
	det, err := s.RegistrarVenta(ctx, ComandoVenta{Items: []LineaCmd{{"kaiak", 7}}, PermitirSinStock: true,
		Pagos: cobro(domain.PagoIngresado{Metodo: domain.Efectivo, Monto: d("414400")})})
	if err != nil || inv.stock["kaiak"] != -2 {
		t.Fatalf("venta forzada: %v, stock %d", err, inv.stock["kaiak"])
	}
	_ = det
}

func TestVentaConProductoSinCosto(t *testing.T) {
	s, repo, _ := entorno()
	det, err := s.RegistrarVenta(ctx, ComandoVenta{Items: []LineaCmd{{"sincosto", 1}, {"kaiak", 1}},
		Pagos: cobro(domain.PagoIngresado{Metodo: domain.Efectivo, Monto: d("60200")})})
	if err != nil {
		t.Fatal(err)
	}
	v := repo.ventas[det.ID]
	_, completo := v.CostoTotal()
	if completo || v.Items[0].CostoTotal != nil || det.CostoTotal.String() != "29000.00" {
		t.Errorf("costo parcial: %+v", v.Items)
	}
}

func TestAnularVenta(t *testing.T) {
	s, repo, inv := entorno()
	det, _ := s.RegistrarVenta(ctx, ComandoVenta{Items: []LineaCmd{{"combo", 1}},
		Pagos: cobro(domain.PagoIngresado{Metodo: domain.Efectivo, Monto: d("62000")})})
	anulada, err := s.AnularVenta(ctx, det.ID, "El cliente lo devolvió")
	if err != nil || anulada.Estado != domain.Anulada {
		t.Fatalf("anular: %+v %v", anulada, err)
	}
	if inv.stock["kaiak"] != 5 || inv.stock["jabon"] != 10 || len(inv.repuestas) != 1 {
		t.Errorf("stock repuesto = %v", inv.stock)
	}
	if *repo.ventas[det.ID].MotivoAnulacion != "El cliente lo devolvió" {
		t.Error("motivo no guardado")
	}
	if _, err := s.AnularVenta(ctx, det.ID, "otra vez"); !errors.Is(err, domain.ErrVentaYaAnulada) {
		t.Errorf("doble anulación: %v", err)
	}
	if _, err := s.AnularVenta(ctx, "nope", "x"); !errors.Is(err, domain.ErrVentaNoEncontrada) {
		t.Errorf("inexistente: %v", err)
	}
}
