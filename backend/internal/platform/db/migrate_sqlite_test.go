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
		"compra_edicion", "movimiento_stock", "gasto_caja", "configuracion", "v_stock_lote", "v_stock_producto",
		"caja_sesion", "caja_arqueo", "caja_edicion", "negocio", "producto_imagen"}
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

	// US-001c: baja lógica de renglones (quitado_en, NULL = vigente) y Down de 00006
	mustExec(t, conn, `INSERT INTO compra_item (id, compra_id, producto_id, cantidad, costo_unitario) VALUES ('ci1','c1','p1',2,10)`)
	var quitado sql.NullString
	if err := conn.QueryRow(`SELECT quitado_en FROM compra_item WHERE id='ci1'`).Scan(&quitado); err != nil || quitado.Valid {
		t.Errorf("quitado_en por defecto debería ser NULL: %v %v", quitado, err)
	}
	mustExec(t, conn, `UPDATE compra_item SET quitado_en='2026-01-04T00:00:00.000Z' WHERE id='ci1'`)
	if err := conn.QueryRow(`SELECT quitado_en FROM compra_item WHERE id='ci1'`).Scan(&quitado); err != nil || !quitado.Valid {
		t.Errorf("quitado_en no se guardó: %v %v", quitado, err)
	}

	// US-002: una sola caja abierta, arqueo por medio y Down de 00007
	mustExec(t, conn, `INSERT INTO caja_sesion (id, abierta_en, monto_inicial) VALUES ('s1','2026-01-01T09:00:00.000Z',1000)`)
	if _, err := conn.Exec(`INSERT INTO caja_sesion (id, abierta_en, monto_inicial) VALUES ('s2','2026-01-01T10:00:00.000Z',0)`); err == nil {
		t.Error("se esperaba error: ya hay una caja abierta")
	}
	mustExec(t, conn, `UPDATE caja_sesion SET cerrada_en='2026-01-01T20:00:00.000Z' WHERE id='s1'`)
	mustExec(t, conn, `INSERT INTO caja_sesion (id, abierta_en, monto_inicial) VALUES ('s2','2026-01-02T09:00:00.000Z',0)`)
	mustExec(t, conn, `INSERT INTO caja_arqueo (sesion_id, metodo, ventas, gastos, esperado, contado) VALUES ('s1','EFECTIVO',500,100,1400,1300)`)
	if _, err := conn.Exec(`INSERT INTO caja_arqueo (sesion_id, metodo, ventas, gastos, esperado, contado) VALUES ('s1','CHEQUE',0,0,0,0)`); err == nil {
		t.Error("se esperaba error por medio inválido en el arqueo")
	}
	if _, err := conn.Exec(`INSERT INTO caja_edicion (id, sesion_id, fecha, motivo, version, cambios) VALUES ('ce1','s1','2026-01-02','x',3,'no')`); err == nil {
		t.Error("se esperaba error: cambios de caja no es JSON")
	}
	// US-003: datos del negocio, una sola fila con valores por defecto
	var nombre string
	var ancho int
	if err := conn.QueryRow(`SELECT nombre, ticket_ancho FROM negocio WHERE id = 1`).Scan(&nombre, &ancho); err != nil || nombre != "Magui" || ancho != 80 {
		t.Errorf("negocio por defecto = %q %d (%v)", nombre, ancho, err)
	}
	if _, err := conn.Exec(`INSERT INTO negocio (id, nombre) VALUES (2, 'Otro')`); err == nil {
		t.Error("se esperaba error: el negocio es una sola fila")
	}
	if _, err := conn.Exec(`UPDATE negocio SET ticket_ancho = 70`); err == nil {
		t.Error("se esperaba error: ancho de papel inválido")
	}
	bajarA(t, m, 7)
	if _, err := conn.Exec(`SELECT 1 FROM negocio`); err == nil {
		t.Error("tras el down de 00008 no debería existir negocio")
	}

	bajarA(t, m, 6)
	if _, err := conn.Exec(`SELECT 1 FROM caja_sesion`); err == nil {
		t.Error("tras el down de 00007 no debería existir caja_sesion")
	}
	if err := m.Up(ctx); err != nil {
		t.Fatalf("up 00007 y 00008: %v", err)
	}

	bajarA(t, m, 5)
	if _, err := conn.Exec(`SELECT quitado_en FROM compra_item`); err == nil {
		t.Error("tras el down de 00006 no debería existir quitado_en")
	}
	if err := m.Up(ctx); err != nil {
		t.Fatalf("up desde 00006: %v", err)
	}

	// Down de 00005: los EDICION_COMPRA pasan a AJUSTE con la nota marcada y el stock no cambia
	bajarA(t, m, 4)
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

// bajarA deshace migraciones hasta quedar en la versión v (así el test no depende de cuántas haya arriba).
func bajarA(t *testing.T, m *Migrator, v int64) {
	t.Helper()
	ctx := context.Background()
	for {
		actual, err := m.Version(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if actual <= v {
			return
		}
		if err := m.Down(ctx); err != nil {
			t.Fatalf("down desde %d: %v", actual, err)
		}
	}
}

func mustExec(t *testing.T, conn *sql.DB, q string) {
	t.Helper()
	if _, err := conn.Exec(q); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}
