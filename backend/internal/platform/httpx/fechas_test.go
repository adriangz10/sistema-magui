package httpx

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestRangoFechasDias(t *testing.T) {
	r := httptest.NewRequest("GET", "/?desde=2026-11-01&hasta=2026-11-01", nil)
	d, h, err := RangoFechas(r)
	if err != nil {
		t.Fatal(err)
	}
	// Día de Argentina (UTC-3): 03:00 UTC a 03:00 UTC del día siguiente.
	if !d.Equal(time.Date(2026, 11, 1, 3, 0, 0, 0, time.UTC)) || !h.Equal(time.Date(2026, 11, 2, 3, 0, 0, 0, time.UTC)) {
		t.Errorf("rango = %v .. %v", d, h)
	}
}

func TestRangoFechasInstantes(t *testing.T) {
	r := httptest.NewRequest("GET", "/?desde=2026-11-01T12:00:00.123Z&hasta=2026-11-01T20:30:00Z", nil)
	d, h, err := RangoFechas(r)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Equal(time.Date(2026, 11, 1, 12, 0, 0, 123e6, time.UTC)) || !h.Equal(time.Date(2026, 11, 1, 20, 30, 0, 0, time.UTC)) {
		t.Errorf("rango = %v .. %v (hasta debe quedar tal cual, exclusivo)", d, h)
	}
}

func TestRangoFechasInvalido(t *testing.T) {
	for _, q := range []string{"?desde=ayer", "?desde=2026-11-02&hasta=2026-11-01", "?desde=2026-11-01T10:00:00Z&hasta=2026-11-01T10:00:00Z"} {
		if _, _, err := RangoFechas(httptest.NewRequest("GET", "/"+q, nil)); err == nil {
			t.Errorf("%s: se esperaba error", q)
		}
	}
}
