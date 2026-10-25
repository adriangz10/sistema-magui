package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"magui/internal/inventario/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/kernel/tx"
)

// --- dobles de prueba ---

type repoMem struct {
	lotes map[string]domain.Lote
	movs  []domain.Movimiento
	dias  int
}

func (r *repoMem) CrearLote(_ context.Context, l domain.Lote) error { r.lotes[l.ID] = l; return nil }
func (r *repoMem) stockLote(id string) int {
	n := 0
	for _, m := range r.movs {
		if m.LoteID != nil && *m.LoteID == id {
			n += m.Cantidad
		}
	}
	return n
}
func (r *repoMem) ObtenerLote(_ context.Context, id string) (domain.LoteStock, error) {
	l, ok := r.lotes[id]
	if !ok {
		return domain.LoteStock{}, domain.ErrLoteNoEncontrado
	}
	return domain.LoteStock{Lote: l, Stock: r.stockLote(id)}, nil
}
func (r *repoMem) RegistrarMovimientos(_ context.Context, m []domain.Movimiento) error {
	r.movs = append(r.movs, m...)
	return nil
}
func (r *repoMem) MovimientosPorReferencia(_ context.Context, ref string, tipo domain.TipoMovimiento) ([]domain.Movimiento, error) {
	var out []domain.Movimiento
	for _, m := range r.movs {
		if m.ReferenciaID != nil && *m.ReferenciaID == ref && m.Tipo == tipo {
			out = append(out, m)
		}
	}
	return out, nil
}
func (r *repoMem) LotesConStock(_ context.Context, p string) ([]domain.LoteStock, error) {
	var out []domain.LoteStock
	for _, l := range r.lotes {
		if s := r.stockLote(l.ID); l.ProductoID == p && s != 0 {
			out = append(out, domain.LoteStock{Lote: l, Stock: s})
		}
	}
	return out, nil
}
func (r *repoMem) StockProducto(_ context.Context, p string) (int, error) {
	n := 0
	for _, m := range r.movs {
		if m.ProductoID == p {
			n += m.Cantidad
		}
	}
	return n, nil
}
func (r *repoMem) ListarMovimientos(context.Context, FiltroMovimientos, paginacion.Solicitud) ([]MovimientoDTO, int, error) {
	return nil, len(r.movs), nil
}
func (r *repoMem) ListarVencimientos(_ context.Context, f FiltroVencimientos, _ paginacion.Solicitud) ([]VencimientoDTO, int, error) {
	var out []VencimientoDTO
	for _, l := range r.lotes {
		if l.Vencimiento == nil || r.stockLote(l.ID) <= 0 {
			continue
		}
		if f.Desde != nil && l.Vencimiento.Antes(*f.Desde) || f.Hasta != nil && !l.Vencimiento.Antes(*f.Hasta) {
			continue
		}
		out = append(out, VencimientoDTO{LoteID: l.ID, Vencimiento: *l.Vencimiento, Stock: r.stockLote(l.ID)})
	}
	return out, len(out), nil
}
func (r *repoMem) ContarAlertas(context.Context, fecha.Fecha, fecha.Fecha) (AlertasDTO, error) {
	return AlertasDTO{Vencidos: 1}, nil
}
func (r *repoMem) DiasAlerta(context.Context) (int, error)          { return r.dias, nil }
func (r *repoMem) GuardarDiasAlerta(_ context.Context, d int) error { r.dias = d; return nil }

type productosMem map[string]InfoProducto

