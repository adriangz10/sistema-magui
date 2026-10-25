package fecha

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseYOperaciones(t *testing.T) {
	f, err := Parse("2026-09-30")
	if err != nil || f.String() != "2026-09-30" {
		t.Fatalf("%v %v", f, err)
	}
	if _, err := Parse("30/09/2026"); err == nil {
		t.Error("formato inválido debería fallar")
	}
	hoy := Nueva(2026, 10, 25)
	if d := hoy.DiasHasta(f); d != -25 {
		t.Errorf("días hasta = %d", d)
	}
	if d := hoy.DiasHasta(hoy.SumarDias(90)); d != 90 {
		t.Errorf("sumar días = %d", d)
	}
	if !f.Antes(hoy) || hoy.Antes(f) {
		t.Error("Antes")
	}
}

func TestDelInstante(t *testing.T) {
	// 01:00 UTC del 26 es todavía el 25 en Argentina (UTC-3)
	art := time.FixedZone("ART", -3*3600)
	if got := DelInstante(time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC), art); got.String() != "2026-10-25" {
		t.Errorf("got %s", got)
	}
}

func TestJSONYScan(t *testing.T) {
	type c struct {
		F *Fecha `json:"f"`
		G Fecha  `json:"g"`
	}
	var v c
	if err := json.Unmarshal([]byte(`{"f":"2026-03-20","g":null}`), &v); err != nil || v.F.String() != "2026-03-20" || !v.G.IsZero() {
		t.Fatalf("%+v %v", v, err)
	}
	b, _ := json.Marshal(c{G: Nueva(2026, 1, 2)})
	if string(b) != `{"f":null,"g":"2026-01-02"}` {
		t.Errorf("json = %s", b)
	}
	var f Fecha
	for _, src := range []any{time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), []byte("2026-03-20"), "2026-03-20T00:00:00Z"} {
		if err := f.Scan(src); err != nil || f.String() != "2026-03-20" {
			t.Errorf("Scan(%v) = %v %v", src, f, err)
		}
	}
	if v, _ := (Fecha{}).Value(); v != nil {
		t.Error("fecha vacía debería guardarse como NULL")
	}
}
