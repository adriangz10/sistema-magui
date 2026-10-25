// Comando api: servidor central (nube) del Sistema Magui.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"magui/internal/auth"
	"magui/internal/platform/config"
	"magui/internal/platform/db"
	"magui/internal/platform/httpserver"
	"magui/internal/platform/logger"
)

// version se sobrescribe al compilar: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.Env)
	if cfg.Auth.SecretoDeDev {
		log.Warn("usando AUTH_SECRET de desarrollo: definilo antes de publicar el sistema")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, err := db.OpenMySQL(ctx, cfg.DB.DSN())
	if err != nil {
		return err
	}
	defer conn.Close()

	// El servidor no arranca con un esquema desactualizado.
	m, err := db.NewMigrator(conn, db.MySQL)
	if err != nil {
		return err
	}
	if err := m.Up(ctx); err != nil {
		return fmt.Errorf("migrar esquema: %w", err)
	}

	acceso, err := auth.NuevoModuloMySQL(conn, cfg.Auth.Secreto, cfg.Auth.DuracionSesion, log)
	if err != nil {
		return err
	}

	router := httpserver.NewRouter(log, httpserver.Options{
		AllowedOrigins: cfg.AllowedOrigins,
		Version:        version,
		Checks:         map[string]httpserver.Check{"db": conn.PingContext},
		Autenticacion:  acceso.Handler.RequerirSesion,
	},
		acceso.Handler,
		// Los demás módulos (catálogo, ventas, ...) se agregan acá en las próximas fases.
	)

	return httpserver.Run(ctx, log, cfg.HTTPAddr, router, cfg.ShutdownTimeout)
}
