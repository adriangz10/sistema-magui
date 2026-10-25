// Package config lee la configuración desde variables de entorno.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config agrupa toda la configuración del proceso.
type Config struct {
	Env             string // "dev" o "prod"
	HTTPAddr        string
	AllowedOrigins  []string
	ShutdownTimeout time.Duration
	DB              DBConfig
}

// DBConfig son los datos de conexión a MySQL.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// DSN arma la cadena de conexión para el driver de MySQL.
func (d DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_0900_ai_ci",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

// Load lee las variables de entorno y aplica valores por defecto de desarrollo.
// Si existe un archivo .env en el directorio actual, lo carga primero
// (las variables ya definidas en el sistema tienen prioridad).
func Load() (Config, error) {
	_ = godotenv.Load()
	cfg := Config{
		Env:             get("APP_ENV", "dev"),
		HTTPAddr:        get("HTTP_ADDR", ":8080"),
		AllowedOrigins:  splitList(get("ALLOWED_ORIGINS", "http://localhost:4200")),
		ShutdownTimeout: 10 * time.Second,
		DB: DBConfig{
			Host:     get("DB_HOST", "localhost"),
			Port:     get("DB_PORT", "3306"),
			User:     get("DB_USER", ""),
			Password: get("DB_PASSWORD", ""),
			Name:     get("DB_NAME", "magui"),
		},
	}
	if cfg.Env != "dev" && cfg.Env != "prod" {
		return Config{}, fmt.Errorf("APP_ENV inválido %q: use dev o prod", cfg.Env)
	}
	return cfg, nil
}

func get(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func splitList(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
