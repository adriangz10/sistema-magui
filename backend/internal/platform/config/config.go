// Package config lee la configuración desde variables de entorno.
package config

import (
	"fmt"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"magui/internal/platform/clienteip"
)

// Config agrupa toda la configuración del proceso.
type Config struct {
	Env            string // "dev" o "prod"
	HTTPAddr       string
	AllowedOrigins []string
	FrontendDir    string // carpeta del frontend compilado; vacío = la API no sirve la app web
	// ProxiesConfiables: IPs o rangos del proxy HTTPS (Caddy). Solo de ellos se acepta X-Forwarded-For.
	ProxiesConfiables []netip.Prefix
	ShutdownTimeout   time.Duration
	DB                DBConfig
	Auth              AuthConfig
}

// AuthConfig configura las sesiones de la dueña.
type AuthConfig struct {
	Secreto        string        // firma los tokens; mínimo 32 caracteres
	SecretoDeDev   bool          // true si se está usando el secreto fijo de desarrollo
	DuracionSesion time.Duration // cuánto dura una sesión antes de pedir la contraseña de nuevo
}

// secretoDev solo se usa en desarrollo cuando no se definió AUTH_SECRET.
const secretoDev = "solo-desarrollo-no-usar-en-produccion-0000"

// DBConfig son los datos de conexión a MySQL.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// DSN arma la cadena de conexión para el driver de MySQL. Fija por sesión innodb_lock_wait_timeout=10
// (el default de MySQL son 50 s): una espera de bloqueo agotada responde 409 en 10 s, bien dentro del
// tiempo de espera del proxy del escritorio (30 s), que si no registraría la venta offline otra vez.
func (d DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_0900_ai_ci&innodb_lock_wait_timeout=10",
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
		FrontendDir:     get("FRONTEND_DIR", ""),
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

	proxies, err := clienteip.ParsearConfiables(get("TRUSTED_PROXIES", ""))
	if err != nil {
		return Config{}, fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	cfg.ProxiesConfiables = proxies

	dur, err := time.ParseDuration(get("SESSION_DURATION", "720h"))
	if err != nil || dur <= 0 {
		return Config{}, fmt.Errorf("SESSION_DURATION inválido: use por ejemplo 720h")
	}
	cfg.Auth = AuthConfig{Secreto: get("AUTH_SECRET", ""), DuracionSesion: dur}
	if cfg.Auth.Secreto == "" {
		if cfg.Env == "prod" {
			return Config{}, fmt.Errorf("AUTH_SECRET es obligatorio en producción")
		}
		cfg.Auth.Secreto, cfg.Auth.SecretoDeDev = secretoDev, true
	}
	if len(cfg.Auth.Secreto) < 32 {
		return Config{}, fmt.Errorf("AUTH_SECRET debe tener al menos 32 caracteres")
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
