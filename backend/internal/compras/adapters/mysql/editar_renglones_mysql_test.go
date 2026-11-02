package mysql_test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"magui/internal/catalogo"
	catapp "magui/internal/catalogo/application"
	catdomain "magui/internal/catalogo/domain"
	"magui/internal/compras"
	comapp "magui/internal/compras/application"
	"magui/internal/inventario"
	invapp "magui/internal/inventario/application"
	invdomain "magui/internal/inventario/domain"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/ventas"
	venapp "magui/internal/ventas/application"
	vendomain "magui/internal/ventas/domain"
)

func pago(m vendomain.MetodoPago, monto string) vendomain.PagoIngresado {
	return vendomain.PagoIngresado{Metodo: m, Monto: d(monto)}
}

// US-001b sobre MySQL real (necesita MAGUI_TEST_MYSQL_DSN; sin él se omite).

func stockDelRenglon(t *testing.T, conn *sql.DB, itemID string) int {
	t.Helper()
	var n int
	err := conn.QueryRow(`SELECT COALESCE(SUM(m.cantidad), 0) FROM movimiento_stock m
		JOIN lote l ON l.id = m.lote_id WHERE l.compra_item_id = ?`, itemID).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

type entornoMySQL struct {
	conn  *sql.DB
	cat   *catapp.Servicio
	inv   *invapp.Servicio
	com   *comapp.Servicio
	kaiak catapp.ProductoDetalle
	jabon catapp.ProductoDetalle
}

func nuevoEntornoMySQL(t *testing.T) *entornoMySQL {
	t.Helper()
	conn := baseTemporal(t)
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cat := catalogo.NuevoModuloMySQL(conn, log).Servicio
	inv := inventario.NuevoModuloMySQL(conn, cat, log).Servicio
	com := compras.NuevoModuloMySQL(conn, cat, inv, log).Servicio
	kaiak := debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple, Datos: catdomain.DatosProducto{
		SKU: "P1", Nombre: "Kaiak Clásico", PrecioVenta: d("59200")}}))
	jabon := debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple, Datos: catdomain.DatosProducto{
		SKU: "P2", Nombre: "Ekos Jabón", PrecioVenta: d("5500")}}))
	return &entornoMySQL{conn: conn, cat: cat, inv: inv, com: com, kaiak: kaiak, jabon: jabon}
}

func (e *entornoMySQL) registrar(t *testing.T, dia *time.Time) comapp.CompraDetalle {
	t.Helper()
	venc := fecha.DelInstante(time.Now(), reloj.ZonaArgentina).SumarDias(300)
	return debe(e.com.RegistrarCompra(context.Background(), comapp.ComandoCompra{Fecha: dia, Items: []comapp.ItemCmd{
		{ProductoID: e.kaiak.ID, Cantidad: 4, CostoUnitario: d("35520"), PrecioVentaNuevo: ptr(d("62000")), Vencimiento: &venc},
		{ProductoID: e.jabon.ID, Cantidad: 10, CostoUnitario: d("2100")},
	}}))
}

func itemsDe(c comapp.CompraDetalle, mod func(k, j *comapp.ItemEdicionCmd)) *[]comapp.ItemEdicionCmd {
	var k, j comapp.ItemEdicionCmd
	for _, it := range c.Items {
		cmd := comapp.ItemEdicionCmd{ID: it.ID, ProductoID: it.ProductoID, Cantidad: it.Cantidad, CostoUnitario: it.CostoUnitario,
			PrecioVentaNuevo: it.PrecioVentaNuevo, Vencimiento: it.Vencimiento}
		if it.Producto == "Kaiak Clásico" {
			k = cmd
		} else {
			j = cmd
		}
	}
	if mod != nil {
		mod(&k, &j)
	}
	return &[]comapp.ItemEdicionCmd{k, j}
}

func (e *entornoMySQL) editar(c comapp.CompraDetalle, mod func(k, j *comapp.ItemEdicionCmd)) (comapp.CompraDetalle, error) {
	dia := fecha.DelInstante(c.Fecha, reloj.ZonaArgentina)
	return e.com.EditarCompra(context.Background(), comapp.ComandoEditarCompra{ID: c.ID, Version: c.Version,
		Fecha: comapp.FechaEditada{Dia: &dia}, Items: itemsDe(c, mod), Origen: "WEB"})
}

