package estadisticas

import (
	"database/sql"
	"log/slog"

	"magui/internal/estadisticas/adapters/http"
	"magui/internal/estadisticas/adapters/mysql"
	"magui/internal/estadisticas/application"
	"magui/internal/kernel/reloj"
)

// Modulo agrupa el servicio de estadísticas y su handler HTTP.
type Modulo struct {
	Servicio *application.Servicio
	Handler  *http.Handler
}

func NuevoModuloMySQL(conn *sql.DB, log *slog.Logger) *Modulo {
	svc := application.NuevoServicio(mysql.Consultas{DB: conn}, reloj.Sistema{}, reloj.ZonaArgentina)
	return &Modulo{Servicio: svc, Handler: http.NuevoHandler(svc, log)}
}
