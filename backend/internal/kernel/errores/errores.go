// Package errores define los errores de negocio comunes. Los casos de uso los devuelven
// y el adaptador HTTP los traduce a códigos de estado, sin que el dominio conozca HTTP.
package errores

import (
	"errors"
	"fmt"
)

// Tipo clasifica el error.
type Tipo int

const (
	TipoInvalido     Tipo = iota + 1 // datos de entrada incorrectos
	TipoNoEncontrado                 // el recurso no existe
	TipoConflicto                    // choca con el estado actual (duplicado, sin stock, ...)
	TipoNoAutorizado                 // falta autenticación o es incorrecta
	TipoDemasiados                   // demasiados intentos
)

// Error es un error de negocio con código estable para el frontend.
type Error struct {
	Tipo    Tipo
	Codigo  string            // ej. "stock_insuficiente" (estable, lo usa el frontend)
	Mensaje string            // texto para mostrar
	Campos  map[string]string // errores por campo de formulario
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Codigo, e.Mensaje) }

// Is permite comparar con errors.Is por código: errors.Is(err, ErrX).
func (e *Error) Is(target error) bool {
	var t *Error
	return errors.As(target, &t) && t.Codigo == e.Codigo
}

func Invalido(codigo, mensaje string) *Error {
	return &Error{Tipo: TipoInvalido, Codigo: codigo, Mensaje: mensaje}
}

func NoEncontrado(codigo, mensaje string) *Error {
	return &Error{Tipo: TipoNoEncontrado, Codigo: codigo, Mensaje: mensaje}
}

func Conflicto(codigo, mensaje string) *Error {
	return &Error{Tipo: TipoConflicto, Codigo: codigo, Mensaje: mensaje}
}

func NoAutorizado(codigo, mensaje string) *Error {
	return &Error{Tipo: TipoNoAutorizado, Codigo: codigo, Mensaje: mensaje}
}

func Demasiados(codigo, mensaje string) *Error {
	return &Error{Tipo: TipoDemasiados, Codigo: codigo, Mensaje: mensaje}
}

// Validacion junta errores por campo y produce un único error "datos_invalidos".
type Validacion struct{ campos map[string]string }

// Agregar registra un problema en un campo (se queda con el primero de cada campo).
func (v *Validacion) Agregar(campo, mensaje string) {
	if v.campos == nil {
		v.campos = map[string]string{}
	}
	if _, ok := v.campos[campo]; !ok {
		v.campos[campo] = mensaje
	}
}

// Si agrega el mensaje cuando la condición se cumple.
func (v *Validacion) Si(cond bool, campo, mensaje string) {
	if cond {
		v.Agregar(campo, mensaje)
	}
}

// Err devuelve nil si no hubo problemas.
func (v *Validacion) Err() error {
	if len(v.campos) == 0 {
		return nil
	}
	return &Error{Tipo: TipoInvalido, Codigo: "datos_invalidos", Mensaje: "Revisá los datos ingresados", Campos: v.campos}
}

// Como extrae el *Error de una cadena de errores.
func Como(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