func TestEditarRenglonesMySQL(t *testing.T) {
	e := nuevoEntornoMySQL(t)
	ctx := context.Background()
	pag := paginacion.Nueva(1, 20)
	c := e.registrar(t, nil)
	if c.Items[0].Consumido != 0 || c.Items[0].PrecioVentaActual == nil {
		t.Fatalf("detalle inicial = %+v", c.Items[0])
	}
	kaiakItem := c.Items[0].ID

	// Se vende 1 unidad de Kaiak (FEFO sobre el lote de la compra).
	debe(e.inv.RegistrarSalida(ctx, invapp.Salida{ProductoID: e.kaiak.ID, Cantidad: 1, Tipo: invdomain.MovVenta}))
	c = debe(e.com.ObtenerCompra(ctx, c.ID))
	if c.Items[0].Consumido != 1 {
		t.Fatalf("consumido = %d", c.Items[0].Consumido)
	}

	// Subir la cantidad 4 -> 6, costo y precio nuevos: es la compra más reciente.
	nuevoVenc := fecha.DelInstante(time.Now(), reloj.ZonaArgentina).SumarDias(500)
	r := debe(e.editar(c, func(k, _ *comapp.ItemEdicionCmd) {
		k.Cantidad, k.CostoUnitario, k.PrecioVentaNuevo, k.Vencimiento = 6, d("36000"), ptr(d("65000")), &nuevoVenc
	}))
	if r.Version != 2 || r.Total.String() != "237000.00" || r.Items[0].Cantidad != 6 || r.Items[0].Consumido != 1 {
		t.Fatalf("editada = %+v", r)
	}
	if n := stockDelRenglon(t, e.conn, kaiakItem); n != 5 { // 4 - 1 vendida + 2
		t.Errorf("stock del lote = %d", n)
	}
	movs := debe(e.inv.ListarMovimientos(ctx, invapp.FiltroMovimientos{Tipo: invdomain.MovEdicionCompra}, pag))
	if movs.Total != 1 || movs.Items[0].Cantidad != 2 || *movs.Items[0].ReferenciaID != c.ID || movs.Items[0].Producto != "Kaiak Clásico" {
		t.Errorf("movimientos EDICION_COMPRA = %+v", movs.Items)
	}
	st := debe(e.inv.StockDeProducto(ctx, e.kaiak.ID))
	if st.Stock != 5 || st.Lotes[0].Costo.String() != "36000.00" || st.Lotes[0].Vencimiento.String() != nuevoVenc.String() {
		t.Errorf("lote = %+v", st)
	}
	pk := debe(e.cat.ObtenerProducto(ctx, e.kaiak.ID))
	if pk.Costo.String() != "36000.00" || pk.PrecioVenta.String() != "65000.00" {
		t.Errorf("producto = %v %s", pk.Costo, pk.PrecioVenta)
	}
	hist := debe(e.cat.HistorialPrecios(ctx, e.kaiak.ID, pag))
	if hist.Total != 2 || hist.Items[0].Origen != catdomain.OrigenCompra || hist.Items[0].PrecioNuevo.String() != "65000.00" ||
		!strings.HasPrefix(hist.Items[0].Motivo, "Edición de compra del ") {
		t.Errorf("historial = %+v", hist.Items)
	}
	if r.Items[0].PrecioVentaActual.String() != "65000.00" {
		t.Errorf("precioVentaActual = %v", r.Items[0].PrecioVentaActual)
	}
	// Auditoría de renglones en el historial de ediciones.
	if len(r.Ediciones) != 1 || len(r.Ediciones[0].Renglones) != 1 || r.Ediciones[0].Renglones[0].Accion != "MODIFICADO" ||
		len(r.Ediciones[0].Renglones[0].Cambios) != 4 {
		t.Errorf("ediciones = %+v", r.Ediciones)
	}

	// Vaciar el vencimiento: el lote queda sin fecha.
	r = debe(e.editar(r, func(k, _ *comapp.ItemEdicionCmd) { k.Vencimiento = nil }))
	st = debe(e.inv.StockDeProducto(ctx, e.kaiak.ID))
	if r.Items[0].Vencimiento != nil || st.Lotes[0].Estado != invdomain.SinFecha {
		t.Errorf("vencimiento no se vació: %+v / %+v", r.Items[0], st.Lotes[0])
	}

	// Sin cambios: misma versión.
	s := debe(e.editar(r, nil))
	if s.Version != r.Version {
		t.Errorf("sin cambios subió la versión: %d", s.Version)
	}

	// Bloqueo: se vendieron 3 más (quedan 2 en el lote; salieron 4 de 6): bajar a 3 se rechaza listando los renglones.
	debe(e.inv.RegistrarSalida(ctx, invapp.Salida{ProductoID: e.kaiak.ID, Cantidad: 3, Tipo: invdomain.MovVenta}))
	debe(e.inv.AjustarStock(ctx, invapp.ComandoAjuste{ProductoID: e.jabon.ID, Cantidad: -9, Motivo: "rotura"}))
	antes := debe(e.com.ObtenerCompra(ctx, c.ID))
	_, err := e.editar(antes, func(k, j *comapp.ItemEdicionCmd) { k.Cantidad, j.Cantidad = 3, 5 })
	ee, ok := errores.Como(err)
	if !ok || ee.Codigo != "compra_consumida" || ee.Campos["items[0].cantidad"] == "" || ee.Campos["items[1].cantidad"] == "" {
		t.Fatalf("err = %v", err)
	}
	despues := debe(e.com.ObtenerCompra(ctx, c.ID))
	if despues.Version != antes.Version || despues.Items[0].Cantidad != 6 || despues.Items[1].Cantidad != 10 || stockDelRenglon(t, e.conn, kaiakItem) != 2 {
		t.Errorf("no debería haber cambiado nada: %+v", despues)
	}
	// Justo hasta lo consumido (6 - 2 = 4).
	debe(e.editar(antes, func(k, _ *comapp.ItemEdicionCmd) { k.Cantidad = 4 }))
	if n := stockDelRenglon(t, e.conn, kaiakItem); n != 0 {
		t.Errorf("stock = %d", n)
	}
}

