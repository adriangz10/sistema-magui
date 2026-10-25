// Package fecha representa un día del calendario sin hora (ej. fecha de vencimiento).
// En JSON y en la base se escribe "2006-01-02".
package fecha

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const formato = time.DateOnly

// Fecha es un día del calendario. El valor cero significa "sin fecha".
type Fecha struct{ t time.Time } // siempre medianoche UTC

// Nueva crea una fecha.
func Nueva(anio int, mes time.Month, dia int) Fecha {
	return Fecha{time.Date(anio, mes, dia, 0, 0, 0, 0, time.UTC)}
}

// Parse interpreta "2026-09-30".
func Parse(s string) (Fecha, error) {
	t, err := time.Parse(formato, s)
	if err != nil {
		return Fecha{}, fmt.Errorf("fecha inválida %q: use AAAA-MM-DD", s)
	}
	return Fecha{t}, nil
}

// DelInstante devuelve el día al que corresponde t en la zona indicada.
func DelInstante(t time.Time, zona *time.Location) Fecha {
	l := t.In(zona)
	return Nueva(l.Year(), l.Month(), l.Day())
}

func (f Fecha) IsZero() bool    { return f.t.IsZero() }
func (f Fecha) String() string  { return f.t.Format(formato) }
func (f Fecha) Time() time.Time { return f.t }

// SumarDias devuelve la fecha n días después (o antes, si n es negativo).
func (f Fecha) SumarDias(n int) Fecha { return Fecha{f.t.AddDate(0, 0, n)} }

// DiasHasta devuelve cuántos días faltan desde f hasta otra (negativo si otra es anterior).
func (f Fecha) DiasHasta(otra Fecha) int { return int(otra.t.Sub(f.t).Hours() / 24) }

func (f Fecha) Antes(o Fecha) bool   { return f.t.Before(o.t) }
func (f Fecha) Despues(o Fecha) bool { return f.t.After(o.t) }

// InicioEn devuelve el instante en que empieza el día en la zona indicada.
func (f Fecha) InicioEn(zona *time.Location) time.Time {
	return time.Date(f.t.Year(), f.t.Month(), f.t.Day(), 0, 0, 0, 0, zona)
}

func (f Fecha) MarshalJSON() ([]byte, error) {
	if f.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(f.String())
}

func (f *Fecha) UnmarshalJSON(b []byte) error {
	var s *string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("fecha inválida: use AAAA-MM-DD")
	}
	if s == nil || *s == "" {
		*f = Fecha{}
		return nil
	}
	v, err := Parse(*s)
	if err != nil {
		return err
	}
	*f = v
	return nil
}

// Scan lee DATE de MySQL (time.Time) o TEXT de SQLite.
func (f *Fecha) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*f = Fecha{}
		return nil
	case time.Time:
		*f = Nueva(v.Year(), v.Month(), v.Day())
		return nil
	case []byte:
		return f.scanTexto(string(v))
	case string:
		return f.scanTexto(v)
	default:
		return fmt.Errorf("fecha: tipo de columna no soportado %T", src)
	}
}

func (f *Fecha) scanTexto(s string) error {
	if len(s) > 10 {
		s = s[:10]
	}
	v, err := Parse(s)
	if err != nil {
		return err
	}
	*f = v
	return nil
}

// Value guarda "2006-01-02", válido tanto para DATE de MySQL como TEXT de SQLite.
func (f Fecha) Value() (driver.Value, error) {
	if f.IsZero() {
		return nil, nil
	}
	return f.String(), nil
}