func (p productosMem) Info(_ context.Context, ids []string) (map[string]InfoProducto, error) {
	out := map[string]InfoProducto{}
	for _, id := range ids {
		if v, ok := p[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}

type idsSec struct{ n int }

func (g *idsSec) Nuevo() string { g.n++; return fmt.Sprintf("id-%03d", g.n) }

// Hoy en Argentina: 25/10/2026
var ahora = time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)

func entorno() (*Servicio, *repoMem) {
	repo := &repoMem{lotes: map[string]domain.Lote{}, dias: 90}
	svc := NuevoServicio(Dependencias{
		Repo: repo,
		Productos: productosMem{
			"kaiak": {ID: "kaiak", Nombre: "Kaiak", Activo: true},
			"combo": {ID: "combo", Nombre: "Combo", Combo: true, Activo: true},
		},
		Tx: tx.SinTransaccion{}, IDs: &idsSec{}, Reloj: reloj.Fijo{T: ahora},
	})
	return svc, repo
}

var ctx = context.Background()

func f(s string) *fecha.Fecha { v, _ := fecha.Parse(s); return &v }

func costo(s string) *dinero.Dinero { d := dinero.MustParse(s); return &d }

// --- tests ---

func TestIngresoYSalidaFEFO(t *testing.T) {
	s, repo := entorno()
	if _, err := s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: 3, Vencimiento: f("2027-05-01"), Costo: costo("100"), Tipo: domain.MovCompra}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: 2, Vencimiento: f("2026-12-01"), Costo: costo("90"), Tipo: domain.MovCompra}); err != nil {
		t.Fatal(err)
	}
	ref := "venta-1"
	asig, err := s.RegistrarSalida(ctx, Salida{ProductoID: "kaiak", Cantidad: 3, Tipo: domain.MovVenta, ReferenciaID: &ref})
	if err != nil {
		t.Fatal(err)
	}
	// Sale primero el que vence en diciembre (2 u. a $90) y luego 1 del de mayo ($100)
	if len(asig) != 2 || asig[0].Cantidad != 2 || asig[0].Costo.String() != "90.00" || asig[1].Cantidad != 1 {
		t.Fatalf("asignaciones = %+v", asig)
	}
	if st, _ := repo.StockProducto(ctx, "kaiak"); st != 2 {
		t.Errorf("stock = %d", st)
	}
	for _, m := range repo.movs[2:] {
		if m.Cantidad >= 0 || m.Tipo != domain.MovVenta || *m.ReferenciaID != ref {
			t.Errorf("movimiento de venta inesperado: %+v", m)
		}
	}
}

func TestSalidaSinStock(t *testing.T) {
	s, repo := entorno()
	_, _ = s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: 1, Tipo: domain.MovCompra})
	_, err := s.RegistrarSalida(ctx, Salida{ProductoID: "kaiak", Cantidad: 2, Tipo: domain.MovVenta})
	if e, ok := errores.Como(err); !ok || e.Codigo != "stock_insuficiente" {
		t.Fatalf("err = %v", err)
	}
	if len(repo.movs) != 1 {
		t.Error("no debería registrar movimientos si falla")
	}
	// Offline: se permite y el faltante queda sin lote
	asig, err := s.RegistrarSalida(ctx, Salida{ProductoID: "kaiak", Cantidad: 2, Tipo: domain.MovVenta, PermitirNegativo: true})
	if err != nil || len(asig) != 2 || asig[1].LoteID != nil {
		t.Fatalf("permitir negativo: %+v %v", asig, err)
	}
	if st, _ := repo.StockProducto(ctx, "kaiak"); st != -1 {
		t.Errorf("stock = %d", st)
	}
}

func TestCombosYProductosInexistentes(t *testing.T) {
	s, _ := entorno()
	if _, err := s.RegistrarIngreso(ctx, Ingreso{ProductoID: "combo", Cantidad: 1}); !errors.Is(err, domain.ErrComboSinStock) {
		t.Errorf("combo: %v", err)
	}
	if _, err := s.RegistrarSalida(ctx, Salida{ProductoID: "nope", Cantidad: 1}); !errors.Is(err, ErrProductoNoEncontrado) {
		t.Errorf("inexistente: %v", err)
	}
	if _, err := s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: 0}); err == nil {
		t.Error("cantidad cero")
	}
}

