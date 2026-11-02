// Package negocio: nombre y logo del comercio y configuración del ticket de cada venta (US-003).
package negocio

import (
	"database/sql"
	"log/slog"

	"magui/internal/negocio/adapters/http"
	"magui/internal/negocio/adapters/mysql"
	"magui/internal/negocio/application"
)

// Modulo agrupa los datos del negocio y su handler HTTP.
type Modulo struct {
	Servicio *application.Servicio
	Handler  *http.Handler
}

func NuevoModuloMySQL(conn *sql.DB, log *slog.Logger) *Modulo {
	svc := application.NuevoServicio(mysql.Repositorio{DB: conn})
	return &Modulo{Servicio: svc, Handler: http.NuevoHandler(svc, log)}
}
