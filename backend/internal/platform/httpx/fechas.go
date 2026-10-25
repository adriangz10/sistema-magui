package httpx

import (
	"net/http"
	"time"

	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/reloj"
)

// RangoFechas lee ?desde=AAAA-MM-DD&hasta=AAAA-MM-DD como días de Argentina.
// Devuelve el inicio de "desde" y el inicio del día siguiente a "hasta" (hasta inclusivo).
func RangoFechas(r *http.Request) (desde, hasta *time.Time, err error) {
	var v errores.Validacion
	if s := r.URL.Query().Get("desde"); s != "" {
		if f, e := fecha.Parse(s); e != nil {
			v.Agregar("desde", "Use el formato AAAA-MM-DD")
		} else {
			t := f.InicioEn(reloj.ZonaArgentina)
			desde = &t
		}
	}
	if s := r.URL.Query().Get("hasta"); s != "" {
		if f, e := fecha.Parse(s); e != nil {
			v.Agregar("hasta", "Use el formato AAAA-MM-DD")
		} else {
			t := f.SumarDias(1).InicioEn(reloj.ZonaArgentina)
			hasta = &t
		}
	}
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
