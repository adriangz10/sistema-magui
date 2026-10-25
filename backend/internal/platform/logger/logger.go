// Package logger crea el logger estructurado del sistema.
package logger

import (
	"log/slog"
	"os"
)

// New devuelve texto legible en desarrollo y JSON en producción.
func New(env string) *slog.Logger {
	if env == "prod" {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}
