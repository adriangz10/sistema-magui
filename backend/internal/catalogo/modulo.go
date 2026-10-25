package catalogo

import (
	"database/sql"
	"log/slog"

	"magui/internal/catalogo/adapters/http"
	"magui/internal/catalogo/adapters/mysql"
	"magui/internal/catalogo/application"
	"magui/internal/kernel/id"
	"magui/internal/kernel/reloj"
	"magui/internal/platform/db"
)

// Modulo agrupa el servicio del catálogo y su handler HTTP.
type Modulo struct {
	Servicio *application.Servicio
	Handler  *http.Handler
}

// NuevoModuloMySQL arma el catálogo para el servidor central.
func NuevoModuloMySQL(conn *sql.DB, log *slog.Logger) *Modulo {
	svc := application.NuevoServicio(application.Dependencias{
		Empresas:   mysql.Empresas{DB: conn},
		Categorias: mysql.Categorias{DB: conn},
		Productos:  mysql.Productos{DB: conn},
		Tx:         db.Transactor{DB: conn},
		IDs:        id.UUIDv7{},
		Reloj:      reloj.Sistema{},
	})
	return &Modulo{Servicio: svc, Handler: http.NuevoHandler(svc, log)}
}
