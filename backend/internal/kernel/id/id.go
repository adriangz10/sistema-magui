// Package id genera identificadores UUID v7 (ordenables por fecha de creación).
// Se generan en la aplicación, no en la base, para poder crear registros sin conexión.
package id

import "github.com/google/uuid"

// Generador es el puerto que usan los casos de uso (en tests se reemplaza por uno fijo).
type Generador interface {
	Nuevo() string
}

// UUIDv7 es el generador real.
type UUIDv7 struct{}

func (UUIDv7) Nuevo() string { return uuid.Must(uuid.NewV7()).String() }

// Nuevo es un atajo al generador real.
func Nuevo() string { return UUIDv7{}.Nuevo() }

// Valido indica si el texto es un UUID bien formado.
func Valido(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil && len(s) == 36
}
