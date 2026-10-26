// Comando escritorio corre el núcleo de la app de escritorio en el navegador, para desarrollo
// y pruebas (la app instalable con Wails usa el mismo núcleo, sin abrir un puerto).
//
//	go run ./cmd/escritorio -servidor http://localhost:8080 -frontend ../frontend/dist/frontend/browser
//
// Abre http://localhost:4300. Cortar el servidor simula quedarse sin internet.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"magui/internal/escritorio"
	"magui/internal/platform/web"
)

func main() {
	servidor := flag.String("servidor", "http://localhost:8080", "dirección del servidor central")
	datos := flag.String("datos", "escritorio.db", "archivo de la base local")
	frontend := flag.String("frontend", "", "carpeta de la app compilada (npm run build)")
	addr := flag.String("addr", "127.0.0.1:4300", "dirección local donde escuchar")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := ejecutar(log, *servidor, *datos, *frontend, *addr); err != nil {
		log.Error("escritorio", "error", err)
		os.Exit(1)
	}
}

func ejecutar(log *slog.Logger, servidor, datos, frontend, addr string) error {
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelar()

	cfg := escritorio.Config{Servidor: servidor, Datos: datos, Log: log}
	if frontend != "" {
		if info, err := os.Stat(frontend); err != nil || !info.IsDir() {
			return fmt.Errorf("-frontend %q no es una carpeta", frontend)
		}
		cfg.Frontend = web.SPA(os.DirFS(frontend))
	}
	app, err := escritorio.Abrir(ctx, cfg)
	if err != nil {
		return err
	}
	defer app.Cerrar()
	go app.Ejecutar(ctx)

	srv := &http.Server{Addr: addr, Handler: web.Seguridad(app.Handler), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		apagar, listo := context.WithTimeout(context.Background(), 5*time.Second)
		defer listo()
		_ = srv.Shutdown(apagar)
	}()
	log.Info("escritorio listo", "abrir", "http://"+addr, "servidor", servidor, "datos", datos)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
