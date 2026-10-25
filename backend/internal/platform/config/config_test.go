package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "dev" || cfg.HTTPAddr != ":8080" {
		t.Fatalf("defaults inesperados: %+v", cfg)
	}
}

func TestLoadInvalidEnv(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	if _, err := Load(); err == nil {
		t.Fatal("se esperaba error con APP_ENV inválido")
	}
}

func TestAuthSecreto(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	t.Setenv("AUTH_SECRET", "")
	cfg, err := Load()
	if err != nil || !cfg.Auth.SecretoDeDev || cfg.Auth.DuracionSesion.Hours() != 720 {
		t.Fatalf("dev sin secreto: %+v, %v", cfg.Auth, err)
	}

	t.Setenv("APP_ENV", "prod")
	if _, err := Load(); err == nil {
		t.Error("prod sin AUTH_SECRET debería fallar")
	}
	t.Setenv("AUTH_SECRET", "corto")
	if _, err := Load(); err == nil {
		t.Error("secreto corto debería fallar")
	}
	t.Setenv("AUTH_SECRET", "un-secreto-de-produccion-bien-largo-123456")
	if cfg, err := Load(); err != nil || cfg.Auth.SecretoDeDev {
		t.Errorf("prod con secreto: %+v, %v", cfg.Auth, err)
	}
}

func TestAllowedOriginsList(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", " http://a.com , ,http://b.com")
	cfg, _ := Load()
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[1] != "http://b.com" {
		t.Fatalf("orígenes: %v", cfg.AllowedOrigins)
	}
}
