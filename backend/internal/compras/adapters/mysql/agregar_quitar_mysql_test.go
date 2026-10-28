package mysql_test

import (
	"context"
	"testing"
	"time"

	catapp "magui/internal/catalogo/application"
	catdomain "magui/internal/catalogo/domain"
	comapp "magui/internal/compras/application"
	invapp "magui/internal/inventario/application"
	invdomain "magui/internal/inventario/domain"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
)

// US-001c sobre MySQL real (necesita MAGUI_TEST_MYSQL_DSN; sin él se omite).

func (e *entornoMySQL) editarLista(c comapp.CompraDetalle, items []comapp.ItemEdicionCmd) (comapp.CompraDetalle, error) {
	dia := fecha.DelInstante(c.Fecha, reloj.ZonaArgentina)
	return e.com.EditarCompra(context.Background(), comapp.ComandoEditarCompra{ID: c.ID, Version: c.Version,
		Fecha: comapp.FechaEditada{Dia: &dia}, Items: &items, Origen: "WEB"})
}

// itemDe arma el renglón existente de la compra para el producto dado.
func itemDe(c comapp.CompraDetalle, producto string) comapp.ItemEdicionCmd {
	for _, it := range c.Items {
		if it.Producto == producto {
			return comapp.ItemEdicionCmd{ID: it.ID, ProductoID: it.ProductoID, Cantidad: it.Cantidad, CostoUnitario: it.CostoUnitario,
				PrecioVentaNuevo: it.PrecioVentaNuevo, Vencimiento: it.Vencimiento}
		}
	}
	panic("sin renglón de " + producto)
}

