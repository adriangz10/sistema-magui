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

func TestAllowedOriginsList(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", " http://a.com , ,http://b.com")
	cfg, _ := Load()
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[1] != "http://b.com" {
		t.Fatalf("orígenes: %v", cfg.AllowedOrigins)
	}
}
