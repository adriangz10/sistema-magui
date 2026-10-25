package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"magui/internal/kernel/errores"
)

// Empresa es la marca a la que se le compra la mercadería (Natura, Avon, ...).
type Empresa struct {
	ID        string
	Nombre    string
	Activo    bool
	Version   int
	UpdatedAt time.Time
}

// NuevaEmpresa valida y crea una empresa activa.
func NuevaEmpresa(id, nombre string, ahora time.Time) (Empresa, error) {
	e := Empresa{ID: id, Activo: true, Version: 1, UpdatedAt: ahora}
	if err := e.Modificar(nombre, true, ahora); err != nil {
		return Empresa{}, err
	}
	e.Version = 1
	return e, nil
}

// Modificar cambia nombre y estado e incrementa la versión.
func (e *Empresa) Modificar(nombre string, activo bool, ahora time.Time) error {
	nombre = LimpiarTexto(nombre)
	var v errores.Validacion
	v.Si(nombre == "", "nombre", "El nombre es obligatorio")
	v.Si(utf8.RuneCountInString(nombre) > 100, "nombre", "Máximo 100 caracteres")
	if err := v.Err(); err != nil {
		return err
	}
	e.Nombre, e.Activo, e.UpdatedAt = nombre, activo, ahora
	e.Version++
	return nil
}

// LimpiarTexto quita espacios al inicio, al final y repetidos.
func LimpiarTexto(s string) string { return strings.Join(strings.Fields(s), " ") }
