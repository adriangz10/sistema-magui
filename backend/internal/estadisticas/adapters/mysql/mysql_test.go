package mysql_test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"magui/internal/caja"
	cajaapp "magui/internal/caja/application"
	cajadomain "magui/internal/caja/domain"
	"magui/internal/catalogo"
	catapp "magui/internal/catalogo/application"
	catdomain "magui/internal/catalogo/domain"
	"magui/internal/compras"
	comapp "magui/internal/compras/application"
	"magui/internal/estadisticas"
	estapp "magui/internal/estadisticas/application"
	estdomain "magui/internal/estadisticas/domain"
	"magui/internal/inventario"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/platform/db"
	"magui/internal/ventas"
	venapp "magui/internal/ventas/application"
	vendomain "magui/internal/ventas/domain"
)

// Integración de todo el backend sobre MySQL real: compras, ventas, gastos y estadísticas.
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

func TestEstadisticasMySQL(t *testing.T) {
	conn := baseTemporal(t)
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cat := catalogo.NuevoModuloMySQL(conn, log).Servicio
	inv := inventario.NuevoModuloMySQL(conn, cat, log).Servicio
	com := compras.NuevoModuloMySQL(conn, cat, inv, log).Servicio
	ven := ventas.NuevoModuloMySQL(conn, cat, inv, nil, log).Servicio
	cj := caja.NuevoModuloMySQL(conn, log).Servicio
	est := estadisticas.NuevoModuloMySQL(conn, log).Servicio

	nuevo := func(sku, nombre, precio string) string {
		return debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple,
			Datos: catdomain.DatosProducto{SKU: sku, Nombre: nombre, PrecioVenta: d(precio)}})).ID
	}
	kaiak, jabon := nuevo("P1", "Kaiak", "10000"), nuevo("P2", "Jabón", "2000")
	sinCosto := nuevo("P3", "Muestra", "500")
	debe(com.RegistrarCompra(ctx, comapp.ComandoCompra{Items: []comapp.ItemCmd{
		{ProductoID: kaiak, Cantidad: 10, CostoUnitario: d("6000")},
		{ProductoID: jabon, Cantidad: 10, CostoUnitario: d("1000")},
	}}))
	debe(inv.AjustarStock(ctx, invAjuste(sinCosto)))

	vender := func(pagos []vendomain.PagoIngresado, items ...venapp.LineaCmd) venapp.VentaDetalle {
		return debe(ven.RegistrarVenta(ctx, venapp.ComandoVenta{Items: items, Pagos: pagos}))
	}
	ef := func(m string) vendomain.PagoIngresado {
		return vendomain.PagoIngresado{Metodo: vendomain.Efectivo, Monto: d(m)}
	}
	qr := func(m string) vendomain.PagoIngresado {
		return vendomain.PagoIngresado{Metodo: vendomain.QR, Monto: d(m)}
	}

	// Venta 1: 2 Kaiak = 20000 (costo 12000), efectivo
	vender([]vendomain.PagoIngresado{ef("20000")}, venapp.LineaCmd{ProductoID: kaiak, Cantidad: 2})
	// Venta 2: 1 Kaiak + 3 jabones = 16000 (costo 9000), QR 10000 + efectivo 6000
	vender([]vendomain.PagoIngresado{qr("10000"), ef("10000")}, venapp.LineaCmd{ProductoID: kaiak, Cantidad: 1}, venapp.LineaCmd{ProductoID: jabon, Cantidad: 3})
	// Venta 3: 1 muestra sin costo = 500, QR
	vender([]vendomain.PagoIngresado{qr("500")}, venapp.LineaCmd{ProductoID: sinCosto, Cantidad: 1})
	// Venta 4: anulada (no cuenta)
	v4 := vender([]vendomain.PagoIngresado{ef("10000")}, venapp.LineaCmd{ProductoID: kaiak, Cantidad: 1})
	debe(ven.AnularVenta(ctx, v4.ID, "Error"))

	// Gastos: 3000 efectivo + 1500 transferencia + uno anulado
	debe(cj.RegistrarGasto(ctx, cajaapp.ComandoGasto{Monto: d("3000"), Motivo: "Luz", Categoria: "Servicios"}))
	debe(cj.RegistrarGasto(ctx, cajaapp.ComandoGasto{Monto: d("1500"), Motivo: "Remis", Metodo: cajadomain.Transferencia}))
	g := debe(cj.RegistrarGasto(ctx, cajaapp.ComandoGasto{Monto: d("99999"), Motivo: "Error de tipeo"}))
	debe(cj.AnularGasto(ctx, g.ID, "Mal cargado"))

	// --- Resumen del día ---
	r := debe(est.Resumen(ctx, estapp.ConsultaPeriodo{Tipo: estdomain.Dia}))
	v := r.Ventas
	if v.Cantidad != 3 || v.Total.String() != "36500.00" || v.Costo.String() != "21000.00" || v.Ganancia.String() != "15500.00" {
		t.Errorf("ventas = %+v", v)
	}
	if v.Anuladas != 1 || v.RenglonesSinCosto != 1 || v.TicketPromedio.String() != "12166.67" || v.Margen != 42.47 {
		t.Errorf("ventas (detalle) = %+v", v)
	}
	if r.Compras.Total.String() != "70000.00" || r.Gastos.Total.String() != "4500.00" || r.Gastos.Cantidad != 2 {
		t.Errorf("compras %+v gastos %+v", r.Compras, r.Gastos)
	}
	if r.GananciaNeta.String() != "11000.00" {
		t.Errorf("ganancia neta = %s", r.GananciaNeta)
	}
	// Efectivo: 20000 + 6000 aplicados − 3000 de gastos en efectivo
	if r.EfectivoNeto.String() != "23000.00" {
		t.Errorf("efectivo neto = %s", r.EfectivoNeto)
	}
	metodo := map[string]estapp.MetodoDTO{}
	for _, m := range r.PorMetodo {
		metodo[m.Metodo] = m
	}
	if metodo["EFECTIVO"].Monto.String() != "26000.00" || metodo["QR"].Monto.String() != "10500.00" || metodo["QR"].Cantidad != 2 || !metodo["TARJETA"].Monto.EsCero() {
		t.Errorf("por método = %+v", r.PorMetodo)
	}

	// --- Serie diaria y por hora ---
	dias := debe(est.VentasPorDia(ctx, estapp.ConsultaPeriodo{Tipo: estdomain.Semana}))
	hoy := time.Now().In(reloj.ZonaArgentina).Format(time.DateOnly)
	var deHoy estapp.VentaDiaDTO
	for _, dd := range dias {
		if dd.Fecha.String() == hoy {
			deHoy = dd
		}
	}
	if len(dias) != 7 || deHoy.Cantidad != 3 || deHoy.Total.String() != "36500.00" || deHoy.Ganancia.String() != "15500.00" {
		t.Errorf("día de hoy = %+v (días %d)", deHoy, len(dias))
	}
	horas := debe(est.VentasPorHora(ctx, estapp.ConsultaPeriodo{Tipo: estdomain.Dia}))
	if h := horas[time.Now().In(reloj.ZonaArgentina).Hour()]; h.Cantidad != 3 {
		t.Errorf("hora actual = %+v", h)
	}

	// --- Ranking ---
	top := debe(est.MasVendidos(ctx, estapp.ConsultaPeriodo{Tipo: estdomain.Mes}, 10, estapp.PorUnidades))
	// Kaiak y Jabón empatan en 3 unidades: desempata el importe
	if len(top) != 3 || top[0].Nombre != "Kaiak" || top[0].Unidades != 3 || top[0].Ganancia.String() != "12000.00" || top[1].Nombre != "Jabón" {
		t.Errorf("por unidades = %+v", top)
	}
	if top[2].Ganancia != nil {
		t.Error("producto sin costo: ganancia desconocida")
	}
	if top := debe(est.MasVendidos(ctx, estapp.ConsultaPeriodo{Tipo: estdomain.Mes}, 1, estapp.PorImporte)); len(top) != 1 || top[0].Nombre != "Kaiak" {
		t.Errorf("por importe = %+v", top)
	}

	// --- Listado de gastos ---
	l := debe(cj.ListarGastos(ctx, cajaapp.Filtro{IncluirAnulados: true}, paginacion.Nueva(1, 10)))
	if l.Total != 3 || l.TotalMonto.String() != "4500.00" {
		t.Errorf("gastos con anulados: %d, suma %s (la suma no incluye anulados)", l.Total, l.TotalMonto)
	}
	if l := debe(cj.ListarGastos(ctx, cajaapp.Filtro{Categoria: "Servicios"}, paginacion.Nueva(1, 10))); l.Total != 1 {
		t.Errorf("por categoría = %d", l.Total)
	}
	cats := debe(cj.Categorias(ctx))
	if len(cats) != len(cajadomain.CategoriasSugeridas) {
		t.Errorf("categorías = %v", cats)
	}

	// --- Período sin movimientos: todo en cero, sin variación ---
	ayer := time.Now().In(reloj.ZonaArgentina).AddDate(0, 0, -400)
	desde := fechaDe(ayer)
	r = debe(est.Resumen(ctx, estapp.ConsultaPeriodo{Tipo: estdomain.Rango, Desde: &desde, Hasta: &desde}))
	if r.Ventas.Cantidad != 0 || !r.Ventas.Total.EsCero() || r.Ventas.Margen != 0 || r.Anterior.VariacionTotal != nil {
		t.Errorf("período vacío = %+v", r)
	}
}
