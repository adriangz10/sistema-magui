package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigracionesSQLite(t *testing.T) {
	ctx := context.Background()
	conn, err := OpenSQLite(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	m, err := NewMigrator(conn, SQLite)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Up(ctx); err != nil {
		t.Fatalf("up: %v", err)
	}
	tablas := []string{"empresa", "categoria", "producto", "producto_categoria", "combo_item",
		"precio_historial", "compra", "compra_item", "lote", "venta", "venta_item", "venta_pago",
		"compra_edicion", "movimiento_stock", "gasto_caja", "configuracion", "v_stock_lote", "v_stock_producto"}
	for _, tb := range tablas {
		var n int
		if err := conn.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name = ?`, tb).Scan(&n); err != nil || n != 1 {
			t.Errorf("falta %s (err=%v)", tb, err)
		}
	}

	// Reglas básicas del esquema
	mustExec(t, conn, `INSERT INTO producto (id, sku, nombre, precio_venta) VALUES ('p1','SKU1','Crema',1000)`)
	mustExec(t, conn, `INSERT INTO lote (id, producto_id) VALUES ('l1','p1')`)
	mustExec(t, conn, `INSERT INTO movimiento_stock (id, producto_id, lote_id, cantidad, tipo, fecha) VALUES ('m1','p1','l1',5,'COMPRA','2026-01-01')`)
	mustExec(t, conn, `INSERT INTO movimiento_stock (id, producto_id, lote_id, cantidad, tipo, fecha) VALUES ('m2','p1','l1',-2,'VENTA','2026-01-02')`)
	var stock int
	if err := conn.QueryRow(`SELECT stock FROM v_stock_producto WHERE producto_id='p1'`).Scan(&stock); err != nil || stock != 3 {
		t.Fatalf("stock = %d, err = %v; se esperaba 3", stock, err)
	}
	if _, err := conn.Exec(`INSERT INTO producto (id, sku, nombre, precio_venta) VALUES ('p2','SKU1','Otro',1)`); err == nil {
		t.Error("se esperaba error por SKU duplicado")
	}
	if _, err := conn.Exec(`INSERT INTO movimiento_stock (id, producto_id, cantidad, tipo, fecha) VALUES ('m3','nope',1,'AJUSTE','2026-01-01')`); err == nil {
		t.Error("se esperaba error de clave foránea (foreign_keys activo)")
	}
	if _, err := conn.Exec(`INSERT INTO venta_pago (id, venta_id, metodo, monto) VALUES ('x','y','CHEQUE',1)`); err == nil {
		t.Error("se esperaba error por método de pago inválido")
	}

	// US-001a: versión de la compra y auditoría de ediciones
	mustExec(t, conn, `INSERT INTO compra (id, fecha, total, created_at, updated_at) VALUES ('c1','2026-01-01',100,'2026-01-01','2026-01-01')`)
	var version int
	if err := conn.QueryRow(`SELECT version FROM compra WHERE id='c1'`).Scan(&version); err != nil || version != 1 {
		t.Errorf("version = %d, err = %v", version, err)
	}
	mustExec(t, conn, `INSERT INTO compra_edicion (id, compra_id, fecha, origen, version, cambios) VALUES ('e1','c1','2026-01-02','WEB',2,'{"v":1}')`)
	if _, err := conn.Exec(`INSERT INTO compra_edicion (id, compra_id, fecha, origen, version, cambios) VALUES ('e2','c1','2026-01-02','WEB',2,'{"v":1}')`); err == nil {
		t.Error("se esperaba error por (compra_id, version) duplicado")
	}
	if _, err := conn.Exec(`INSERT INTO compra_edicion (id, compra_id, fecha, origen, version, cambios) VALUES ('e3','c1','2026-01-02','WEB',3,'x')`); err == nil {
		t.Error("se esperaba error: cambios no es JSON")
	}
	if _, err := conn.Exec(`INSERT INTO compra_edicion (id, compra_id, fecha, origen, version, cambios) VALUES ('e4','c1','2026-01-02','FAX',4,'{}')`); err == nil {
		t.Error("se esperaba error por origen inválido")
	}

	// US-001b: movimiento EDICION_COMPRA (CHECK reconstruido) y vistas intactas
	mustExec(t, conn, `INSERT INTO movimiento_stock (id, producto_id, lote_id, cantidad, tipo, referencia_id, fecha) VALUES ('m4','p1','l1',2,'EDICION_COMPRA','c1','2026-01-03')`)
	mustExec(t, conn, `INSERT INTO movimiento_stock (id, producto_id, lote_id, cantidad, tipo, fecha) VALUES ('m5','p1','l1',-1,'EDICION_COMPRA','2026-01-03')`)
	if _, err := conn.Exec(`INSERT INTO movimiento_stock (id, producto_id, lote_id, cantidad, tipo, fecha) VALUES ('m6','p1','l1',1,'OTRO','2026-01-03')`); err == nil {
		t.Error("se esperaba error por tipo de movimiento inválido")
	}
	if _, err := conn.Exec(`INSERT INTO movimiento_stock (id, producto_id, lote_id, cantidad, tipo, fecha) VALUES ('m7','p1','l1',0,'EDICION_COMPRA','2026-01-03')`); err == nil {
		t.Error("se esperaba error: cantidad cero")
	}
	// Los movimientos de antes de la reconstrucción siguen y las vistas calculan con el tipo nuevo
	if err := conn.QueryRow(`SELECT stock FROM v_stock_producto WHERE producto_id='p1'`).Scan(&stock); err != nil || stock != 4 {
		t.Errorf("v_stock_producto = %d, err = %v; se esperaba 4", stock, err)
	}
	if err := conn.QueryRow(`SELECT stock FROM v_stock_lote WHERE lote_id='l1'`).Scan(&stock); err != nil || stock != 4 {
		t.Errorf("v_stock_lote = %d, err = %v; se esperaba 4", stock, err)
	}
	for _, ix := range []string{"idx_ms_producto", "idx_ms_lote", "idx_ms_referencia", "idx_ms_fecha", "idx_ms_created"} {
		var n int
		if err := conn.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='index' AND name = ?`, ix).Scan(&n); err != nil || n != 1 {
			t.Errorf("falta el índice %s tras 00005 (err=%v)", ix, err)
		}
	}

	// Down de 00005: los EDICION_COMPRA pasan a AJUSTE con la nota marcada y el stock no cambia
	if err := m.Down(ctx); err != nil {
		t.Fatalf("down 00005: %v", err)
	}
	var tipo, nota string
	if err := conn.QueryRow(`SELECT tipo, nota FROM movimiento_stock WHERE id='m4'`).Scan(&tipo, &nota); err != nil || tipo != "AJUSTE" || nota != "[EDICION_COMPRA] " {
		t.Errorf("m4 tras down = %q %q (%v)", tipo, nota, err)
	}
	if err := conn.QueryRow(`SELECT stock FROM v_stock_producto WHERE producto_id='p1'`).Scan(&stock); err != nil || stock != 4 {
		t.Errorf("stock tras down = %d (%v)", stock, err)
	}
	if _, err := conn.Exec(`INSERT INTO movimiento_stock (id, producto_id, lote_id, cantidad, tipo, fecha) VALUES ('m8','p1','l1',1,'EDICION_COMPRA','2026-01-03')`); err == nil {
		t.Error("tras el down el CHECK no debería aceptar EDICION_COMPRA")
	}
	if err := m.Up(ctx); err != nil {
		t.Fatalf("up tras down: %v", err)
	}

	// Ida y vuelta
	if err := m.Reset(ctx); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := m.Up(ctx); err != nil {
		t.Fatalf("up tras reset: %v", err)
	}
}

func mustExec(t *testing.T, conn *sql.DB, q string) {
	t.Helper()
	if _, err := conn.Exec(q); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}
