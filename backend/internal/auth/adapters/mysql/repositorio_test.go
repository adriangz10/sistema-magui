package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"magui/internal/auth/domain"
	"magui/internal/platform/db"
)

// Integración: necesita MAGUI_TEST_MYSQL_DSN (ej. usuario:clave@tcp(localhost:3306)/).
func TestRepositorioMySQL(t *testing.T) {
	dsn := os.Getenv("MAGUI_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("MAGUI_TEST_MYSQL_DSN no definido")
	}
	ctx := context.Background()
	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	nombre := fmt.Sprintf("magui_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + nombre); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec("DROP DATABASE " + nombre)

	conn, err := db.OpenMySQL(ctx, dsn+nombre+"?parseTime=true&loc=UTC")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	m, _ := db.NewMigrator(conn, db.MySQL)
	if err := m.Up(ctx); err != nil {
		t.Fatal(err)
	}

	repo := Repositorio{DB: conn}
	if _, existe, err := repo.Obtener(ctx); err != nil || existe {
		t.Fatalf("base nueva: existe=%v err=%v", existe, err)
	}
	ahora := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	for v := 1; v <= 2; v++ { // insertar y luego actualizar
		if err := repo.Guardar(ctx, domain.Credencial{Hash: fmt.Sprintf("h%d", v), Version: v, ActualizadaEn: ahora}); err != nil {
			t.Fatal(err)
		}
	}
	c, existe, err := repo.Obtener(ctx)
	if err != nil || !existe || c.Hash != "h2" || c.Version != 2 || !c.ActualizadaEn.Equal(ahora) {
		t.Fatalf("credencial = %+v existe=%v err=%v", c, existe, err)
	}
	var filas int
	_ = conn.QueryRow(`SELECT COUNT(*) FROM credencial_acceso`).Scan(&filas)
	if filas != 1 {
		t.Fatalf("debería haber una sola fila, hay %d", filas)
	}
}