func TestEditarRenglonNoEsLaMasRecienteMySQL(t *testing.T) {
	e := nuevoEntornoMySQL(t)
	ctx := context.Background()
	vieja := time.Now().AddDate(0, 0, -10)
	c1 := e.registrar(t, &vieja)
	e.registrar(t, nil) // más reciente: fijó costo 35.520 y precio 62.000
	// Editar la vieja no toca el producto.
	debe(e.editar(c1, func(k, _ *comapp.ItemEdicionCmd) { k.CostoUnitario, k.PrecioVentaNuevo = d("99999"), ptr(d("88888")) }))
	pk := debe(e.cat.ObtenerProducto(ctx, e.kaiak.ID))
	if pk.Costo.String() != "35520.00" || pk.PrecioVenta.String() != "62000.00" {
		t.Errorf("producto = %v %s", pk.Costo, pk.PrecioVenta)
	}
	// P-A1: registrar una compra con fecha anterior tampoco pisa el costo.
	antigua := time.Now().AddDate(0, 0, -30)
	debe(e.com.RegistrarCompra(ctx, comapp.ComandoCompra{Fecha: &antigua, Items: []comapp.ItemCmd{
		{ProductoID: e.kaiak.ID, Cantidad: 1, CostoUnitario: d("1"), PrecioVentaNuevo: ptr(d("2"))}}}))
	pk = debe(e.cat.ObtenerProducto(ctx, e.kaiak.ID))
	if pk.Costo.String() != "35520.00" || pk.PrecioVenta.String() != "62000.00" {
		t.Errorf("el alta antigua pisó el producto: %v %s", pk.Costo, pk.PrecioVenta)
	}
}

func TestEditarTodoONadaMySQL(t *testing.T) {
	e := nuevoEntornoMySQL(t)
	ctx := context.Background()
	c := e.registrar(t, nil)
	// El segundo renglón fallará por consumo: ni el primero (válido) ni la cabecera deben aplicarse.
	debe(e.inv.AjustarStock(ctx, invapp.ComandoAjuste{ProductoID: e.jabon.ID, Cantidad: -9, Motivo: "rotura"}))
	_, err := e.editar(c, func(k, j *comapp.ItemEdicionCmd) { k.Cantidad, k.CostoUnitario, j.Cantidad = 8, d("40000"), 5 })
	if err == nil {
		t.Fatal("debería rechazarse")
	}
	after := debe(e.com.ObtenerCompra(ctx, c.ID))
	if after.Version != 1 || after.Items[0].Cantidad != 4 || after.Items[0].CostoUnitario.String() != "35520.00" || len(after.Ediciones) != 0 {
		t.Errorf("quedó a medias: %+v", after)
	}
	pk := debe(e.cat.ObtenerProducto(ctx, e.kaiak.ID))
	if pk.Costo.String() != "35520.00" {
		t.Errorf("producto = %v", pk.Costo)
	}
}

