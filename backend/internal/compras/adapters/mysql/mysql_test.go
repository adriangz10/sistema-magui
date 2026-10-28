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
	comenlaces "magui/internal/compras/adapters/enlaces"
	commysql "magui/internal/compras/adapters/mysql"
	comapp "magui/internal/compras/application"
	comdomain "magui/internal/compras/domain"
	"magui/internal/inventario"
	invapp "magui/internal/inventario/application"
	invdomain "magui/internal/inventario/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/id"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/platform/db"
)

// Integración de catálogo + inventario + compras sobre MySQL real.
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

func ptr[T any](v T) *T { return &v }

func d(s string) dinero.Dinero { return dinero.MustParse(s) }

func TestCompraEInventarioMySQL(t *testing.T) {
	conn := baseTemporal(t)
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cat := catalogo.NuevoModuloMySQL(conn, log).Servicio
	inv := inventario.NuevoModuloMySQL(conn, cat, log).Servicio
	com := compras.NuevoModuloMySQL(conn, cat, inv, log).Servicio
	pag := paginacion.Nueva(1, 10)
	hoy := fecha.DelInstante(time.Now(), reloj.ZonaArgentina)

	natura := debe(cat.CrearEmpresa(ctx, "Natura"))
	kaiak := debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple, Datos: catdomain.DatosProducto{
		SKU: "P1", Nombre: "Kaiak", PrecioVenta: d("59200"), StockMinimo: 2}}))
	jabon := debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple, Datos: catdomain.DatosProducto{
		SKU: "P2", Nombre: "Jabón", PrecioVenta: d("5500")}}))
	combo := debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoCombo, Datos: catdomain.DatosProducto{
		Nombre: "Combo", PrecioVenta: d("60000"), Items: []catdomain.ComboItem{{ProductoID: kaiak.ID, Cantidad: 1}, {ProductoID: jabon.ID, Cantidad: 1}}}}))

	// --- Compra: Kaiak en dos lotes (uno por vencer) + jabones con precio nuevo ---
	pronto, lejos := hoy.SumarDias(20), hoy.SumarDias(400)
	compra := debe(com.RegistrarCompra(ctx, comapp.ComandoCompra{EmpresaID: natura.ID, Notas: "Ciclo 15", Items: []comapp.ItemCmd{
		{ProductoID: kaiak.ID, Cantidad: 2, CostoUnitario: d("35000"), Vencimiento: &pronto},
		{ProductoID: kaiak.ID, Cantidad: 3, CostoUnitario: d("36000"), Vencimiento: &lejos, PrecioVentaNuevo: ptr(d("62000"))},
		{ProductoID: jabon.ID, Cantidad: 10, CostoUnitario: d("2500.50")},
	}}))
	if compra.Total.String() != "203005.00" || compra.Unidades != 15 || len(compra.Items) != 3 || *compra.Empresa != "Natura" {
		t.Fatalf("compra = %+v", compra)
	}
	if compra.Items[0].Subtotal.String() != "70000.00" || compra.Items[0].Vencimiento.String() != pronto.String() {
		t.Errorf("renglón 1 = %+v", compra.Items[0])
	}

	// Stock por lote
	st := debe(inv.StockDeProducto(ctx, kaiak.ID))
	if st.Stock != 5 || len(st.Lotes) != 2 || st.Lotes[0].Estado != invdomain.PorVencer || *st.Lotes[0].DiasRestantes != 20 {
		t.Fatalf("stock kaiak = %+v", st)
	}

	// El catálogo quedó con el último costo y el precio nuevo (con historial de origen COMPRA)
	pk := debe(cat.ObtenerProducto(ctx, kaiak.ID))
	if pk.Costo.String() != "36000.00" || pk.PrecioVenta.String() != "62000.00" || pk.Stock != 5 {
		t.Errorf("kaiak en catálogo = costo %v precio %s stock %d", pk.Costo, pk.PrecioVenta, pk.Stock)
	}
	if h := debe(cat.HistorialPrecios(ctx, kaiak.ID, pag)); h.Total != 1 || h.Items[0].Origen != catdomain.OrigenCompra {
		t.Errorf("historial = %+v", h.Items)
	}
	if pj := debe(cat.ObtenerProducto(ctx, jabon.ID)); pj.PrecioVenta.String() != "5500.00" || pj.Costo.String() != "2500.50" {
		t.Errorf("jabón sin precio nuevo = %s / %v", pj.PrecioVenta, pj.Costo)
	}

	// --- Salida por FEFO: primero el lote que vence antes ---
	asig := debe(inv.RegistrarSalida(ctx, invapp.Salida{ProductoID: kaiak.ID, Cantidad: 3, Tipo: invdomain.MovVenta}))
	if len(asig) != 2 || asig[0].Cantidad != 2 || asig[0].Costo.String() != "35000.00" || asig[1].Cantidad != 1 {
		t.Fatalf("FEFO = %+v", asig)
	}
	if _, err := inv.RegistrarSalida(ctx, invapp.Salida{ProductoID: kaiak.ID, Cantidad: 3, Tipo: invdomain.MovVenta}); err == nil {
		t.Error("debería faltar stock (quedan 2)")
	}

	// --- Ajuste manual y movimientos ---
	st = debe(inv.AjustarStock(ctx, invapp.ComandoAjuste{ProductoID: jabon.ID, Cantidad: -1, Motivo: "Se rompió"}))
	if st.Stock != 9 {
		t.Errorf("tras ajuste = %d", st.Stock)
	}
	movs := debe(inv.ListarMovimientos(ctx, invapp.FiltroMovimientos{ProductoID: kaiak.ID}, pag))
	if movs.Total != 4 || movs.Items[0].Tipo != invdomain.MovVenta || movs.Items[0].Producto != "Kaiak" {
		t.Errorf("movimientos kaiak = %d, primero %+v", movs.Total, movs.Items[0])
	}
	if a := debe(inv.ListarMovimientos(ctx, invapp.FiltroMovimientos{Tipo: invdomain.MovAjuste}, pag)); a.Total != 1 || a.Items[0].Nota != "Se rompió" {
		t.Errorf("ajustes = %+v", a.Items)
	}

	// --- Vencimientos y alertas ---
	// Lote ya vencido cargado por ajuste
	debe(inv.AjustarStock(ctx, invapp.ComandoAjuste{ProductoID: jabon.ID, Cantidad: 1, Motivo: "Apareció en depósito", Vencimiento: ptr(hoy.SumarDias(-5))}))
	if v := debe(inv.ListarVencimientos(ctx, invapp.FiltroVencidos, pag)); v.Total != 1 || v.Items[0].DiasRestantes != -5 || v.Items[0].Producto != "Jabón" {
		t.Errorf("vencidos = %+v", v.Items)
	}
	// El lote "pronto" de Kaiak quedó en 0 por la venta: ya no figura por vencer
	if v := debe(inv.ListarVencimientos(ctx, invapp.FiltroPorVencer, pag)); v.Total != 0 {
		t.Errorf("por vencer = %+v", v.Items)
	}
	alertas := debe(inv.Alertas(ctx))
	if alertas.Vencidos != 1 || alertas.PorVencer != 0 || alertas.StockBajo != 1 || alertas.DiasAlerta != 90 {
		t.Errorf("alertas = %+v", alertas) // Kaiak: stock 2 <= mínimo 2
	}
	if err := inv.CambiarDiasAlerta(ctx, 30); err != nil || debe(inv.DiasAlerta(ctx)) != 30 {
		t.Errorf("configuración: %v", err)
	}

	// --- Listado de compras ---
	desde := hoy.InicioEn(reloj.ZonaArgentina)
	if l := debe(com.ListarCompras(ctx, comapp.Filtro{Desde: &desde, EmpresaID: natura.ID}, pag)); l.Total != 1 || l.Items[0].Productos != 3 {
		t.Errorf("compras = %+v", l)
	}
	manana := hoy.SumarDias(1).InicioEn(reloj.ZonaArgentina)
	if l := debe(com.ListarCompras(ctx, comapp.Filtro{Desde: &manana}, pag)); l.Total != 0 {
		t.Errorf("compras desde mañana = %d", l.Total)
	}

	// --- Una compra con un combo falla entera y no deja nada ---
	var antes int
	_ = conn.QueryRow(`SELECT COUNT(*) FROM movimiento_stock`).Scan(&antes)
	_, err := com.RegistrarCompra(ctx, comapp.ComandoCompra{Items: []comapp.ItemCmd{
		{ProductoID: jabon.ID, Cantidad: 1, CostoUnitario: d("1")},
		{ProductoID: combo.ID, Cantidad: 1, CostoUnitario: d("1")},
	}})
	if err == nil {
		t.Fatal("comprar un combo debería fallar")
	}
	var despues, comprasN int
	_ = conn.QueryRow(`SELECT COUNT(*) FROM movimiento_stock`).Scan(&despues)
	_ = conn.QueryRow(`SELECT COUNT(*) FROM compra`).Scan(&comprasN)
	if despues != antes || comprasN != 1 {
		t.Errorf("quedaron datos a medias: movimientos %d->%d, compras %d", antes, despues, comprasN)
	}
}

