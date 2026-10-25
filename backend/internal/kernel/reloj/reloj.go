// Package reloj abstrae la hora actual para poder fijarla en tests.
package reloj

import "time"

// Reloj es el puerto que usan los casos de uso.
type Reloj interface {
	Ahora() time.Time
}

// Sistema devuelve la hora real en UTC.
type Sistema struct{}

func (Sistema) Ahora() time.Time { return time.Now().UTC() }

// Fijo devuelve siempre la misma hora (para tests).
type Fijo struct{ T time.Time }

func (f Fijo) Ahora() time.Time { return f.T }

// ZonaArgentina es la zona horaria para agrupar ventas "del día".
var ZonaArgentina = cargarZona()

func cargarZona() *time.Location {
	if loc, err := time.LoadLocation("America/Argentina/Buenos_Aires"); err == nil {
		return loc
	}
	// Sin base de zonas horarias (ej. Windows sin tzdata): Argentina no tiene horario de verano.
	return time.FixedZone("ART", -3*60*60)
}
