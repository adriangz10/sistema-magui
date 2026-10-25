package domain

import (
	"time"
	"unicode/utf8"

	"magui/internal/kernel/errores"
)

// LongitudMinima de la contraseña.
const LongitudMinima = 8

// LongitudMaxima: bcrypt solo usa los primeros 72 bytes.
const LongitudMaxima = 72

// Credencial es la contraseña única de la dueña.
type Credencial struct {
	Hash          string
	Version       int // aumenta en cada cambio; invalida las sesiones anteriores
	ActualizadaEn time.Time
}

// Errores de negocio del módulo.
var (
	ErrClaveIncorrecta = errores.NoAutorizado("clave_incorrecta", "La contraseña es incorrecta")
	ErrSesionInvalida  = errores.NoAutorizado("sesion_invalida", "Tu sesión expiró. Ingresá de nuevo.")
	ErrSinClave        = errores.Conflicto("clave_no_configurada",
		"Todavía no se configuró la contraseña. Usá el comando admin establecer-clave.")
	ErrDemasiadosIntentos = errores.Demasiados("demasiados_intentos",
		"Demasiados intentos fallidos. Esperá unos minutos e intentá de nuevo.")
)

// ValidarClaveNueva revisa el largo de una contraseña nueva.
func ValidarClaveNueva(campo, clave string) error {
	var v errores.Validacion
	n := utf8.RuneCountInString(clave)
	v.Si(n < LongitudMinima, campo, "Debe tener al menos 8 caracteres")
	v.Si(len(clave) > LongitudMaxima, campo, "Es demasiado larga (máximo 72 caracteres)")
	return v.Err()
}

// SiguienteVersion devuelve la credencial con un hash nuevo y la versión incrementada.
func (c Credencial) SiguienteVersion(hash string, ahora time.Time) Credencial {
	return Credencial{Hash: hash, Version: c.Version + 1, ActualizadaEn: ahora}
}
