package auth

import (
	"database/sql"
	"log/slog"
	"time"

	"magui/internal/auth/adapters/http"
	"magui/internal/auth/adapters/mysql"
	"magui/internal/auth/adapters/seguridad"
	"magui/internal/auth/application"
	"magui/internal/kernel/reloj"
)

// Intentos fallidos permitidos antes de bloquear un origen durante VentanaBloqueo.
const (
	MaxIntentosFallidos = 5
	VentanaBloqueo      = 15 * time.Minute
)

// Modulo agrupa el servicio y su handler HTTP ya cableados para MySQL.
type Modulo struct {
	Servicio *application.Servicio
	Handler  *http.Handler
}

// NuevoModuloMySQL arma el módulo de acceso para el servidor central.
func NuevoModuloMySQL(conn *sql.DB, secreto string, duracion time.Duration, log *slog.Logger) (*Modulo, error) {
	tokens, err := seguridad.NuevoJWT(secreto, duracion)
	if err != nil {
		return nil, err
	}
	svc := application.NuevoServicio(
		mysql.Repositorio{DB: conn},
		seguridad.Bcrypt{},
		tokens,
		seguridad.NuevoLimitador(MaxIntentosFallidos, VentanaBloqueo),
		reloj.Sistema{},
	)
	return &Modulo{Servicio: svc, Handler: http.NuevoHandler(svc, log)}, nil
}