func TestAjustes(t *testing.T) {
	s, repo := entorno()
	// Positivo sin lote: crea un lote nuevo con vencimiento
	st, err := s.AjustarStock(ctx, ComandoAjuste{ProductoID: "kaiak", Cantidad: 4, Motivo: "Conteo", Vencimiento: f("2026-11-10")})
	if err != nil || st.Stock != 4 || len(st.Lotes) != 1 {
		t.Fatalf("ajuste positivo: %+v %v", st, err)
	}
	if l := st.Lotes[0]; l.Estado != domain.PorVencer || *l.DiasRestantes != 16 {
		t.Errorf("lote = %+v", l)
	}
	loteID := st.Lotes[0].ID

	// Negativo sobre un lote: no puede quitar más de lo que tiene
	if _, err := s.AjustarStock(ctx, ComandoAjuste{ProductoID: "kaiak", Cantidad: -5, Motivo: "Vencido", LoteID: loteID}); err == nil {
		t.Error("debería impedir stock negativo en el lote")
	}
	st, err = s.AjustarStock(ctx, ComandoAjuste{ProductoID: "kaiak", Cantidad: -1, Motivo: "Rotura", LoteID: loteID})
	if err != nil || st.Stock != 3 {
		t.Fatalf("ajuste de lote: %+v %v", st, err)
	}
	// Negativo general sin stock suficiente
	if _, err := s.AjustarStock(ctx, ComandoAjuste{ProductoID: "kaiak", Cantidad: -10, Motivo: "x"}); err == nil {
		t.Error("ajuste negativo mayor al stock")
	}
	if _, err := s.AjustarStock(ctx, ComandoAjuste{ProductoID: "kaiak", Cantidad: 1, Motivo: ""}); err == nil {
		t.Error("motivo obligatorio")
	}
	if _, err := s.AjustarStock(ctx, ComandoAjuste{ProductoID: "kaiak", Cantidad: 1, Motivo: "x", LoteID: "nope"}); !errors.Is(err, domain.ErrLoteNoEncontrado) {
		t.Errorf("lote inexistente: %v", err)
	}
	for _, m := range repo.movs {
		if m.Tipo != domain.MovAjuste || m.Nota == "" {
			t.Errorf("movimiento de ajuste sin tipo o nota: %+v", m)
		}
	}
}

func TestVencimientos(t *testing.T) {
	s, _ := entorno()
	for _, v := range []string{"2026-10-01", "2026-11-15", "2027-06-01"} {
		_, _ = s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: 1, Vencimiento: f(v), Tipo: domain.MovCompra})
	}
	casos := map[EstadoVencimientos]domain.EstadoVencimiento{FiltroVencidos: domain.Vencido, FiltroPorVencer: domain.PorVencer}
	for filtro, estado := range casos {
		pag, err := s.ListarVencimientos(ctx, filtro, paginacion.Nueva(1, 10))
		if err != nil || pag.Total != 1 || pag.Items[0].Estado != estado {
			t.Errorf("%s: %+v %v", filtro, pag, err)
		}
	}
	if pag, _ := s.ListarVencimientos(ctx, FiltroTodos, paginacion.Nueva(1, 10)); pag.Total != 3 {
		t.Errorf("todos = %d", pag.Total)
	}
	if _, err := s.ListarVencimientos(ctx, "otro", paginacion.Nueva(1, 10)); err == nil {
		t.Error("filtro inválido")
	}
}

func TestConfiguracionDiasAlerta(t *testing.T) {
	s, _ := entorno()
	if err := s.CambiarDiasAlerta(ctx, 400); err == nil {
		t.Error("fuera de rango")
	}
	if err := s.CambiarDiasAlerta(ctx, 30); err != nil {
		t.Fatal(err)
	}
	a, _ := s.Alertas(ctx)
	if a.DiasAlerta != 30 || a.Vencidos != 1 {
		t.Errorf("alertas = %+v", a)
	}
}

func TestRevertirSalidas(t *testing.T) {
	s, repo := entorno()
	_, _ = s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: 2, Vencimiento: f("2026-12-01"), Tipo: domain.MovCompra})
	_, _ = s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: 2, Vencimiento: f("2027-12-01"), Tipo: domain.MovCompra})
	ref := "venta-1"
	if _, err := s.RegistrarSalida(ctx, Salida{ProductoID: "kaiak", Cantidad: 3, Tipo: domain.MovVenta, ReferenciaID: &ref}); err != nil {
		t.Fatal(err)
	}
	n, err := s.RevertirSalidas(ctx, ref, domain.MovVenta, "Anulación")
	if err != nil || n != 3 {
		t.Fatalf("repuestas = %d, %v", n, err)
	}
	if st, _ := repo.StockProducto(ctx, "kaiak"); st != 4 {
		t.Errorf("stock = %d", st)
	}
	// Cada lote recupera lo suyo
	for _, l := range repo.lotes {
		if repo.stockLote(l.ID) != 2 {
			t.Errorf("lote %s = %d", l.ID, repo.stockLote(l.ID))
		}
	}
	// Idempotente
	if n, _ := s.RevertirSalidas(ctx, ref, domain.MovVenta, "otra vez"); n != 0 {
		t.Errorf("segunda reversión repuso %d", n)
	}
}