func (e *entornoMySQL) cuenta(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := e.conn.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestAgregarYQuitarRenglonesMySQL(t *testing.T) {
	e := nuevoEntornoMySQL(t)
	ctx := context.Background()
	pag := paginacion.Nueva(1, 20)
	chronos := debe(e.cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple, Datos: catdomain.DatosProducto{
		SKU: "P3", Nombre: "Chronos Crema", PrecioVenta: d("20000")}}))

	c1 := e.registrar(t, nil)
	// Una compra anterior de Kaiak a 33.000: pasará a ser la más reciente si se quita Kaiak de C1.
	hace10 := time.Now().AddDate(0, 0, -10)
	debe(e.com.RegistrarCompra(ctx, comapp.ComandoCompra{Fecha: &hace10, Items: []comapp.ItemCmd{
		{ProductoID: e.kaiak.ID, Cantidad: 2, CostoUnitario: d("33000")}}}))
	if p := debe(e.cat.ObtenerProducto(ctx, e.kaiak.ID)); p.Costo.String() != "35520.00" {
		t.Fatalf("la compra vieja no debe pisar el costo: %v", p.Costo)
	}

	// Agregar Chronos: lote nuevo con COMPRA +2.
	venc := fecha.DelInstante(time.Now(), reloj.ZonaArgentina).SumarDias(400)
	nuevo := comapp.ItemEdicionCmd{ProductoID: chronos.ID, Cantidad: 2, CostoUnitario: d("15000"), PrecioVentaNuevo: ptr(d("24000")), Vencimiento: &venc}
	c2 := debe(e.editarLista(c1, []comapp.ItemEdicionCmd{itemDe(c1, "Kaiak Clásico"), itemDe(c1, "Ekos Jabón"), nuevo}))
	if c2.Version != 2 || c2.Productos != 3 || c2.Total.String() != "193080.00" {
		t.Fatalf("tras agregar = %+v", c2.CompraResumen)
	}
	if st := debe(e.inv.StockDeProducto(ctx, chronos.ID)); st.Stock != 2 || len(st.Lotes) != 1 {
		t.Errorf("stock Chronos = %+v", st)
	}
	if p := debe(e.cat.ObtenerProducto(ctx, chronos.ID)); p.Costo.String() != "15000.00" || p.PrecioVenta.String() != "24000.00" {
		t.Errorf("Chronos = costo %v precio %v", p.Costo, p.PrecioVenta)
	}
	if len(c2.Ediciones) != 1 || c2.Ediciones[0].Renglones[0].Accion != "AGREGADO" {
		t.Errorf("ediciones = %+v", c2.Ediciones)
	}

	// Quitar Ekos Jabón (sin consumo): EDICION_COMPRA -10, renglón dado de baja, lote y movimientos conservados.
	jabonItem := itemDe(c2, "Ekos Jabón").ID
	c3 := debe(e.editarLista(c2, []comapp.ItemEdicionCmd{itemDe(c2, "Kaiak Clásico"), itemDe(c2, "Chronos Crema")}))
	if c3.Version != 3 || c3.Productos != 2 || c3.Total.String() != "172080.00" || len(c3.Items) != 2 {
		t.Fatalf("tras quitar = %+v", c3.CompraResumen)
	}
	for _, it := range c3.Items {
		if it.ID == jabonItem {
			t.Error("el renglón quitado no debe aparecer en el detalle")
		}
	}
	if n := e.cuenta(t, `SELECT COUNT(*) FROM compra_item WHERE id = ? AND quitado_en IS NOT NULL`, jabonItem); n != 1 {
		t.Errorf("el renglón debe seguir en la base marcado como quitado (%d)", n)
	}
	if n := e.cuenta(t, `SELECT COUNT(*) FROM lote WHERE compra_item_id = ?`, jabonItem); n != 1 {
		t.Errorf("el lote debe conservar su vínculo (%d)", n)
	}
	if n := e.cuenta(t, `SELECT COUNT(*) FROM movimiento_stock m JOIN lote l ON l.id = m.lote_id WHERE l.compra_item_id = ?`, jabonItem); n != 2 {
		t.Errorf("movimientos del lote = %d (COMPRA + EDICION_COMPRA)", n)
	}
	if n := stockDelRenglon(t, e.conn, jabonItem); n != 0 {
		t.Errorf("stock del lote quitado = %d", n)
	}
	if st := debe(e.inv.StockDeProducto(ctx, e.jabon.ID)); st.Stock != 0 {
		t.Errorf("stock de Ekos = %d", st.Stock)
	}
	// Ekos no tiene otra compra: se mantiene su último costo.
	if p := debe(e.cat.ObtenerProducto(ctx, e.jabon.ID)); p.Costo == nil || p.Costo.String() != "2100.00" {
		t.Errorf("costo de Ekos = %v", p.Costo)
	}
	// Listado y resumen sin el renglón quitado.
	lista := debe(e.com.ListarCompras(ctx, comapp.Filtro{}, pag))
	var visto bool
	for _, r := range lista.Items {
		if r.ID == c1.ID {
			visto = true
			if r.Productos != 2 || r.Unidades != 6 {
				t.Errorf("resumen = %+v", r)
			}
		}
	}
	if !visto {
		t.Fatal("la compra no está en el listado")
	}
	movs := debe(e.inv.ListarMovimientos(ctx, invapp.FiltroMovimientos{Tipo: invdomain.MovEdicionCompra}, pag))
	if movs.Total != 1 || movs.Items[0].Cantidad != -10 {
		t.Errorf("EDICION_COMPRA = %+v", movs.Items)
	}

	// Kaiak con una unidad vendida: no se puede quitar (409 con quitados[<id>]) y no cambia nada.
	debe(e.inv.RegistrarSalida(ctx, invapp.Salida{ProductoID: e.kaiak.ID, Cantidad: 1, Tipo: invdomain.MovVenta}))
	kaiakItem := itemDe(c3, "Kaiak Clásico").ID
	_, err := e.editarLista(c3, []comapp.ItemEdicionCmd{itemDe(c3, "Chronos Crema")})
	ee, ok := errores.Como(err)
	if !ok || ee.Codigo != "compra_consumida" || ee.Campos["quitados["+kaiakItem+"]"] == "" {
		t.Fatalf("err = %v", err)
	}
	if again := debe(e.com.ObtenerCompra(ctx, c1.ID)); again.Version != 3 || again.Productos != 2 {
		t.Errorf("no debía cambiar nada: %+v", again.CompraResumen)
	}

	// Devuelta la unidad con un ajuste (el lote vuelve a 4 de 4), se puede quitar: Kaiak pasa a la compra anterior (33.000), el precio no se revierte.
	debe(e.inv.AjustarStock(ctx, invapp.ComandoAjuste{ProductoID: e.kaiak.ID, Cantidad: 1, Motivo: "devuelto", LoteID: loteDeItem(t, e, kaiakItem)}))
	c3 = debe(e.com.ObtenerCompra(ctx, c1.ID))
	// El ajuste positivo deja el lote en 4 de 4: sin consumo.
	c4 := debe(e.editarLista(c3, []comapp.ItemEdicionCmd{itemDe(c3, "Chronos Crema")}))
	if c4.Version != 4 || c4.Productos != 1 {
		t.Fatalf("tras quitar Kaiak = %+v", c4.CompraResumen)
	}
	pk := debe(e.cat.ObtenerProducto(ctx, e.kaiak.ID))
	if pk.Costo.String() != "33000.00" || pk.PrecioVenta.String() != "62000.00" {
		t.Errorf("Kaiak = costo %v precio %v (esperado 33000 y 62000)", pk.Costo, pk.PrecioVenta)
	}

	// No se puede dejar la compra vacía.
	_, err = e.editarLista(c4, []comapp.ItemEdicionCmd{})
	if ee, ok := errores.Como(err); !ok || ee.Campos["items"] == "" {
		t.Errorf("vacía: %v", err)
	}
}

