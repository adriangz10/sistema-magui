// Comando api: servidor central (nube) del Sistema Magui.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"magui/internal/platform/config"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Los módulos (catálogo, ventas, ...) se cablean acá a partir de la Fase 2.
	router := httpserver.NewRouter(log, cfg.AllowedOrigins, version)

	return httpserver.Run(ctx, log, cfg.HTTPAddr, router, cfg.ShutdownTimeout)
}
