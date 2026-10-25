package inventario

import (
	"database/sql"
	"log/slog"

	catalogo "magui/internal/catalogo/application"
	enlace "magui/internal/inventario/adapters/catalogo"
	"magui/internal/inventario/adapters/http"
	"magui/internal/inventario/adapters/mysql"
	"magui/internal/inventario/application"
	"magui/internal/kernel/id"
	"magui/internal/kernel/reloj"
	"magui/internal/platform/db"
)

// Modulo agrupa el servicio de inventario y su handler HTTP.
type Modulo struct {
	Servicio *application.Servicio
	Handler  *http.Handler
}

// NuevoModuloMySQL arma el inventario para el servidor central.
func NuevoModuloMySQL(conn *sql.DB, cat *catalogo.Servicio, log *slog.Logger) *Modulo {
	svc := application.NuevoServicio(application.Dependencias{
		Repo:      mysql.Repositorio{DB: conn},
		Productos: enlace.Productos{Catalogo: cat},
		Tx:        db.Transactor{DB: conn},
		IDs:       id.UUIDv7{},
		Reloj:     reloj.Sistema{},
	})
	return &Modulo{Servicio: svc, Handler: http.NuevoHandler(svc, log)}
}
