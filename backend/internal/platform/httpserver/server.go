// Package httpserver arma el router HTTP común y controla el arranque/apagado.
package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"magui/internal/platform/httpx"
)

// Module es lo que cada módulo (catálogo, ventas, ...) implementa para colgar sus rutas
// protegidas (requieren sesión).
type Module interface {
	Routes(r chi.Router)
}

// PublicModule es opcional: rutas accesibles sin sesión (ej. login).
type PublicModule interface {
	PublicRoutes(r chi.Router)
}

// Check verifica una dependencia (ej. la base de datos) para /health.
type Check func(ctx context.Context) error

// Options configura el router.
type Options struct {
	AllowedOrigins []string
	Version        string
	Checks         map[string]Check
	// Autenticacion protege las rutas de los módulos. nil = sin protección (solo tests/escritorio).
	Autenticacion func(http.Handler) http.Handler
	// Frontend sirve la app web en todo lo que no es /api (opcional).
	Frontend http.Handler
}

// NewRouter crea el router con middlewares comunes y monta todo bajo /api.
func NewRouter(log *slog.Logger, opt Options, modules ...Module) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger(log))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   opt.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false, // la sesión viaja en el encabezado Authorization, no en cookies
		MaxAge:           300,
	}))

	r.Route("/api", func(api chi.Router) {
		api.Get("/health", health(opt.Version, opt.Checks))
		for _, m := range modules {
			if pm, ok := m.(PublicModule); ok {
				pm.PublicRoutes(api)
			}
		}
		api.Group(func(prot chi.Router) {
			if opt.Autenticacion != nil {
				prot.Use(opt.Autenticacion)
			}
			for _, m := range modules {
				m.Routes(prot)
			}
		})
		api.NotFound(func(w http.ResponseWriter, _ *http.Request) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{
				"error": map[string]string{"code": "ruta_no_encontrada", "message": "La ruta no existe"},
			})
		})
	})
	if opt.Frontend != nil {
		r.Handle("/*", opt.Frontend)
	}
	return r
}

func health(version string, checks map[string]Check) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		estado, code := "ok", http.StatusOK
		resultados := map[string]string{}
		for nombre, check := range checks {
			if err := check(ctx); err != nil {
				resultados[nombre] = "error"
				estado, code = "degradado", http.StatusServiceUnavailable
				continue
			}
			resultados[nombre] = "ok"
		}
		httpx.JSON(w, code, map[string]any{"status": estado, "version": version, "checks": resultados})
	}
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("http",
				"method", r.Method, "path", r.URL.Path, "status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()))
		})
	}
}

// Run levanta el servidor y lo apaga ordenadamente cuando se cancela ctx.
func Run(ctx context.Context, log *slog.Logger, addr string, h http.Handler, shutdownTimeout time.Duration) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Info("servidor HTTP escuchando", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("apagando servidor HTTP")
		sctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(sctx)
	}
}
