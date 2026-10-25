package domain

import (
	"testing"
	"time"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
)

var art = time.FixedZone("ART", -3*3600)

// Domingo 25/10/2026 a las 15:00 de Argentina
var ahora = time.Date(2026, 10, 25, 18, 0, 0, 0, time.UTC)

func enART(t time.Time) string { return t.In(art).Format("2006-01-02 15:04") }

func TestPeriodosEnCurso(t *testing.T) {
	casos := []struct {
		tipo                             TipoPeriodo
		desde, hasta, antDesde, antHasta string
	}{
		{Dia, "2026-10-25 00:00", "2026-10-26 00:00", "2026-10-24 00:00", "2026-10-24 15:00"},
		{Semana, "2026-10-19 00:00", "2026-10-26 00:00", "2026-10-12 00:00", "2026-10-18 15:00"},
		{Mes, "2026-10-01 00:00", "2026-11-01 00:00", "2026-09-01 00:00", "2026-09-25 15:00"},
		{Anio, "2026-01-01 00:00", "2027-01-01 00:00", "2025-01-01 00:00", "2025-10-25 15:00"},
	}
	for _, c := range casos {
		p, err := CalcularPeriodo(c.tipo, ahora, art, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		got := [4]string{enART(p.Desde), enART(p.Hasta), enART(p.AnteriorDesde), enART(p.AnteriorHasta)}
		want := [4]string{c.desde, c.hasta, c.antDesde, c.antHasta}
		if got != want {
			t.Errorf("%s: %v, want %v", c.tipo, got, want)
		}
	}
}

func TestPeriodoRango(t *testing.T) {
	d, h := fecha.Nueva(2026, 10, 1), fecha.Nueva(2026, 10, 10)
	p, err := CalcularPeriodo(Rango, ahora, art, &d, &h)
	if err != nil {
		t.Fatal(err)
	}
	if enART(p.Desde) != "2026-10-01 00:00" || enART(p.Hasta) != "2026-10-11 00:00" ||
		enART(p.AnteriorDesde) != "2026-09-21 00:00" || !p.AnteriorHasta.Equal(p.Desde) {
		t.Errorf("rango = %+v", p)
	}
	if len(p.Dias(art)) != 10 {
		t.Errorf("días = %d", len(p.Dias(art)))
	}
	if _, err := CalcularPeriodo(Rango, ahora, art, &h, &d); err == nil {
		t.Error("hasta antes de desde")
	}
	lejos := d.SumarDias(400)
	if _, err := CalcularPeriodo(Rango, ahora, art, &d, &lejos); err == nil {
		t.Error("rango de más de un año")
	}
	if _, err := CalcularPeriodo(Rango, ahora, art, nil, nil); err == nil {
		t.Error("rango sin fechas")
	}
	if _, err := CalcularPeriodo("quincena", ahora, art, nil, nil); err == nil {
		t.Error("período desconocido")
	}
}

func TestCalculos(t *testing.T) {
	if Porcentaje(dinero.MustParse("34680"), dinero.MustParse("70200")) != 49.4 {
		t.Error("margen")
	}
	if v := Variacion(dinero.MustParse("150"), dinero.MustParse("100")); v == nil || *v != 50 {
		t.Errorf("variación = %v", v)
	}
	if Variacion(dinero.MustParse("10"), dinero.Cero) != nil {
		t.Error("sin base no hay variación")
	}
	if Promedio(dinero.MustParse("100"), 3).String() != "33.33" || !Promedio(dinero.MustParse("1"), 0).EsCero() {
		t.Error("promedio")
	}
}
