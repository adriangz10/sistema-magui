package caja

import (
	"database/sql"
	"log/slog"

	"magui/internal/caja/adapters/http"
	"magui/internal/caja/adapters/mysql"
	"magui/internal/caja/application"
	"magui/internal/kernel/id"
	"magui/internal/kernel/reloj"
)

// Modulo agrupa el servicio de caja y su handler HTTP.
type Modulo struct {
	Servicio *application.Servicio
	Handler  *http.Handler
}

func NuevoModuloMySQL(conn *sql.DB, log *slog.Logger) *Modulo {
	svc := application.NuevoServicio(mysql.Repositorio{DB: conn}, id.UUIDv7{}, reloj.Sistema{})
	return &Modulo{Servicio: svc, Handler: http.NuevoHandler(svc, log)}
}
