package mysql_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"magui/internal/negocio"
	"magui/internal/negocio/application"
	"magui/internal/negocio/domain"
	"magui/internal/platform/db"
)

// Necesita MAGUI_TEST_MYSQL_DSN (ej. usuario:clave@tcp(localhost:3306)/).
func baseTemporal(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("MAGUI_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("MAGUI_TEST_MYSQL_DSN no definido")
	}
	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	nombre := fmt.Sprintf("magui_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + nombre + " CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec("DROP DATABASE " + nombre); admin.Close() })
	conn, err := db.OpenMySQL(context.Background(), dsn+nombre+"?parseTime=true&loc=UTC")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	m, _ := db.NewMigrator(conn, db.MySQL)
	if err := m.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	return conn
}

var png1x1 = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func TestNegocioMySQL(t *testing.T) {
	conn := baseTemporal(t)
	ctx := context.Background()
	svc := negocio.NuevoModuloMySQL(conn, slog.New(slog.NewTextHandler(io.Discard, nil))).Servicio

	n, err := svc.Obtener(ctx)
	if err != nil || n.Nombre != "Magui" || n.TieneLogo || n.Ticket.Habilitado || n.Ticket.AnchoPapel != 80 || *n.Ticket.Pie != "¡Gracias por su compra!" {
		t.Fatalf("por defecto = %+v (%v)", n, err)
	}

	datos := "Av. Siempreviva 742\nCUIT 20-12345678-9"
	n, err = svc.Guardar(ctx, application.ComandoNegocio{Nombre: "Perfumería Magui", Ticket: application.TicketDTO{
		Habilitado: true, ImprimirAlCobrar: true, AnchoPapel: 58, MostrarLogo: false, Datos: &datos}})
	if err != nil {
		t.Fatal(err)
	}
	n, _ = svc.Obtener(ctx)
	if n.Nombre != "Perfumería Magui" || !n.Ticket.ImprimirAlCobrar || n.Ticket.MostrarLogo || n.Ticket.AnchoPapel != 58 ||
		*n.Ticket.Datos != datos || n.Ticket.Pie != nil {
		t.Errorf("guardado = %+v", n)
	}

	marca, err := svc.CambiarLogo(ctx, png1x1)
	if err != nil || !marca.TieneLogo || marca.LogoVersion != 1 {
		t.Fatalf("logo: %+v %v", marca, err)
	}
	img, tipo, err := svc.Logo(ctx)
	if err != nil || tipo != "image/png" || !bytes.Equal(img, png1x1) {
		t.Errorf("leer logo: %q %v", tipo, err)
	}
	marca, _ = svc.CambiarLogo(ctx, nil)
	if marca.TieneLogo || marca.LogoVersion != 2 {
		t.Errorf("quitar logo: %+v", marca)
	}
	if _, _, err := svc.Logo(ctx); !errors.Is(err, domain.ErrSinLogo) {
		t.Errorf("sin logo: %v", err)
	}
}