// codigoPermitido indica si el error es uno de los esperados al perder una carrera.
func codigoPermitido(err error, codigos ...string) bool {
	ee, ok := errores.Como(err)
	if !ok {
		return false
	}
	for _, c := range codigos {
		if ee.Codigo == c {
			return true
		}
	}
	return false
}

func (e *entornoMySQL) productoNuevo(t *testing.T, sku string) catapp.ProductoDetalle {
	t.Helper()
	return debe(e.cat.CrearProducto(context.Background(), catapp.ComandoProducto{Tipo: catdomain.TipoSimple,
		Datos: catdomain.DatosProducto{SKU: sku, Nombre: "Prod " + sku, PrecioVenta: d("1000")}}))
}

// Escenario "Venta simultánea contra una baja": en un lote de 4, vender 3 y bajar a 2 (-2) no caben
// juntos. Se serializan: exactamente una de las dos gana y el lote nunca queda en negativo.
func TestVentaSimultaneaContraBajaMySQL(t *testing.T) {
	e := nuevoEntornoMySQL(t)
	ctx := context.Background()
	for i := 0; i < 8; i++ {
		prod := e.productoNuevo(t, fmt.Sprintf("R%d", i))
		c := debe(e.com.RegistrarCompra(ctx, comapp.ComandoCompra{Items: []comapp.ItemCmd{
			{ProductoID: prod.ID, Cantidad: 4, CostoUnitario: d("100")}}}))
		item := c.Items[0].ID
		dia := fecha.DelInstante(c.Fecha, reloj.ZonaArgentina)
		var wg sync.WaitGroup
		var errVenta, errEdicion error
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, errVenta = e.inv.RegistrarSalida(ctx, invapp.Salida{ProductoID: prod.ID, Cantidad: 3, Tipo: invdomain.MovVenta})
		}()
		go func() {
			defer wg.Done()
			_, errEdicion = e.com.EditarCompra(ctx, comapp.ComandoEditarCompra{ID: c.ID, Version: c.Version, Fecha: comapp.FechaEditada{Dia: &dia},
				Items: &[]comapp.ItemEdicionCmd{{ID: item, ProductoID: prod.ID, Cantidad: 2, CostoUnitario: d("100")}}})
		}()
		wg.Wait()
		stock := stockDelRenglon(t, e.conn, item)
		if stock < 0 {
			t.Fatalf("ronda %d: el lote quedó en negativo (%d); venta=%v edición=%v", i, stock, errVenta, errEdicion)
		}
		if (errVenta == nil) == (errEdicion == nil) {
			t.Fatalf("ronda %d: debía ganar exactamente una; venta=%v edición=%v (stock %d)", i, errVenta, errEdicion, stock)
		}
		if errVenta == nil && (stock != 1 || !codigoPermitido(errEdicion, "compra_consumida", "operacion_concurrente")) {
			t.Errorf("ronda %d: ganó la venta: stock=%d edición=%v", i, stock, errEdicion)
		}
		if errEdicion == nil && (stock != 2 || !codigoPermitido(errVenta, "stock_insuficiente", "operacion_concurrente")) {
			t.Errorf("ronda %d: ganó la edición: stock=%d venta=%v", i, stock, errVenta)
		}
	}
}

