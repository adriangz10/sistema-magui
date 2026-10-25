package dinero

import (
	"encoding/json"
	"testing"
)

func TestParse(t *testing.T) {
	ok := map[string]int64{
		"59200": 5920000, "59200.5": 5920050, "59200.50": 5920050, "59200,05": 5920005,
		"-10.25": -1025, "0": 0, " 7 ": 700, "+3.1": 310,
	}
	for in, want := range ok {
		got, err := Parse(in)
		if err != nil || got.Centavos() != want {
			t.Errorf("Parse(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "1.234", "1.000,00", "1e5", ".5", "--1", "1.2.3"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) debería fallar", in)
		}
	}
}

func TestString(t *testing.T) {
	for c, want := range map[int64]string{5920000: "59200.00", 5: "0.05", -1025: "-10.25", 0: "0.00"} {
		if got := DesdeCentavos(c).String(); got != want {
			t.Errorf("%d -> %q, want %q", c, got, want)
		}
	}
}

func TestAplicarPorcentaje(t *testing.T) {
	casos := []struct {
		monto string
		pct   int64
		want  string
	}{
		{"1000.00", 1000, "1100.00"},   // +10 %
		{"1000.00", -2500, "750.00"},   // -25 %
		{"59200.00", 1050, "65416.00"}, // +10,5 %
		{"0.15", 5000, "0.23"},         // 0.225 -> 0.23 (mitad hacia arriba)
	}
	for _, c := range casos {
		if got := MustParse(c.monto).AplicarPorcentaje(c.pct).String(); got != c.want {
			t.Errorf("%s %+d -> %s, want %s", c.monto, c.pct, got, c.want)
		}
	}
}

func TestRedondearA(t *testing.T) {
	if got := MustParse("65416.00").RedondearA(DesdePesos(100)).String(); got != "65400.00" {
		t.Errorf("got %s", got)
	}
	if got := MustParse("65450.00").RedondearA(DesdePesos(100)).String(); got != "65500.00" {
		t.Errorf("got %s", got)
	}
	if got := MustParse("10.00").RedondearA(0).String(); got != "10.00" {
		t.Errorf("paso 0 no debería cambiar: %s", got)
	}
}

func TestPorcentajeDe(t *testing.T) {
	// margen: 250 de 1000 = 25,00 %
	if got := DesdePesos(250).PorcentajeDe(DesdePesos(1000)); got != 2500 {
		t.Errorf("got %d", got)
	}
	if got := DesdePesos(1).PorcentajeDe(Cero); got != 0 {
		t.Errorf("total cero: %d", got)
	}
}

func TestJSON(t *testing.T) {
	type cuerpo struct {
		Monto Dinero `json:"monto"`
	}
	b, _ := json.Marshal(cuerpo{MustParse("59200")})
	if string(b) != `{"monto":"59200.00"}` {
		t.Errorf("marshal: %s", b)
	}
	for _, in := range []string{`{"monto":"1500.5"}`, `{"monto":1500.5}`} {
		var c cuerpo
		if err := json.Unmarshal([]byte(in), &c); err != nil || c.Monto.Centavos() != 150050 {
			t.Errorf("unmarshal %s -> %d, %v", in, c.Monto, err)
		}
	}
	var c cuerpo
	if err := json.Unmarshal([]byte(`{"monto":"1.999"}`), &c); err == nil {
		t.Error("debería rechazar 3 decimales")
	}
}

func TestScan(t *testing.T) {
	var d Dinero
	for _, src := range []any{[]byte("59200.00"), "59200.00", int64(5920000)} {
		if err := d.Scan(src); err != nil || d.Centavos() != 5920000 {
			t.Errorf("Scan(%#v) = %d, %v", src, d, err)
		}
	}
	if err := d.Scan(float64(1.5)); err == nil {
		t.Error("un REAL debería rechazarse (perdería precisión)")
	}
}
