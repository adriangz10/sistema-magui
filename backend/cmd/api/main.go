// Comando api: servidor central (nube) del Sistema Magui.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // zonas horarias dentro del binario (la imagen mínima no las trae)

	"magui/internal/auth"
	"magui/internal/caja"
	"magui/internal/catalogo"
	"magui/internal/compras"
	"magui/internal/estadisticas"
	"magui/internal/inventario"
	"magui/internal/negocio"
	"magui/internal/platform/config"
	"magui/internal/platform/db"
	"magui/internal/platform/httpserver"
	"magui/internal/platform/logger"
	"magui/internal/platform/web"
	sincro "magui/internal/sync"
	"magui/internal/ventas"
)

// version se sobrescribe al compilar: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	// "api salud": chequeo de salud para Docker (la imagen no trae curl ni wget).
	if len(os.Args) > 1 && os.Args[1] == "salud" {
		os.Exit(salud())
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// salud pide /api/health al propio servidor: 0 si responde bien, 1 si no.
func salud() int {
	puerto := "8080"
	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		if _, p, err := net.SplitHostPort(addr); err == nil && p != "" {
			puerto = p
		}
	}
	c := http.Client{Timeout: 3 * time.Second}
	res, err := c.Get("http://127.0.0.1:" + puerto + "/api/health")
	if err != nil {
		fmt.Fprintln(os.Stderr, "salud:", err)
		return 1
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "salud: estado", res.StatusCode)
		return 1
	}
	return 0
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

	cat := catalogo.NuevoModuloMySQL(conn, log)
	inv := inventario.NuevoModuloMySQL(conn, cat.Servicio, log)
	cj := caja.NuevoModuloMySQL(conn, log)
	// Sin caja abierta no se vende desde la web ni el celular (US-002).
	ven := ventas.NuevoModuloMySQL(conn, cat.Servicio, inv.Servicio, cj.Sesiones, log)

	var frontend http.Handler
	if cfg.FrontendDir != "" {
		if info, err := os.Stat(cfg.FrontendDir); err != nil || !info.IsDir() {
			return fmt.Errorf("FRONTEND_DIR %q no es una carpeta: compilá el frontend con npm run build", cfg.FrontendDir)
		}
		frontend = web.SPA(os.DirFS(cfg.FrontendDir))
		log.Info("sirviendo la app web", "carpeta", cfg.FrontendDir)
	}

	router := httpserver.NewRouter(log, httpserver.Options{
		AllowedOrigins:    cfg.AllowedOrigins,
		Frontend:          frontend,
		ProxiesConfiables: cfg.ProxiesConfiables,
		Version:           version,
		Checks:            map[string]httpserver.Check{"db": conn.PingContext},
		Autenticacion:     acceso.Handler.RequerirSesion,
	},
		acceso.Handler,
		cat.Handler,
		inv.Handler,
		compras.NuevoModuloMySQL(conn, cat.Servicio, inv.Servicio, log).Handler,
		ven.Handler,
		sincro.NuevoModulo(cat.Servicio, ven.Servicio, log).Handler,
		cj.Handler,
		negocio.NuevoModuloMySQL(conn, log).Handler,
		estadisticas.NuevoModuloMySQL(conn, log).Handler,
	)

	return httpserver.Run(ctx, log, cfg.HTTPAddr, web.Seguridad(router), cfg.ShutdownTimeout)
}
