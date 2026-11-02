package caja

import (
	"database/sql"
	"log/slog"

	"magui/internal/caja/adapters/http"
	"magui/internal/caja/adapters/mysql"
	"magui/internal/caja/application"
	"magui/internal/kernel/id"
	"magui/internal/kernel/reloj"
	"magui/internal/platform/db"
)

// Modulo agrupa los servicios de caja (gastos y apertura/cierre) y su handler HTTP.
type Modulo struct {
	Servicio *application.Servicio
	Sesiones *application.ServicioSesiones
	Handler  *http.Handler
}

func NuevoModuloMySQL(conn *sql.DB, log *slog.Logger) *Modulo {
	svc := application.NuevoServicio(mysql.Repositorio{DB: conn}, id.UUIDv7{}, reloj.Sistema{})
	ses := application.NuevoServicioSesiones(mysql.Sesiones{DB: conn}, db.NuevoTransactorMySQL(conn), id.UUIDv7{}, reloj.Sistema{})
	return &Modulo{Servicio: svc, Sesiones: ses, Handler: http.NuevoHandler(svc, ses, log)}
}