func loteDeItem(t *testing.T, e *entornoMySQL, itemID string) string {
	t.Helper()
	var id string
	if err := e.conn.QueryRow(`SELECT id FROM lote WHERE compra_item_id = ?`, itemID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCambiarFechaRecalculaUltimoCostoMySQL(t *testing.T) {
	e := nuevoEntornoMySQL(t)
	ctx := context.Background()
	c1 := e.registrar(t, nil) // hoy, Kaiak a 35.520
	hace10 := time.Now().AddDate(0, 0, -10)
	debe(e.com.RegistrarCompra(ctx, comapp.ComandoCompra{Fecha: &hace10, Items: []comapp.ItemCmd{
		{ProductoID: e.kaiak.ID, Cantidad: 2, CostoUnitario: d("33000")}}}))

	// Mover C1 a hace 20 días: la compra de hace 10 pasa a ser la más reciente.
	dia := fecha.DelInstante(time.Now().AddDate(0, 0, -20), reloj.ZonaArgentina)
	c2 := debe(e.com.EditarCompra(ctx, comapp.ComandoEditarCompra{ID: c1.ID, Version: c1.Version, Fecha: comapp.FechaEditada{Dia: &dia}}))
	if c2.Version != 2 {
		t.Fatalf("version = %d", c2.Version)
	}
	pk := debe(e.cat.ObtenerProducto(ctx, e.kaiak.ID))
	if pk.Costo.String() != "33000.00" || pk.PrecioVenta.String() != "62000.00" {
		t.Errorf("tras mover atrás: costo %v precio %v", pk.Costo, pk.PrecioVenta)
	}
	// Y de vuelta a hoy: C1 vuelve a ser la más reciente.
	hoy := fecha.DelInstante(time.Now(), reloj.ZonaArgentina)
	c3 := debe(e.com.EditarCompra(ctx, comapp.ComandoEditarCompra{ID: c1.ID, Version: c2.Version, Fecha: comapp.FechaEditada{Dia: &hoy}}))
	pk = debe(e.cat.ObtenerProducto(ctx, e.kaiak.ID))
	if c3.Version != 3 || pk.Costo.String() != "35520.00" {
		t.Errorf("tras volver: version %d costo %v", c3.Version, pk.Costo)
	}
}

func TestProductoInactivoMySQL(t *testing.T) {
	e := nuevoEntornoMySQL(t)
	ctx := context.Background()
	viejo := debe(e.cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple, Datos: catdomain.DatosProducto{
		SKU: "P9", Nombre: "Viejo", PrecioVenta: d("1000")}}))
	c1 := e.registrar(t, nil)
	debe(e.cat.CambiarEstadoProducto(ctx, viejo.ID, false))
	// Alta con un inactivo: 422 producto_inactivo.
	_, err := e.com.RegistrarCompra(ctx, comapp.ComandoCompra{Items: []comapp.ItemCmd{
		{ProductoID: viejo.ID, Cantidad: 1, CostoUnitario: d("10")}}})
	if ee, ok := errores.Como(err); !ok || ee.Codigo != "producto_inactivo" || ee.Campos["items[0].productoId"] == "" {
		t.Fatalf("alta: %v", err)
	}
	// Agregar un inactivo en la edición: 422 y no cambia nada.
	nuevo := comapp.ItemEdicionCmd{ProductoID: viejo.ID, Cantidad: 1, CostoUnitario: d("10")}
	_, err = e.editarLista(c1, []comapp.ItemEdicionCmd{itemDe(c1, "Kaiak Clásico"), itemDe(c1, "Ekos Jabón"), nuevo})
	if ee, ok := errores.Como(err); !ok || ee.Codigo != "producto_inactivo" || ee.Campos["items[2].productoId"] == "" {
		t.Fatalf("edición: %v", err)
	}
	// Un renglón existente de un producto desactivado luego se puede editar.
	debe(e.cat.CambiarEstadoProducto(ctx, e.jabon.ID, false))
	j := itemDe(c1, "Ekos Jabón")
	j.Cantidad = 11
	if c2 := debe(e.editarLista(c1, []comapp.ItemEdicionCmd{itemDe(c1, "Kaiak Clásico"), j})); c2.Version != 2 {
		t.Errorf("version = %d", c2.Version)
	}
}