// US-001a: editar la cabecera de una compra sobre MySQL real.
func TestEditarCabeceraCompraMySQL(t *testing.T) {
	conn := baseTemporal(t)
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cat := catalogo.NuevoModuloMySQL(conn, log).Servicio
	inv := inventario.NuevoModuloMySQL(conn, cat, log).Servicio
	com := compras.NuevoModuloMySQL(conn, cat, inv, log).Servicio
	pag := paginacion.Nueva(1, 10)

	natura := debe(cat.CrearEmpresa(ctx, "Natura"))
	avon := debe(cat.CrearEmpresa(ctx, "Avon"))
	kaiak := debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple, Datos: catdomain.DatosProducto{
		SKU: "P1", Nombre: "Kaiak", PrecioVenta: d("59200")}}))
	hoy := fecha.DelInstante(time.Now(), reloj.ZonaArgentina)
	original := hoy.SumarDias(-7)
	nueva := hoy.SumarDias(-8)

	c := debe(com.RegistrarCompra(ctx, comapp.ComandoCompra{EmpresaID: natura.ID,
		Fecha: ptr(original.InicioEn(reloj.ZonaArgentina).Add(12 * time.Hour)),
		Items: []comapp.ItemCmd{{ProductoID: kaiak.ID, Cantidad: 2, CostoUnitario: d("35000")}}}))
	if c.Version != 1 || c.EditadaEn != nil || c.Ediciones == nil || len(c.Ediciones) != 0 {
		t.Fatalf("compra recién registrada = %+v", c)
	}

	// Edición feliz: versión 2, historial con los tres cambios, listado con editadaEn.
	diaNuevo := nueva
	e := debe(com.EditarCompra(ctx, comapp.ComandoEditarCompra{ID: c.ID, Version: 1, Fecha: comapp.FechaEditada{Dia: &diaNuevo},
		EmpresaID: avon.ID, Notas: "Pedido ciclo 15", Origen: "WEB", Motivo: "Error de carga"}))
	if e.Version != 2 || e.EditadaEn == nil || *e.Empresa != "Avon" || e.Total.String() != c.Total.String() || len(e.Ediciones) != 1 {
		t.Fatalf("editada = %+v", e)
	}
	ed := e.Ediciones[0]
	if ed.Version != 2 || ed.Origen != "WEB" || *ed.Motivo != "Error de carga" || len(ed.Cabecera) != 3 || ed.Cabecera[1].Antes != "Natura" ||
		ed.Cabecera[1].Despues != "Avon" || len(ed.Renglones) != 0 {
		t.Errorf("edición = %+v", ed)
	}
	l := debe(com.ListarCompras(ctx, comapp.Filtro{}, pag))
	if l.Total != 1 || l.Items[0].Version != 2 || l.Items[0].EditadaEn == nil {
		t.Errorf("listado = %+v", l.Items)
	}

	// Sin cambios: misma versión, sin nueva entrada.
	s := debe(com.EditarCompra(ctx, comapp.ComandoEditarCompra{ID: c.ID, Version: 2, Fecha: comapp.FechaEditada{Dia: &diaNuevo},
		EmpresaID: avon.ID, Notas: "Pedido ciclo 15", Motivo: "nada"}))
	if s.Version != 2 || len(s.Ediciones) != 1 {
		t.Errorf("sin cambios = version %d, ediciones %d", s.Version, len(s.Ediciones))
	}

	// Versión vieja: 409 y no se aplica.
	if _, err := com.EditarCompra(ctx, comapp.ComandoEditarCompra{ID: c.ID, Version: 1, Fecha: comapp.FechaEditada{Dia: &diaNuevo}, Notas: "otra"}); err == nil ||
		!errors.Is(err, comdomain.ErrVersionDesactualizada) {
		t.Errorf("versión vieja: %v", err)
	}
	// ActualizarCompra directo con versión vieja.
	repo := commysql.Repositorio{DB: conn}
	if err := repo.ActualizarCompra(ctx, comdomain.Compra{ID: c.ID, Version: 9}, 1); !errors.Is(err, comdomain.ErrVersionDesactualizada) {
		t.Errorf("ActualizarCompra: %v", err)
	}

	// Quitar la empresa.
	q := debe(com.EditarCompra(ctx, comapp.ComandoEditarCompra{ID: c.ID, Version: 2, Fecha: comapp.FechaEditada{Dia: &diaNuevo}, Notas: "Pedido ciclo 15"}))
	if q.Version != 3 || q.EmpresaID != nil || q.Ediciones[0].Cabecera[0].Despues != nil {
		t.Errorf("sin empresa = %+v", q.Ediciones[0])
	}

	// Empresa inexistente: 404 y nada cambia.
	if _, err := com.EditarCompra(ctx, comapp.ComandoEditarCompra{ID: c.ID, Version: 3, Fecha: comapp.FechaEditada{Dia: &diaNuevo},
		EmpresaID: "no-existe", Notas: "Pedido ciclo 15"}); err == nil {
		t.Error("empresa inexistente debería fallar")
	}

	// Todo o nada: si falla la auditoría, la compra queda como estaba (transacción real).
	falla := comapp.NuevoServicio(comapp.Dependencias{
		Repo: repoFallaEdicion{repo}, Catalogo: comenlaces.Catalogo{Svc: cat}, Inventario: comenlaces.Inventario{Svc: inv},
		Tx: db.NuevoTransactorMySQL(conn), IDs: id.UUIDv7{}, Reloj: reloj.Sistema{}})
	if _, err := falla.EditarCompra(ctx, comapp.ComandoEditarCompra{ID: c.ID, Version: 3, Fecha: comapp.FechaEditada{Dia: &diaNuevo}, Notas: "se pierde"}); err == nil {
		t.Fatal("debería fallar la auditoría")
	}
	after := debe(com.ObtenerCompra(ctx, c.ID))
	if after.Version != 3 || after.Notas == nil || *after.Notas != "Pedido ciclo 15" || len(after.Ediciones) != 2 {
		t.Errorf("no se revirtió: %+v", after)
	}
}

type repoFallaEdicion struct{ commysql.Repositorio }

func (repoFallaEdicion) GuardarEdicion(context.Context, comdomain.Edicion) error {
	return errors.New("falla simulada")
}
