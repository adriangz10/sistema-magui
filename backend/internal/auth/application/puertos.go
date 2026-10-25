package application

import (
	"context"
	"time"

	"magui/internal/auth/domain"
)

// RepositorioCredencial guarda la contraseña de la dueña.
type RepositorioCredencial interface {
	// Obtener devuelve existe = false si todavía no se configuró.
	Obtener(ctx context.Context) (cred domain.Credencial, existe bool, err error)
	Guardar(ctx context.Context, cred domain.Credencial) error
}

// Hasher cifra y verifica contraseñas.
type Hasher interface {
	Hashear(clave string) (string, error)
	Coincide(hash, clave string) bool
}

// Sesion es el token que el frontend envía en cada solicitud.
type Sesion struct {
	Token    string    `json:"token"`
	ExpiraEn time.Time `json:"expiraEn"`
}

// Tokens emite y lee tokens firmados que llevan la versión de la credencial.
type Tokens interface {
	Emitir(version int, ahora time.Time) (Sesion, error)
	// Leer devuelve la versión de la credencial con la que se emitió; error si es inválido o venció.
	Leer(token string, ahora time.Time) (version int, err error)
}

// Limitador frena los intentos de adivinar la contraseña.
type Limitador interface {
	Permitido(origen string, ahora time.Time) bool
	RegistrarFallo(origen string, ahora time.Time)
	Reiniciar(origen string)
}
