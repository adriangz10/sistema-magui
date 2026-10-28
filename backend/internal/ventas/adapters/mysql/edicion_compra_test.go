package mysql_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"magui/internal/catalogo"
	catapp "magui/internal/catalogo/application"
	catdomain "magui/internal/catalogo/domain"
	"magui/internal/compras"
	comapp "magui/internal/compras/application"
	"magui/internal/inventario"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/reloj"
	"magui/internal/ventas"
	venapp "magui/internal/ventas/application"
	vendomain "magui/internal/ventas/domain"
)

// Regresión de ventas con el bloqueo de lotes (ADR 0002) y con compras editadas (US-001b):
// una venta toma stock de un lote cuya cantidad se corrigió, y anularla lo devuelve a ese lote.
func TestVentasSobreCompraEditadaMySQL(t *testing.T) {
	conn := baseTemporal(t)
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cat := catalogo.NuevoModuloMySQL(conn, log).Servicio
	inv := inventario.NuevoModuloMySQL(conn, cat, log).Servicio
	com := compras.NuevoModuloMySQL(conn, cat, inv, log).Servicio
	ven := ventas.NuevoModuloMySQL(conn, cat, inv, log).Servicio
	hoy := fecha.DelInstante(time.Now(), reloj.ZonaArgentina)

	kaiak := debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple,
		Datos: catdomain.DatosProducto{SKU: "P1", Nombre: "Kaiak", PrecioVenta: d("59200")}}))
	venc := hoy.SumarDias(200)
	c := debe(com.RegistrarCompra(ctx, comapp.ComandoCompra{Items: []comapp.ItemCmd{
		{ProductoID: kaiak.ID, Cantidad: 4, CostoUnitario: d("30000"), Vencimiento: &venc}}}))

	// La compra se corrige: 4 -> 6 unidades y costo 31.000.
	dia := fecha.DelInstante(c.Fecha, reloj.ZonaArgentina)
	c = debe(com.EditarCompra(ctx, comapp.ComandoEditarCompra{ID: c.ID, Version: c.Version, Fecha: comapp.FechaEditada{Dia: &dia},
		Items: &[]comapp.ItemEdicionCmd{{ID: c.Items[0].ID, ProductoID: kaiak.ID, Cantidad: 6, CostoUnitario: d("31000"), Vencimiento: &venc}}}))
	if debe(inv.StockDeProducto(ctx, kaiak.ID)).Stock != 6 {
		t.Fatal("la edición debería sumar 2 unidades")
	}

	// Venta de 5: sale del lote editado con su costo nuevo.
	v := debe(ven.RegistrarVenta(ctx, venapp.ComandoVenta{
		Items: []venapp.LineaCmd{{ProductoID: kaiak.ID, Cantidad: 5}},
		Pagos: []vendomain.PagoIngresado{pago(vendomain.Efectivo, "296000")}}))
	if v.CostoTotal.String() != "155000.00" {
		t.Errorf("costo de la venta = %s", v.CostoTotal)
	}
	// Ya salieron 5 de las 6: no se puede bajar por debajo.
	c = debe(com.ObtenerCompra(ctx, c.ID))
	if c.Items[0].Consumido != 5 {
		t.Errorf("consumido = %d", c.Items[0].Consumido)
	}
	// Anular la venta devuelve las unidades al mismo lote y libera el bloqueo por consumo.
	debe(ven.AnularVenta(ctx, v.ID, "Error de carga"))
	if c = debe(com.ObtenerCompra(ctx, c.ID)); c.Items[0].Consumido != 0 {
		t.Errorf("tras anular, consumido = %d", c.Items[0].Consumido)
	}
}
