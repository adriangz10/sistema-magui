package ventas

import (
	"database/sql"
	"log/slog"

	catalogo "magui/internal/catalogo/application"
	inventario "magui/internal/inventario/application"
	"magui/internal/kernel/id"
	"magui/internal/kernel/reloj"
	"magui/internal/platform/db"
	"magui/internal/ventas/adapters/enlaces"
	"magui/internal/ventas/adapters/http"
	"magui/internal/ventas/adapters/mysql"
	"magui/internal/ventas/application"
)

// Modulo agrupa el servicio de ventas y su handler HTTP.
type Modulo struct {
	Servicio *application.Servicio
	Handler  *http.Handler
}

// NuevoModuloMySQL arma ventas para el servidor central.
func NuevoModuloMySQL(conn *sql.DB, cat *catalogo.Servicio, inv *inventario.Servicio, log *slog.Logger) *Modulo {
	svc := application.NuevoServicio(application.Dependencias{
		Repo:       mysql.Repositorio{DB: conn},
		Catalogo:   enlaces.Catalogo{Svc: cat},
		Inventario: enlaces.Inventario{Svc: inv},
		Tx:         db.NuevoTransactorMySQL(conn),
		IDs:        id.UUIDv7{},
		Reloj:      reloj.Sistema{},
	})
	return &Modulo{Servicio: svc, Handler: http.NuevoHandler(svc, log)}
}
