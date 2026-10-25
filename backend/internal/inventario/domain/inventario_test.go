package domain

import (
	"testing"
	"time"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
)

func venc(s string) *fecha.Fecha {
	f, _ := fecha.Parse(s)
	return &f
}

func lote(id string, v *fecha.Fecha, stock int, creado int) LoteStock {
	c := dinero.DesdePesos(int64(creado * 100))
	return LoteStock{Lote: Lote{ID: id, Vencimiento: v, Costo: &c, CreatedAt: time.Unix(int64(creado), 0)}, Stock: stock}
}

func TestAsignarFEFO(t *testing.T) {
	lotes := []LoteStock{
		lote("sin-fecha", nil, 10, 1),
		lote("dic", venc("2026-12-01"), 2, 2),
		lote("nov", venc("2026-11-01"), 1, 3),
		lote("vacio", venc("2026-10-01"), 0, 4),
	}
	got := AsignarFEFO(lotes, 5, fecha.Nueva(2026, 10, 25))
	want := []struct {
		lote string
		cant int
	}{{"nov", 1}, {"dic", 2}, {"sin-fecha", 2}}
	if len(got) != len(want) {
		t.Fatalf("asignaciones = %+v", got)
	}
	for i, w := range want {
		if *got[i].LoteID != w.lote || got[i].Cantidad != w.cant {
			t.Errorf("[%d] = %s x%d, want %s x%d", i, *got[i].LoteID, got[i].Cantidad, w.lote, w.cant)
		}
	}
	if got[0].Costo.String() != "300.00" {
		t.Errorf("debería llevar el costo del lote: %v", got[0].Costo)
	}
}

func TestAsignarFEFOSinStockSuficiente(t *testing.T) {
	got := AsignarFEFO([]LoteStock{lote("a", nil, 2, 1)}, 5, fecha.Nueva(2026, 10, 25))
	if len(got) != 2 || got[1].LoteID != nil || got[1].Cantidad != 3 {
		t.Fatalf("faltante sin lote: %+v", got)
	}
	if got := AsignarFEFO(nil, 1, fecha.Nueva(2026, 10, 25)); len(got) != 1 || got[0].LoteID != nil {
		t.Fatalf("sin lotes: %+v", got)
	}
}

func TestAsignarFEFOMismaFechaPrimeroElMasViejo(t *testing.T) {
	got := AsignarFEFO([]LoteStock{lote("nuevo", venc("2026-11-01"), 5, 9), lote("viejo", venc("2026-11-01"), 5, 1)}, 1, fecha.Nueva(2026, 10, 25))
	if *got[0].LoteID != "viejo" {
		t.Errorf("got %s", *got[0].LoteID)
	}
}

func TestClasificar(t *testing.T) {
	hoy := fecha.Nueva(2026, 10, 25)
	casos := []struct {
		v      *fecha.Fecha
		estado EstadoVencimiento
		dias   int
	}{
		{venc("2026-10-24"), Vencido, -1},
		{venc("2026-10-25"), PorVencer, 0},
		{venc("2027-01-22"), PorVencer, 89},
		{venc("2027-01-23"), Vigente, 90},
	}
	for _, c := range casos {
		e, d := Clasificar(c.v, hoy, 90)
		if e != c.estado || *d != c.dias {
			t.Errorf("%s: %s %d, want %s %d", c.v, e, *d, c.estado, c.dias)
		}
	}
	if e, d := Clasificar(nil, hoy, 90); e != SinFecha || d != nil {
		t.Errorf("sin fecha: %s %v", e, d)
	}
}

func TestValidaciones(t *testing.T) {
	if ValidarAjuste(0, "") == nil || ValidarAjuste(1, " ") == nil || ValidarAjuste(-2, "Rotura") != nil {
		t.Error("ValidarAjuste")
	}
	if ValidarDiasAlerta(0) == nil || ValidarDiasAlerta(366) == nil || ValidarDiasAlerta(90) != nil {
		t.Error("ValidarDiasAlerta")
	}
	if err := ErrStockInsuficiente(3); err == nil || err.Error() != "stock_insuficiente: No hay stock suficiente (disponible: 3)" {
		t.Errorf("mensaje = %v", err)
	}
}

func TestAsignarFEFOVencidosAlFinal(t *testing.T) {
	hoy := fecha.Nueva(2026, 10, 25)
	got := AsignarFEFO([]LoteStock{
		lote("vencido", venc("2026-09-30"), 1, 1),
		lote("sin-fecha", nil, 1, 2),
		lote("vigente", venc("2027-01-01"), 1, 3),
	}, 3, hoy)
	orden := []string{"vigente", "sin-fecha", "vencido"}
	for i, w := range orden {
		if *got[i].LoteID != w {
			t.Errorf("[%d] = %s, want %s", i, *got[i].LoteID, w)
		}
	}
}
