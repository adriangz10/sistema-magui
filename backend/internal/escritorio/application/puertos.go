// Package application contiene los casos de uso del escritorio y sus puertos.
package application

import (
	"context"
	"errors"
	"time"

	sincro "magui/internal/sync/application"
)

// Errores del cliente del servidor.
var (
	// ErrSinConexion: no se pudo llegar al servidor (sin internet, servidor caído, 502/503/504).
	ErrSinConexion = errors.New("sin conexión con el servidor")
	// ErrSesionVencida: el servidor rechazó el token; hay que ingresar de nuevo con conexión.
	ErrSesionVencida = errors.New("la sesión con el servidor venció")
)

// Servidor es el puerto hacia el servidor central.
type Servidor interface {
	CopiaCatalogo(ctx context.Context, token string) (sincro.CopiaCatalogo, error)
	EnviarVentas(ctx context.Context, token string, ventas []sincro.VentaOffline) (sincro.RespuestaEnvio, error)
}

// Cola guarda las ventas locales por subir y la copia del catálogo.
type Cola interface {
	// Pendientes devuelve las ventas por subir, de la más vieja a la más nueva.
	Pendientes(ctx context.Context, limite int) ([]sincro.VentaOffline, error)
	MarcarEnviadas(ctx context.Context, ids []string, en time.Time) error
	MarcarRechazada(ctx context.Context, id, motivo string) error
	// Contar devuelve cuántas ventas faltan subir y cuántas rechazó el servidor.
	Contar(ctx context.Context) (pendientes, rechazadas int, err error)
	// ReemplazarCopia guarda la copia nueva del catálogo; las ventas ya enviadas quedan reflejadas en ella.
	ReemplazarCopia(ctx context.Context, c sincro.CopiaCatalogo) error
	// HayCopia indica si ya se bajó alguna vez el catálogo.
	HayCopia(ctx context.Context) (bool, time.Time, error)
	// Productos devuelve la copia con el stock local (restando lo vendido que la copia no refleja).
	Productos(ctx context.Context) ([]sincro.ProductoCopia, error)
}

// Sesion es lo que se recuerda del último ingreso con conexión.
type Sesion struct {
	Token    string    `json:"token"`
	ExpiraEn time.Time `json:"expiraEn"`
	// HashClave permite ingresar sin conexión con la última contraseña usada en esta PC.
	HashClave string `json:"hashClave,omitempty"`
}

// Sesiones persiste la sesión con el servidor.
type Sesiones interface {
	Leer(ctx context.Context) (Sesion, error) // vacía si nunca se ingresó
	Guardar(ctx context.Context, s Sesion) error
}

// Claves cifra y compara contraseñas (bcrypt).
type Claves interface {
	Cifrar(clave string) (string, error)
	Coincide(hash, clave string) bool
}
