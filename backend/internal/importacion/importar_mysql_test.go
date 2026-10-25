package importacion

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"magui/internal/platform/db"
)

// Test de integración: necesita MAGUI_TEST_MYSQL_DSN, por ejemplo
//
//	usuario:clave@tcp(localhost:3306)/
//
// Crea una base temporal, la migra, importa y la borra.
func TestImportarMySQL(t *testing.T) {
	dsn := os.Getenv("MAGUI_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("MAGUI_TEST_MYSQL_DSN no definido")
	}
	ctx := context.Background()
	conn := baseTemporal(t, ctx, dsn)

	m, err := db.NewMigrator(conn, db.MySQL)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Up(ctx); err != nil {
		t.Fatalf("migrar: %v", err)
	}

	venc := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	filas := []Fila{
		{NumeroFila: 2, SKU: "P1", Nombre: "Kaiak", Stock: 3, PrecioCentavos: 5920000, Vencimiento: &venc,
			Categorias: [][]string{{"PERFUMERÍA", "Perfumería femenina"}, {"REGALOS"}}, MostrarEnTienda: true},
		{NumeroFila: 3, SKU: "P2", Nombre: "Jabón", Stock: 0, PrecioCentavos: 150050,
			Categorias: [][]string{{"perfumería", "Perfumería masculina"}}, MostrarEnTienda: true},
	}
	opt := Opciones{Empresa: "Natura", Ahora: func() time.Time { return time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC) }}

	res, err := Importar(ctx, conn, filas, opt)
	if err != nil {
		t.Fatal(err)
	}
	want := Resultado{Productos: 2, Categorias: 4, Lotes: 1, Unidades: 3, ConVencimiento: 1}
	if res != want {
		t.Fatalf("resultado = %+v, want %+v", res, want)
	}

	var stock int
	var precio string
	if err := conn.QueryRow(`SELECT s.stock, p.precio_venta FROM producto p JOIN v_stock_producto s ON s.producto_id = p.id WHERE p.sku='P1'`).
		Scan(&stock, &precio); err != nil || stock != 3 || precio != "59200.00" {
		t.Fatalf("P1: stock=%d precio=%s err=%v", stock, precio, err)
	}
	var raices int
	_ = conn.QueryRow(`SELECT COUNT(*) FROM categoria WHERE padre_id IS NULL`).Scan(&raices)
	if raices != 2 { // PERFUMERÍA (sin duplicar por mayúsculas/tildes) y REGALOS
		t.Errorf("categorías raíz = %d, want 2", raices)
	}
	var sinEmpresa int
	_ = conn.QueryRow(`SELECT COUNT(*) FROM producto WHERE empresa_id IS NULL`).Scan(&sinEmpresa)
	if sinEmpresa != 0 {
		t.Errorf("productos sin empresa = %d", sinEmpresa)
	}

	// Idempotente: una segunda importación no duplica nada
	res2, err := Importar(ctx, conn, filas, opt)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Productos != 0 || res2.Omitidos != 2 || res2.Categorias != 0 {
		t.Fatalf("segunda importación = %+v", res2)
	}
}

func baseTemporal(t *testing.T, ctx context.Context, dsn string) *sql.DB {
	t.Helper()
	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	nombre := fmt.Sprintf("magui_test_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+nombre+" CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.ExecContext(context.Background(), "DROP DATABASE "+nombre) })

	conn, err := db.OpenMySQL(ctx, dsn+nombre+"?parseTime=true&loc=UTC&charset=utf8mb4")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}
