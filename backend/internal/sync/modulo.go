package sync

import (
	"log/slog"

	catalogo "magui/internal/catalogo/application"
	"magui/internal/kernel/reloj"
	"magui/internal/sync/adapters/enlaces"
	"magui/internal/sync/adapters/http"
	"magui/internal/sync/application"
	ventas "magui/internal/ventas/application"
)

// Modulo agrupa la sincronización del lado del servidor.
type Modulo struct {
	Servicio *application.Servicio
	Handler  *http.Handler
}

// NuevoModulo arma la sincronización sobre los servicios de catálogo y ventas.
func NuevoModulo(cat *catalogo.Servicio, ven *ventas.Servicio, log *slog.Logger) *Modulo {
	svc := application.NuevoServicio(enlaces.Catalogo{Svc: cat}, enlaces.Ventas{Svc: ven}, reloj.Sistema{})
	return &Modulo{Servicio: svc, Handler: http.NuevoHandler(svc, log)}
}