// Dos ventas simultáneas de productos DISTINTOS, con los lotes intercalados (ids alternados) y
// los renglones en orden inverso: no deben fallar (READ COMMITTED sin gap locks + reintento).
func TestVentasConcurrentesDeProductosDistintosMySQL(t *testing.T) {
	e := nuevoEntornoMySQL(t)
	ctx := context.Background()
	ven := ventas.NuevoModuloMySQL(e.conn, e.cat, e.inv, nil, slog.New(slog.NewTextHandler(io.Discard, nil))).Servicio
	// Lotes intercalados: A, B, A, B, A, B.
	items := []comapp.ItemCmd{}
	for range 3 {
		items = append(items,
			comapp.ItemCmd{ProductoID: e.kaiak.ID, Cantidad: 20, CostoUnitario: d("100")},
			comapp.ItemCmd{ProductoID: e.jabon.ID, Cantidad: 20, CostoUnitario: d("100")})
	}
	debe(e.com.RegistrarCompra(ctx, comapp.ComandoCompra{Items: items}))
	for ronda := 0; ronda < 5; ronda++ {
		var wg sync.WaitGroup
		errs := make([]error, 4)
		venta := func(i int, a, b string) {
			defer wg.Done()
			_, errs[i] = ven.RegistrarVenta(ctx, venapp.ComandoVenta{
				Items: []venapp.LineaCmd{{ProductoID: a, Cantidad: 1}, {ProductoID: b, Cantidad: 1}},
				Pagos: []vendomain.PagoIngresado{pago(vendomain.Efectivo, "100000")}})
		}
		wg.Add(4)
		go venta(0, e.kaiak.ID, e.jabon.ID)
		go venta(1, e.jabon.ID, e.kaiak.ID) // orden inverso
		go venta(2, e.kaiak.ID, e.jabon.ID)
		go venta(3, e.jabon.ID, e.kaiak.ID)
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Errorf("ronda %d: la venta %d no debía fallar: %v", ronda, i, err)
			}
		}
	}
	if debe(e.inv.StockDeProducto(ctx, e.kaiak.ID)).Stock != 40 || debe(e.inv.StockDeProducto(ctx, e.jabon.ID)).Stock != 40 {
		t.Error("cada producto debía quedar con 60 - 20 ventas = 40")
	}
}

// Dos renglones del mismo producto (dos lotes) contra una venta simultánea: ningún lote queda en negativo.
func TestDosRenglonesDelMismoProductoContraUnaVentaMySQL(t *testing.T) {
	e := nuevoEntornoMySQL(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		prod := e.productoNuevo(t, fmt.Sprintf("D%d", i))
		c := debe(e.com.RegistrarCompra(ctx, comapp.ComandoCompra{Items: []comapp.ItemCmd{
			{ProductoID: prod.ID, Cantidad: 4, CostoUnitario: d("100")},
			{ProductoID: prod.ID, Cantidad: 4, CostoUnitario: d("100")}}}))
		dia := fecha.DelInstante(c.Fecha, reloj.ZonaArgentina)
		var wg sync.WaitGroup
		var errVenta, errEdicion error
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, errVenta = e.inv.RegistrarSalida(ctx, invapp.Salida{ProductoID: prod.ID, Cantidad: 6, Tipo: invdomain.MovVenta})
		}()
		go func() {
			defer wg.Done()
			_, errEdicion = e.com.EditarCompra(ctx, comapp.ComandoEditarCompra{ID: c.ID, Version: c.Version, Fecha: comapp.FechaEditada{Dia: &dia},
				Items: &[]comapp.ItemEdicionCmd{
					{ID: c.Items[0].ID, ProductoID: prod.ID, Cantidad: 1, CostoUnitario: d("100")},
					{ID: c.Items[1].ID, ProductoID: prod.ID, Cantidad: 1, CostoUnitario: d("100")}}})
		}()
		wg.Wait()
		s0, s1 := stockDelRenglon(t, e.conn, c.Items[0].ID), stockDelRenglon(t, e.conn, c.Items[1].ID)
		if s0 < 0 || s1 < 0 {
			t.Fatalf("ronda %d: lote en negativo (%d, %d); venta=%v edición=%v", i, s0, s1, errVenta, errEdicion)
		}
		// Vender 6 de 8 y dejar los dos renglones en 1 (-6) no caben juntos: una de las dos pierde.
		if errVenta == nil && errEdicion == nil {
			t.Fatalf("ronda %d: ambas ganaron con stocks %d y %d", i, s0, s1)
		}
		if errVenta != nil && !codigoPermitido(errVenta, "stock_insuficiente", "operacion_concurrente") {
			t.Errorf("ronda %d: venta: %v", i, errVenta)
		}
		if errEdicion != nil && !codigoPermitido(errEdicion, "compra_consumida", "operacion_concurrente") {
			t.Errorf("ronda %d: edición: %v", i, errEdicion)
		}
	}
}
