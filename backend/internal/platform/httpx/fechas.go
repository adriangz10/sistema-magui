package httpx

import (
	"net/http"
	"time"

	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/reloj"
)

// RangoFechas lee ?desde=&hasta= como días de Argentina (AAAA-MM-DD): devuelve el inicio de "desde"
// y el inicio del día siguiente a "hasta" (hasta inclusivo).
// También acepta instantes RFC 3339 (ej. el período de una caja): ahí "hasta" es exclusivo tal cual.
func RangoFechas(r *http.Request) (desde, hasta *time.Time, err error) {
	var v errores.Validacion
	leer := func(campo string, diaSiguiente bool) *time.Time {
		s := r.URL.Query().Get(campo)
		if s == "" {
			return nil
		}
		if f, e := fecha.Parse(s); e == nil {
			if diaSiguiente {
				f = f.SumarDias(1)
			}
			t := f.InicioEn(reloj.ZonaArgentina)
			return &t
		}
		if t, e := time.Parse(time.RFC3339Nano, s); e == nil {
			return &t
		}
		v.Agregar(campo, "Use el formato AAAA-MM-DD o fecha y hora ISO")
		return nil
	}
	desde, hasta = leer("desde", false), leer("hasta", true)
	if desde != nil && hasta != nil && !desde.Before(*hasta) {
		v.Agregar("hasta", "Debe ser igual o posterior a desde")
	}
	return desde, hasta, v.Err()
}

// FechaOInstante interpreta "AAAA-MM-DD" (mediodía de Argentina) o RFC 3339.
// Texto vacío devuelve nil.
func FechaOInstante(campo, s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	if f, err := fecha.Parse(s); err == nil {
		t := f.InicioEn(reloj.ZonaArgentina).Add(12 * time.Hour)
		return &t, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		var v errores.Validacion
		v.Agregar(campo, "Use AAAA-MM-DD o fecha y hora ISO")
		return nil, v.Err()
	}
	return &t, nil
}
