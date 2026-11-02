package mysql_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"magui/internal/caja"
	cajaapp "magui/internal/caja/application"
	cajadomain "magui/internal/caja/domain"
	"magui/internal/catalogo"
	catapp "magui/internal/catalogo/application"
	catdomain "magui/internal/catalogo/domain"
	"magui/internal/inventario"
	invapp "magui/internal/inventario/application"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/paginacion"
	"magui/internal/platform/db"
	"magui/internal/ventas"
	venapp "magui/internal/ventas/application"
	vendomain "magui/internal/ventas/domain"
)

// Integración de apertura/cierre de caja (US-002) sobre MySQL real.
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

func debe[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func d(s string) dinero.Dinero { return dinero.MustParse(s) }

func contado(ef, tr, qr, ta string) map[cajadomain.Metodo]dinero.Dinero {
	return map[cajadomain.Metodo]dinero.Dinero{cajadomain.Efectivo: d(ef), cajadomain.Transferencia: d(tr),
		cajadomain.QR: d(qr), cajadomain.Tarjeta: d(ta)}
}

func TestCajaAperturaCierreMySQL(t *testing.T) {
	conn := baseTemporal(t)
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cat := catalogo.NuevoModuloMySQL(conn, log).Servicio
	inv := inventario.NuevoModuloMySQL(conn, cat, log).Servicio
	cj := caja.NuevoModuloMySQL(conn, log)
	ven := ventas.NuevoModuloMySQL(conn, cat, inv, cj.Sesiones, log).Servicio

	prod := debe[catapp.ProductoDetalle](t)(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple,
		Datos: catdomain.DatosProducto{SKU: "P1", Nombre: "Kaiak", PrecioVenta: d("10000")}})).ID
	if _, err := inv.AjustarStock(ctx, invapp.ComandoAjuste{ProductoID: prod, Cantidad: 20, Motivo: "Inicial"}); err != nil {
		t.Fatal(err)
	}
	vender := func(pagos ...vendomain.PagoIngresado) (venapp.VentaDetalle, error) {
		return ven.RegistrarVenta(ctx, venapp.ComandoVenta{Items: []venapp.LineaCmd{{ProductoID: prod, Cantidad: 1}}, Pagos: pagos})
	}
	pago := func(m vendomain.MetodoPago, monto string) vendomain.PagoIngresado {
		return vendomain.PagoIngresado{Metodo: m, Monto: d(monto)}
	}

	// Sin caja abierta no se vende.
	if _, err := vender(pago(vendomain.Efectivo, "10000")); !errors.Is(err, cajadomain.ErrCajaCerrada) {
		t.Fatalf("venta sin caja: %v", err)
	}

	ab := debe[cajaapp.SesionDTO](t)(cj.Sesiones.Abrir(ctx, d("5000"), "Turno mañana"))
	if _, err := cj.Sesiones.Abrir(ctx, 0, ""); !errors.Is(err, cajadomain.ErrCajaYaAbierta) {
		t.Errorf("segunda apertura: %v", err)
	}

	// Ventas: 10000 efectivo (le dieron 20000), 10000 mitad QR mitad tarjeta, una anulada.
	if _, err := vender(pago(vendomain.Efectivo, "20000")); err != nil {
		t.Fatal(err)
	}
	debe[venapp.VentaDetalle](t)(vender(pago(vendomain.QR, "5000"), pago(vendomain.Tarjeta, "5000")))
	anulada := debe[venapp.VentaDetalle](t)(vender(pago(vendomain.Transferencia, "10000")))
	debe[venapp.VentaDetalle](t)(ven.AnularVenta(ctx, anulada.ID, "Error"))
	// Gastos: 1500 en efectivo; uno anulado no cuenta.
	debe[cajaapp.GastoDTO](t)(cj.Servicio.RegistrarGasto(ctx, cajaapp.ComandoGasto{Monto: d("1500"), Motivo: "Flete"}))
	g := debe[cajaapp.GastoDTO](t)(cj.Servicio.RegistrarGasto(ctx, cajaapp.ComandoGasto{Monto: d("999"), Motivo: "Error"}))
	debe[cajaapp.GastoDTO](t)(cj.Servicio.AnularGasto(ctx, g.ID, "Mal cargado"))

	act, err := cj.Sesiones.Actual(ctx)
	if err != nil || act == nil {
		t.Fatalf("actual: %v %v", act, err)
	}
	if act.CantidadVentas != 2 || act.Arqueo[0].Esperado.String() != "13500.00" || act.Arqueo[2].Ventas.String() != "5000.00" {
		t.Errorf("arqueo en vivo = %+v", act)
	}

	ce := debe[cajaapp.SesionDTO](t)(cj.Sesiones.Cerrar(ctx, ab.ID, cajaapp.ComandoCierre{Version: act.Version,
		Contado: contado("13000", "0", "5000", "5100"), Notas: "Faltó cambio"}))
	if ce.Abierta || ce.CerradaEn == nil || ce.Diferencia.String() != "-400.00" || *ce.Notas != "Faltó cambio" {
		t.Errorf("cierre = %+v", ce)
	}
	if dif := ce.Arqueo[0].Diferencia.String(); dif != "-500.00" {
		t.Errorf("efectivo: diferencia %s", dif)
	}
	if dif := ce.Arqueo[3].Diferencia.String(); dif != "100.00" {
		t.Errorf("tarjeta: diferencia %s", dif)
	}
	// Cerrada: no se vende más.
	if _, err := vender(pago(vendomain.Efectivo, "10000")); !errors.Is(err, cajadomain.ErrCajaCerrada) {
		t.Errorf("venta tras cerrar: %v", err)
	}

	// Editar: corrige el efectivo contado; queda en el historial.
	det := debe[cajaapp.SesionDetalle](t)(cj.Sesiones.Editar(ctx, ab.ID, cajaapp.ComandoEdicion{Version: ce.Version,
		MontoInicial: d("5000"), Contado: contado("13500", "0", "5000", "5100"), Notas: "Faltó cambio", Motivo: "Recontado"}))
	if det.Diferencia.String() != "100.00" || !det.Editada || len(det.Ediciones) != 1 || det.Ediciones[0].Cambios[0].Campo != "contado.EFECTIVO" {
		t.Errorf("edición = %+v", det)
	}
	if _, err := cj.Sesiones.Editar(ctx, ab.ID, cajaapp.ComandoEdicion{Version: ce.Version, MontoInicial: d("1"),
		Contado: contado("0", "0", "0", "0"), Motivo: "x"}); !errors.Is(err, cajadomain.ErrVersionDesactualizada) {
		t.Errorf("edición con versión vieja: %v", err)
	}

	// Segunda caja: queda primera en el historial; la cerrada conserva su arqueo.
	debe[cajaapp.SesionDTO](t)(cj.Sesiones.Abrir(ctx, 0, ""))
	lista := debe[paginacion.Pagina[cajaapp.SesionDTO]](t)(cj.Sesiones.Listar(ctx, paginacion.Solicitud{Pagina: 1, Tamano: 10}))
	if lista.Total != 2 || !lista.Items[0].Abierta || lista.Items[1].ID != ab.ID || lista.Items[1].Arqueo[0].Contado.String() != "13500.00" {
		t.Errorf("historial = %+v", lista)
	}

	// Las ventas de la caja cerrada se consultan por su período.
	vs := debe[paginacion.Pagina[venapp.VentaResumen]](t)(ven.ListarVentas(ctx, venapp.Filtro{Desde: &ce.AbiertaEn, Hasta: ce.CerradaEn},
		paginacion.Solicitud{Pagina: 1, Tamano: 10}))
	if vs.Total != 3 { // 2 confirmadas + 1 anulada
		t.Errorf("ventas de la caja = %d", vs.Total)
	}
}

// Dos aperturas simultáneas: el índice único deja una sola caja abierta.
func TestCajaAperturaSimultaneaMySQL(t *testing.T) {
	conn := baseTemporal(t)
	ctx := context.Background()
	cj := caja.NuevoModuloMySQL(conn, slog.New(slog.NewTextHandler(io.Discard, nil)))
	var wg sync.WaitGroup
	errs := make([]error, 5)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = cj.Sesiones.Abrir(ctx, 0, "")
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case !errors.Is(err, cajadomain.ErrCajaYaAbierta):
			t.Errorf("error inesperado: %v", err)
		}
	}
	var abiertas int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM caja_sesion WHERE cerrada_en IS NULL`).Scan(&abiertas); err != nil {
		t.Fatal(err)
	}
	if ok != 1 || abiertas != 1 {
		t.Errorf("aperturas exitosas = %d, abiertas en la base = %d", ok, abiertas)
	}
}
