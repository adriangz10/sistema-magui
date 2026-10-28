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
	// Bloqueos pedidos (ADR 0002): el fake no bloquea, solo cuenta que se pidió la lectura bloqueante.
	bloqueantes int
	// lecturas del stock fuera de los lotes (el chequeo de faltante usa lotes bloqueados + sin lote).
	lecturasSinLote int
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
func (r *repoMem) ObtenerLote(_ context.Context, id string, bloquear bool) (domain.LoteStock, error) {
	if bloquear {
		r.bloqueantes++
	}
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
func (r *repoMem) BloquearLotesDeProducto(context.Context, string) error { r.bloqueantes++; return nil }
func (r *repoMem) LotesPorCompraItem(_ context.Context, ids []string, bloquear bool) ([]domain.LoteStock, error) {
	if bloquear {
		r.bloqueantes++
	}
	var out []domain.LoteStock
	for _, l := range r.lotes {
		for _, id := range ids {
			if l.CompraItemID != nil && *l.CompraItemID == id {
				out = append(out, domain.LoteStock{Lote: l, Stock: r.stockLote(l.ID)})
			}
		}
	}
	return out, nil
}
func (r *repoMem) ActualizarLote(_ context.Context, id string, c *dinero.Dinero, v *fecha.Fecha) error {
	l := r.lotes[id]
	l.Costo, l.Vencimiento = c, v
	r.lotes[id] = l
	return nil
}
func (r *repoMem) LotesConStock(_ context.Context, p string, bloquear bool) ([]domain.LoteStock, error) {
	if bloquear {
		r.bloqueantes++
	}
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
func (r *repoMem) StockSinLote(_ context.Context, p string) (int, error) {
	r.lecturasSinLote++
	n := 0
	for _, m := range r.movs {
		if m.ProductoID == p && m.LoteID == nil {
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

// ADR 0002: las salidas y los ajustes por lote leen el stock con lectura bloqueante.
func TestSalidasYAjustesUsanLecturaBloqueante(t *testing.T) {
	s, repo := entorno()
	st, _ := s.AjustarStock(ctx, ComandoAjuste{ProductoID: "kaiak", Cantidad: 5, Motivo: "Conteo"})
	repo.bloqueantes = 0
	if _, err := s.RegistrarSalida(ctx, Salida{ProductoID: "kaiak", Cantidad: 1, Tipo: domain.MovVenta}); err != nil {
		t.Fatal(err)
	}
	if repo.bloqueantes == 0 {
		t.Error("RegistrarSalida debería bloquear los lotes del producto")
	}
	repo.bloqueantes = 0
	if _, err := s.AjustarStock(ctx, ComandoAjuste{ProductoID: "kaiak", Cantidad: -1, Motivo: "Rotura", LoteID: st.Lotes[0].ID}); err != nil {
		t.Fatal(err)
	}
	if repo.bloqueantes < 2 {
		t.Errorf("ajustarLote debería bloquear el producto y leer el lote bloqueando (%d)", repo.bloqueantes)
	}
	repo.bloqueantes = 0
	_, _ = s.StockDeProducto(ctx, "kaiak")
	if repo.bloqueantes != 0 {
		t.Error("las consultas de solo lectura no deberían bloquear")
	}
}

func lotePorCompra(t *testing.T, s *Servicio, repo *repoMem, itemID string, cant int, costoIni string, venc *fecha.Fecha) string {
	t.Helper()
	id, err := s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: cant, Costo: costo(costoIni), Vencimiento: venc,
		CompraItemID: &itemID, Tipo: domain.MovCompra})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestAjustarLoteDeCompra(t *testing.T) {
	s, repo := entorno()
	lote := lotePorCompra(t, s, repo, "item-1", 4, "100", f("2027-09-30"))
	n0 := len(repo.movs)

	// Sube 2: movimiento EDICION_COMPRA +2 en el mismo lote con la compra como referencia.
	err := s.AjustarLoteDeCompra(ctx, AjusteLoteCompra{LoteID: lote, Diferencia: 2, Costo: *costo("100"), Vencimiento: f("2027-09-30"), CompraID: "c1", Nota: "Edición de compra del 20/10/2026"})
	if err != nil {
		t.Fatal(err)
	}
	m := repo.movs[n0]
	if len(repo.movs) != n0+1 || m.Cantidad != 2 || m.Tipo != domain.MovEdicionCompra || *m.LoteID != lote || *m.ReferenciaID != "c1" || m.Nota == "" {
		t.Errorf("movimiento = %+v", repo.movs[n0:])
	}
	if repo.stockLote(lote) != 6 {
		t.Errorf("stock = %d", repo.stockLote(lote))
	}

	// Venden 5 (queda 1): bajar 2 deja el lote negativo -> rechazado y sin movimiento; bajar 1 hasta 0 se acepta.
	_, _ = s.RegistrarSalida(ctx, Salida{ProductoID: "kaiak", Cantidad: 5, Tipo: domain.MovVenta})
	n1 := len(repo.movs)
	err = s.AjustarLoteDeCompra(ctx, AjusteLoteCompra{LoteID: lote, Diferencia: -2, Costo: *costo("100"), Vencimiento: f("2027-09-30"), CompraID: "c1"})
	if e, ok := errores.Como(err); !ok || e.Codigo != "stock_insuficiente" || len(repo.movs) != n1 {
		t.Fatalf("por debajo del consumo: %v", err)
	}
	if err := s.AjustarLoteDeCompra(ctx, AjusteLoteCompra{LoteID: lote, Diferencia: -1, Costo: *costo("100"), Vencimiento: f("2027-09-30"), CompraID: "c1"}); err != nil {
		t.Fatal(err)
	}
	if repo.stockLote(lote) != 0 {
		t.Errorf("stock = %d", repo.stockLote(lote))
	}
	if err := s.AjustarLoteDeCompra(ctx, AjusteLoteCompra{LoteID: "nope", Diferencia: 1}); !errors.Is(err, domain.ErrLoteNoEncontrado) {
		t.Errorf("lote inexistente: %v", err)
	}
}

func TestAjustarLoteDeCompraCostoYVencimientoSinMovimiento(t *testing.T) {
	s, repo := entorno()
	lote := lotePorCompra(t, s, repo, "item-1", 4, "100", f("2027-09-30"))
	n0 := len(repo.movs)
	// Diferencia 0: solo se actualiza el lote en el lugar.
	if err := s.AjustarLoteDeCompra(ctx, AjusteLoteCompra{LoteID: lote, Costo: *costo("120"), Vencimiento: f("2027-12-31"), CompraID: "c1"}); err != nil {
		t.Fatal(err)
	}
	l := repo.lotes[lote]
	if len(repo.movs) != n0 || l.Costo.String() != "120.00" || l.Vencimiento.String() != "2027-12-31" {
		t.Errorf("lote = %+v movs=%d", l, len(repo.movs)-n0)
	}
	// Vencimiento nil = el lote queda sin vencimiento (SIN_FECHA).
	if err := s.AjustarLoteDeCompra(ctx, AjusteLoteCompra{LoteID: lote, Costo: *costo("120"), Vencimiento: nil, CompraID: "c1"}); err != nil {
		t.Fatal(err)
	}
	if repo.lotes[lote].Vencimiento != nil {
		t.Errorf("vencimiento = %v", repo.lotes[lote].Vencimiento)
	}
	// Sin cambios de nada: no escribe.
	repo.lotes[lote] = domain.Lote{ID: lote, ProductoID: "kaiak", Costo: costo("120")}
	if err := s.AjustarLoteDeCompra(ctx, AjusteLoteCompra{LoteID: lote, Costo: *costo("120"), CompraID: "c1"}); err != nil || len(repo.movs) != n0 {
		t.Errorf("sin cambios: %v", err)
	}
}

func TestLotesDeCompra(t *testing.T) {
	s, repo := entorno()
	lotePorCompra(t, s, repo, "item-1", 4, "100", nil)
	lotePorCompra(t, s, repo, "item-2", 2, "50", nil)
	ls, err := s.LotesDeCompra(ctx, []string{"item-1"}, true)
	if err != nil || len(ls) != 1 || ls[0].Stock != 4 || repo.bloqueantes == 0 {
		t.Errorf("lotes = %+v %v", ls, err)
	}
	if ls, _ := s.LotesDeCompra(ctx, nil, false); len(ls) != 0 {
		t.Error("sin ids no devuelve nada")
	}
}

// El chequeo de faltante usa los lotes ya leídos bajo bloqueo más los movimientos sin lote,
// no una lectura aparte del stock total (ADR 0002).
func TestFaltanteSeCalculaConLosLotesBloqueados(t *testing.T) {
	s, repo := entorno()
	_, _ = s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: 1, Tipo: domain.MovCompra})
	// Venta offline sin stock: deja un movimiento sin lote de -1 (stock total del producto 0).
	if _, err := s.RegistrarSalida(ctx, Salida{ProductoID: "kaiak", Cantidad: 2, Tipo: domain.MovVenta, PermitirNegativo: true}); err != nil {
		t.Fatal(err)
	}
	repo.lecturasSinLote = 0
	// Hay 0 en total (lote 0 + sin lote -1 + ...): pedir 1 más debe fallar con disponible 0.
	_, _ = s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: 1, Tipo: domain.MovCompra})
	_, err := s.RegistrarSalida(ctx, Salida{ProductoID: "kaiak", Cantidad: 2, Tipo: domain.MovVenta})
	e, ok := errores.Como(err)
	if !ok || e.Codigo != "stock_insuficiente" || e.Mensaje != "No hay stock suficiente (disponible: 0)" {
		t.Fatalf("err = %v", err)
	}
	if repo.lecturasSinLote == 0 {
		t.Error("debería sumar los movimientos sin lote")
	}
	// Lote con 1 unidad y un -1 sin lote (offline previo): el total es 0, no se puede vender 1.
	s, repo = entorno()
	_, _ = s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: 1, Tipo: domain.MovCompra})
	repo.movs = append(repo.movs, domain.Movimiento{ID: "x", ProductoID: "kaiak", Cantidad: -1, Tipo: domain.MovVenta})
	if _, err := s.RegistrarSalida(ctx, Salida{ProductoID: "kaiak", Cantidad: 2, Tipo: domain.MovVenta}); err == nil {
		t.Error("debería faltar stock")
	}
	// Con stock suficiente en lotes no consulta nada aparte.
	s, repo = entorno()
	_, _ = s.RegistrarIngreso(ctx, Ingreso{ProductoID: "kaiak", Cantidad: 3, Tipo: domain.MovCompra})
	if _, err := s.RegistrarSalida(ctx, Salida{ProductoID: "kaiak", Cantidad: 2, Tipo: domain.MovVenta}); err != nil || repo.lecturasSinLote != 0 {
		t.Errorf("err=%v lecturas=%d", err, repo.lecturasSinLote)
	}
}
