package mysql_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"magui/internal/catalogo"
	catapp "magui/internal/catalogo/application"
	catdomain "magui/internal/catalogo/domain"
	"magui/internal/compras"
	comapp "magui/internal/compras/application"
	"magui/internal/inventario"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/platform/db"
	"magui/internal/ventas"
	venapp "magui/internal/ventas/application"
	vendomain "magui/internal/ventas/domain"
)

// Integración completa (catálogo + inventario + compras + ventas) sobre MySQL real.
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

func debe[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func d(s string) dinero.Dinero { return dinero.MustParse(s) }

func pago(m vendomain.MetodoPago, monto string) vendomain.PagoIngresado {
	return vendomain.PagoIngresado{Metodo: m, Monto: d(monto)}
}

func contar(t *testing.T, conn *sql.DB, q string) int {
	var n int
	if err := conn.QueryRow(q).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestVentasMySQL(t *testing.T) {
	conn := baseTemporal(t)
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	catM := catalogo.NuevoModuloMySQL(conn, log)
	invM := inventario.NuevoModuloMySQL(conn, catM.Servicio, log)
	cat, inv := catM.Servicio, invM.Servicio
	com := compras.NuevoModuloMySQL(conn, cat, inv, log).Servicio
	ven := ventas.NuevoModuloMySQL(conn, cat, inv, log).Servicio
	pag := paginacion.Nueva(1, 10)
	hoy := fecha.DelInstante(time.Now(), reloj.ZonaArgentina)

	nuevo := func(sku, nombre, precio string) catapp.ProductoDetalle {
		return debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple,
			Datos: catdomain.DatosProducto{SKU: sku, Nombre: nombre, PrecioVenta: d(precio)}}))
	}
	kaiak, jabon := nuevo("P1", "Kaiak", "59200"), nuevo("P2", "Jabón", "5500")
	combo := debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoCombo, Datos: catdomain.DatosProducto{
		Nombre: "Combo Madre", PrecioVenta: d("62000"),
		Items: []catdomain.ComboItem{{ProductoID: kaiak.ID, Cantidad: 1}, {ProductoID: jabon.ID, Cantidad: 2}}}}))

	// Stock: Kaiak en 2 lotes con distinto costo; jabones
	pronto, lejos := hoy.SumarDias(30), hoy.SumarDias(300)
	debe(com.RegistrarCompra(ctx, comapp.ComandoCompra{Items: []comapp.ItemCmd{
		{ProductoID: kaiak.ID, Cantidad: 2, CostoUnitario: d("30000"), Vencimiento: &lejos},
		{ProductoID: kaiak.ID, Cantidad: 2, CostoUnitario: d("28000"), Vencimiento: &pronto},
		{ProductoID: jabon.ID, Cantidad: 10, CostoUnitario: d("2500")},
	}}))

	// --- Venta con pago combinado: 3 Kaiak (2 del lote "pronto" a 28000 + 1 a 30000) ---
	v1 := debe(ven.RegistrarVenta(ctx, venapp.ComandoVenta{
		Items: []venapp.LineaCmd{{ProductoID: kaiak.ID, Cantidad: 3}},
		Pagos: []vendomain.PagoIngresado{pago(vendomain.Transferencia, "100000"), pago(vendomain.Efectivo, "80000")},
	}))
	if v1.Total.String() != "177600.00" || v1.CostoTotal.String() != "86000.00" || v1.Ganancia.String() != "91600.00" || !v1.CostoCompleto {
		t.Errorf("venta 1: total %s costo %s ganancia %s", v1.Total, v1.CostoTotal, v1.Ganancia)
	}
	if len(v1.Pagos) != 2 || v1.Vuelto.String() != "2400.00" || v1.Pagos[1].MontoRecibido.String() != "80000.00" {
		t.Errorf("pagos = %+v vuelto %s", v1.Pagos, v1.Vuelto)
	}
	if len(v1.Metodos) != 2 || v1.Items[0].Producto != "Kaiak" || v1.Unidades != 3 {
		t.Errorf("resumen = %+v", v1.VentaResumen)
	}
	st := debe(inv.StockDeProducto(ctx, kaiak.ID))
	if st.Stock != 1 || len(st.Lotes) != 1 || st.Lotes[0].Vencimiento.String() != lejos.String() {
		t.Errorf("FEFO: quedó %+v", st)
	}

	// --- Combo con tarjeta ---
	v2 := debe(ven.RegistrarVenta(ctx, venapp.ComandoVenta{Origen: vendomain.OrigenMovil,
		Items: []venapp.LineaCmd{{ProductoID: combo.ID, Cantidad: 1}},
		Pagos: []vendomain.PagoIngresado{pago(vendomain.Tarjeta, "62000")}}))
	if v2.CostoTotal.String() != "35000.00" || v2.Origen != vendomain.OrigenMovil || v2.Items[0].Tipo != "COMBO" {
		t.Errorf("venta combo = %+v", v2)
	}
	if debe(inv.StockDeProducto(ctx, jabon.ID)).Stock != 8 || debe(inv.StockDeProducto(ctx, kaiak.ID)).Stock != 0 {
		t.Error("el combo debería descontar 1 Kaiak y 2 jabones")
	}

	// --- Sin stock: falla entera y no deja rastros ---
	ventasAntes, movsAntes := contar(t, conn, `SELECT COUNT(*) FROM venta`), contar(t, conn, `SELECT COUNT(*) FROM movimiento_stock`)
	_, err := ven.RegistrarVenta(ctx, venapp.ComandoVenta{
		Items: []venapp.LineaCmd{{ProductoID: jabon.ID, Cantidad: 1}, {ProductoID: kaiak.ID, Cantidad: 5}},
		Pagos: []vendomain.PagoIngresado{pago(vendomain.Efectivo, "999999")}})
	if e, _ := errores.Como(err); e == nil || e.Codigo != "stock_insuficiente" || e.Mensaje != "Kaiak: No hay stock suficiente (disponible: 0)" {
		t.Fatalf("sin stock: %v", err)
	}
	if contar(t, conn, `SELECT COUNT(*) FROM venta`) != ventasAntes || contar(t, conn, `SELECT COUNT(*) FROM movimiento_stock`) != movsAntes {
		t.Error("una venta fallida no debe dejar datos")
	}

	// --- Falta plata ---
	if _, err := ven.RegistrarVenta(ctx, venapp.ComandoVenta{Items: []venapp.LineaCmd{{ProductoID: jabon.ID, Cantidad: 1}},
		Pagos: []vendomain.PagoIngresado{pago(vendomain.QR, "5000")}}); err == nil || err.Error() != "pagos_insuficientes: Faltan $500.00 para completar el cobro" {
		t.Errorf("falta plata: %v", err)
	}

	// --- Listados ---
	if l := debe(ven.ListarVentas(ctx, venapp.Filtro{Metodo: vendomain.Tarjeta}, pag)); l.Total != 1 || l.Items[0].ID != v2.ID {
		t.Errorf("filtro por tarjeta = %+v", l)
	}
	if l := debe(ven.ListarVentas(ctx, venapp.Filtro{}, pag)); l.Total != 2 || l.Items[0].ID != v2.ID {
		t.Errorf("todas (la más reciente primero) = %+v", l)
	}

	// --- Anulación: el stock vuelve a sus lotes ---
	an := debe(ven.AnularVenta(ctx, v1.ID, "Error de carga"))
	if an.Estado != vendomain.Anulada || *an.MotivoAnulacion != "Error de carga" || an.AnuladaEn == nil {
		t.Errorf("anulada = %+v", an)
	}
	st = debe(inv.StockDeProducto(ctx, kaiak.ID))
	if st.Stock != 3 || len(st.Lotes) != 2 { // vuelven las 3 unidades de la venta anulada
		t.Errorf("tras anular: %+v", st)
	}
	if _, err := ven.AnularVenta(ctx, v1.ID, "otra"); !errors.Is(err, vendomain.ErrVentaYaAnulada) {
		t.Errorf("doble anulación: %v", err)
	}
	if l := debe(ven.ListarVentas(ctx, venapp.Filtro{Estado: vendomain.Anulada}, pag)); l.Total != 1 {
		t.Errorf("anuladas = %d", l.Total)
	}

	// --- Venta sin stock confirmada por la cajera ---
	v3 := debe(ven.RegistrarVenta(ctx, venapp.ComandoVenta{PermitirSinStock: true,
		Items: []venapp.LineaCmd{{ProductoID: kaiak.ID, Cantidad: 6}},
		Pagos: []vendomain.PagoIngresado{pago(vendomain.Efectivo, "355200")}}))
	if debe(inv.StockDeProducto(ctx, kaiak.ID)).Stock != -3 || v3.Total.String() != "355200.00" {
		t.Error("venta forzada: el stock debería quedar en -3")
	}
}
