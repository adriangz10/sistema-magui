package application

import (
	"errors"
	"testing"
	"time"

	"magui/internal/compras/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
)

// Compra C1 de la historia US-001b: Kaiak 4 x 35.520 (vence 2027-09-30, precio 62.000) y Jabón 10 x 2.100.
type entornoB struct {
	s        *Servicio
	repo     *repoMem
	cat      *catalogoMem
	inv      *inventarioMem
	id       string
	kaiak    string // id de renglón
	jabon    string
	venc     fecha.Fecha
	version  int
	fechaStr string
}

func nuevoB(t *testing.T) *entornoB {
	t.Helper()
	s, repo, cat, inv := entorno()
	costoK := d("35520")
	cat.productos["kaiak"] = ProductoCompra{Nombre: "Kaiak Clásico", Costo: &costoK, PrecioVenta: d("62000")}
	costoJ := d("2100")
	cat.productos["jabon"] = ProductoCompra{Nombre: "Ekos Jabón", Costo: &costoJ, PrecioVenta: d("3000")}
	venc, _ := fecha.Parse("2027-09-30")
	precio := d("62000")
	det, err := s.RegistrarCompra(ctx, ComandoCompra{Items: []ItemCmd{
		{ProductoID: "kaiak", Cantidad: 4, CostoUnitario: costoK, PrecioVentaNuevo: &precio, Vencimiento: &venc},
		{ProductoID: "jabon", Cantidad: 10, CostoUnitario: costoJ},
	}})
	if err != nil {
		t.Fatal(err)
	}
	c := repo.compras[det.ID]
	cat.costos, cat.motivo = nil, ""
	return &entornoB{s: s, repo: repo, cat: cat, inv: inv, id: det.ID, kaiak: c.Items[0].ID, jabon: c.Items[1].ID, venc: venc, version: 1}
}

// items arma la lista completa de la compra sin cambios y deja que el test modifique cada renglón.
func (e *entornoB) items(mod func(k, j *ItemEdicionCmd)) *[]ItemEdicionCmd {
	precio := d("62000")
	venc := e.venc
	k := ItemEdicionCmd{ID: e.kaiak, ProductoID: "kaiak", Cantidad: 4, CostoUnitario: d("35520"), PrecioVentaNuevo: &precio, Vencimiento: &venc}
	j := ItemEdicionCmd{ID: e.jabon, ProductoID: "jabon", Cantidad: 10, CostoUnitario: d("2100")}
	if mod != nil {
		mod(&k, &j)
	}
	return &[]ItemEdicionCmd{k, j}
}

func (e *entornoB) editar(mod func(k, j *ItemEdicionCmd)) (CompraDetalle, error) {
	return e.s.EditarCompra(ctx, ComandoEditarCompra{ID: e.id, Version: e.version, Fecha: FechaEditada{Instante: &ahora},
		Items: e.items(mod), Motivo: "Error de carga"})
}

// consumir simula ventas/ajustes: deja el stock del lote del renglón en `stock`.
func (e *entornoB) consumir(item string, stock int) {
	l := e.inv.lotes[item]
	l.Stock = stock
	e.inv.lotes[item] = l
}

// otraCompra agrega otra compra de Kaiak con la fecha y el created_at indicados.
func (e *entornoB) otraCompra(id string, fechaC, creada time.Time) {
	e.repo.compras[id] = domain.Compra{ID: id, Fecha: fechaC, CreatedAt: creada, Version: 1,
		Items: []domain.Item{{ID: id + "-i", ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("30000")}}}
}

func nadaEscrito(t *testing.T, e *entornoB) {
	t.Helper()
	if e.repo.escrituras != 0 || e.repo.itemsEscritos != 0 || len(e.repo.ediciones) != 0 || len(e.inv.ajustes) != 0 || len(e.cat.costos) != 0 {
		t.Errorf("no se debería haber escrito nada: compra=%d items=%d ediciones=%d ajustes=%d costos=%d",
			e.repo.escrituras, e.repo.itemsEscritos, len(e.repo.ediciones), len(e.inv.ajustes), len(e.cat.costos))
	}
}

func TestEditarRenglonSubirCantidad(t *testing.T) {
	e := nuevoB(t)
	det, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 6 })
	if err != nil {
		t.Fatal(err)
	}
	if det.Version != 2 || det.Total.String() != "234120.00" {
		t.Errorf("detalle = %+v", det.CompraResumen)
	}
	if len(e.inv.ajustes) != 1 {
		t.Fatalf("ajustes = %+v", e.inv.ajustes)
	}
	a := e.inv.ajustes[0]
	if a.LoteID != "lote-"+e.kaiak || a.Diferencia != 2 || a.CompraID != e.id || a.Nota != "Edición de compra del 25/10/2026" {
		t.Errorf("ajuste = %+v", a)
	}
	if len(e.cat.costos) != 0 {
		t.Errorf("solo cambió la cantidad: el catálogo no se toca (%+v)", e.cat.costos)
	}
	if e.inv.bloqueos == 0 {
		t.Error("debería leer los lotes de forma bloqueante")
	}
	// Auditoría: cantidad 4 -> 6.
	r := e.repo.ediciones[0].Cambios.Renglones
	if len(r) != 1 || r[0].Accion != "MODIFICADO" || r[0].Producto != "Kaiak Clásico" || r[0].Cambios[0].Campo != "cantidad" ||
		r[0].Cambios[0].Antes != 4 || r[0].Cambios[0].Despues != 6 {
		t.Errorf("renglones = %+v", r)
	}
}

func TestEditarRenglonBajarSinSuperarElConsumo(t *testing.T) {
	e := nuevoB(t)
	e.consumir(e.kaiak, 3) // se vendió 1
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 3 }); err != nil {
		t.Fatal(err)
	}
	if len(e.inv.ajustes) != 1 || e.inv.ajustes[0].Diferencia != -1 {
		t.Errorf("ajustes = %+v", e.inv.ajustes)
	}
}

func TestEditarRenglonBajarJustoHastaLoConsumido(t *testing.T) {
	e := nuevoB(t)
	e.consumir(e.kaiak, 1) // salieron 3
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 3 }); err != nil {
		t.Fatalf("debería aceptarse: %v", err)
	}
}

func TestEditarRenglonBloqueoPorConsumoNoEscribeNada(t *testing.T) {
	e := nuevoB(t)
	e.consumir(e.kaiak, 1)
	e.consumir(e.jabon, 2) // salieron 8
	// Un renglón válido (costo) junto con dos rechazados: todo o nada.
	_, err := e.editar(func(k, j *ItemEdicionCmd) {
		k.Cantidad, j.Cantidad = 2, 5
		k.CostoUnitario = d("36000")
	})
	ee, ok := errores.Como(err)
	if !ok || ee.Codigo != "compra_consumida" || ee.Tipo != errores.TipoConflicto {
		t.Fatalf("err = %v", err)
	}
	if ee.Campos["items[0].cantidad"] == "" || ee.Campos["items[1].cantidad"] == "" || len(ee.Campos) != 2 {
		t.Errorf("deberían listarse los dos renglones: %v", ee.Campos)
	}
	nadaEscrito(t, e)
	if e.repo.compras[e.id].Version != 1 {
		t.Error("la versión no debería cambiar")
	}
	// La base errores.Conflicto no se contamina para el próximo pedido.
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 6 }); err != nil {
		t.Errorf("tras un rechazo, una edición válida debe pasar: %v", err)
	}
}

func TestEditarRenglonCostoYVencimientoSinMovimiento(t *testing.T) {
	e := nuevoB(t)
	nuevoVenc, _ := fecha.Parse("2027-12-31")
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.CostoUnitario = d("36000"); k.Vencimiento = &nuevoVenc }); err != nil {
		t.Fatal(err)
	}
	a := e.inv.ajustes[0]
	if a.Diferencia != 0 || a.Costo.String() != "36000.00" || a.Vencimiento.String() != "2027-12-31" {
		t.Errorf("ajuste = %+v", a)
	}
}

// El frontend manda vencimiento null para vaciarlo: el lote queda sin vencimiento.
func TestEditarRenglonVencimientoNullDejaElLoteSinVencimiento(t *testing.T) {
	e := nuevoB(t)
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Vencimiento = nil }); err != nil {
		t.Fatal(err)
	}
	if len(e.inv.ajustes) != 1 || e.inv.ajustes[0].Vencimiento != nil {
		t.Fatalf("ajustes = %+v", e.inv.ajustes)
	}
	if e.repo.compras[e.id].Items[0].Vencimiento != nil {
		t.Error("el renglón debería quedar sin vencimiento")
	}
	r := e.repo.ediciones[0].Cambios.Renglones[0].Cambios[0]
	if r.Campo != "vencimiento" || r.Antes != "2027-09-30" || r.Despues != nil {
		t.Errorf("auditoría = %+v", r)
	}
}

func TestEditarRenglonEsLaMasRecienteActualizaCostoYPrecio(t *testing.T) {
	e := nuevoB(t)
	p := d("65000")
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.CostoUnitario = d("36000"); k.PrecioVentaNuevo = &p }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 1 {
		t.Fatalf("costos = %+v", e.cat.costos)
	}
	c := e.cat.costos[0]
	if c.ProductoID != "kaiak" || c.Costo.String() != "36000.00" || c.PrecioVenta == nil || c.PrecioVenta.String() != "65000.00" {
		t.Errorf("costo = %+v", c)
	}
	if e.cat.motivo != "Edición de compra del 25/10/2026" {
		t.Errorf("motivo = %q", e.cat.motivo)
	}
}

func TestEditarRenglonNoEsLaMasReciente(t *testing.T) {
	e := nuevoB(t)
	e.otraCompra("otra", ahora.AddDate(0, 0, 5), ahora)
	p := d("65000")
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.CostoUnitario = d("36000"); k.PrecioVentaNuevo = &p }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 0 {
		t.Errorf("no es la más reciente: el producto no se toca (%+v)", e.cat.costos)
	}
	if len(e.inv.ajustes) != 1 || e.inv.ajustes[0].Costo.String() != "36000.00" {
		t.Errorf("el lote sí se actualiza: %+v", e.inv.ajustes)
	}
}

func TestEditarRenglonDesempatePorCreatedAt(t *testing.T) {
	// Otra compra con la misma fecha pero creada antes: C1 es la más reciente.
	e := nuevoB(t)
	e.otraCompra("antes", ahora, ahora.Add(-time.Hour))
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.CostoUnitario = d("36000") }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 1 {
		t.Errorf("C1 debería ser la más reciente: %+v", e.cat.costos)
	}
	// Si la otra se creó después, ya no lo es.
	e = nuevoB(t)
	e.otraCompra("despues", ahora, ahora.Add(time.Hour))
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.CostoUnitario = d("36000") }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 0 {
		t.Errorf("la otra es más reciente: %+v", e.cat.costos)
	}
}

func TestEditarRenglonPrecioSinCambioNoSeReaplica(t *testing.T) {
	// Alguien puso a mano el precio en 70.000; se edita solo la cantidad.
	e := nuevoB(t)
	k := e.cat.productos["kaiak"]
	k.PrecioVenta = d("70000")
	e.cat.productos["kaiak"] = k
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 5 }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 0 {
		t.Errorf("costos = %+v", e.cat.costos)
	}
	// Se edita solo el costo: el costo se actualiza y el precio del renglón (sin cambio) no se reaplica.
	e = nuevoB(t)
	k = e.cat.productos["kaiak"]
	k.PrecioVenta = d("70000")
	e.cat.productos["kaiak"] = k
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.CostoUnitario = d("36000") }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 1 || e.cat.costos[0].PrecioVenta != nil {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

func TestEditarRenglonVaciarPrecioNoRevierte(t *testing.T) {
	e := nuevoB(t)
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.PrecioVentaNuevo = nil }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 0 {
		t.Errorf("vaciar el precio no cambia el producto: %+v", e.cat.costos)
	}
	if len(e.repo.ediciones) != 1 || e.repo.ediciones[0].Cambios.Renglones[0].Cambios[0].Campo != "precioVentaNuevo" {
		t.Errorf("el cambio sí se audita: %+v", e.repo.ediciones)
	}
	// Con el costo cambiado a la vez, se actualiza el costo pero no el precio.
	e = nuevoB(t)
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.PrecioVentaNuevo = nil; k.CostoUnitario = d("36000") }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 1 || e.cat.costos[0].PrecioVenta != nil {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

func TestEditarRenglonPrecioIgualAlActualNoEmite(t *testing.T) {
	e := nuevoB(t)
	k := e.cat.productos["kaiak"]
	k.PrecioVenta = d("65000")
	e.cat.productos["kaiak"] = k
	p := d("65000")
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.PrecioVentaNuevo = &p }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 0 {
		t.Errorf("el precio ya es el actual y el costo no cambió: %+v", e.cat.costos)
	}
}

func TestEditarRenglonVariosRenglonesDelMismoProductoTomaElUltimo(t *testing.T) {
	e := nuevoB(t)
	// Segundo renglón de Kaiak directamente en el repo (el alta ya lo permite).
	c := e.repo.compras[e.id]
	c.Items = append(c.Items, domain.Item{ID: "zzz", ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("40000")})
	e.repo.compras[e.id] = c
	e.inv.lotes["zzz"] = domain.LoteRenglon{LoteID: "lote-zzz", Stock: 1}
	precio, venc := d("62000"), e.venc
	items := &[]ItemEdicionCmd{
		{ID: e.kaiak, ProductoID: "kaiak", Cantidad: 4, CostoUnitario: d("36000"), PrecioVentaNuevo: &precio, Vencimiento: &venc},
		{ID: e.jabon, ProductoID: "jabon", Cantidad: 10, CostoUnitario: d("2100")},
		{ID: "zzz", ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("40000")},
	}
	if _, err := e.s.EditarCompra(ctx, ComandoEditarCompra{ID: e.id, Version: 1, Fecha: FechaEditada{Instante: &ahora}, Items: items}); err != nil {
		t.Fatal(err)
	}
	// El último renglón de Kaiak (zzz) no cambió: el costo vigente sigue siendo 40.000.
	if len(e.cat.costos) != 1 || e.cat.costos[0].Costo.String() != "40000.00" {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

func TestEditarRenglonSinCambiosNoEscribe(t *testing.T) {
	e := nuevoB(t)
	det, err := e.editar(nil)
	if err != nil {
		t.Fatal(err)
	}
	if det.Version != 1 {
		t.Errorf("version = %d", det.Version)
	}
	nadaEscrito(t, e)
}

func TestEditarRenglonErrores(t *testing.T) {
	e := nuevoB(t)
	// valores inválidos
	_, err := e.editar(func(k, j *ItemEdicionCmd) { k.Cantidad = 0; j.CostoUnitario = d("-1") })
	if ee, _ := errores.Como(err); ee == nil || ee.Campos["items[0].cantidad"] == "" || ee.Campos["items[1].costoUnitario"] == "" {
		t.Errorf("valores: %v", err)
	}
	// producto distinto
	_, err = e.editar(func(k, _ *ItemEdicionCmd) { k.ProductoID = "jabon" })
	if ee, _ := errores.Como(err); ee == nil || ee.Campos["items[0].productoId"] == "" {
		t.Errorf("producto: %v", err)
	}
	// renglón de otra compra
	_, err = e.editar(func(k, _ *ItemEdicionCmd) { k.ID = "ajeno" })
	if ee, _ := errores.Como(err); ee == nil || ee.Campos["items[0].id"] == "" {
		t.Errorf("id ajeno: %v", err)
	}
	// versión vieja con items
	e.version = 9
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 5 }); !errors.Is(err, domain.ErrVersionDesactualizada) {
		t.Errorf("versión: %v", err)
	}
	nadaEscrito(t, e)
}

func TestEditarRenglonPropagaFallasDeInventarioYAuditoria(t *testing.T) {
	e := nuevoB(t)
	e.inv.falla = errors.New("lote bloqueado")
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 5 }); err == nil || err.Error() != "lote bloqueado" {
		t.Errorf("inventario: %v", err)
	}
	e = nuevoB(t)
	e.repo.falloEdicion = errors.New("disco lleno")
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 5 }); err == nil || err.Error() != "disco lleno" {
		t.Errorf("auditoría: %v", err)
	}
	// El revertir real lo hace la transacción (ver mysql_test).
}

func TestEditarRenglonProductoInactivoSeAcepta(t *testing.T) {
	// El caso de uso no consulta el estado del producto: la edición no lo reactiva (RN-EC-14).
	e := nuevoB(t)
	if _, err := e.editar(func(_, j *ItemEdicionCmd) { j.Cantidad = 12 }); err != nil {
		t.Fatal(err)
	}
	if len(e.inv.ajustes) != 1 || e.inv.ajustes[0].Diferencia != 2 || len(e.cat.costos) != 0 {
		t.Errorf("ajustes=%+v costos=%+v", e.inv.ajustes, e.cat.costos)
	}
}

func TestObtenerCompraCalculaConsumido(t *testing.T) {
	e := nuevoB(t)
	e.consumir(e.kaiak, 1)
	e.consumir(e.jabon, 12) // stock mayor que la cantidad (ajuste positivo): nunca negativo
	det, err := e.s.ObtenerCompra(ctx, e.id)
	if err != nil {
		t.Fatal(err)
	}
	if det.Items[0].Consumido != 3 || det.Items[1].Consumido != 0 {
		t.Errorf("consumido = %d, %d", det.Items[0].Consumido, det.Items[1].Consumido)
	}
}

// P-A1: el alta solo actualiza último costo y precio si es la compra más reciente del producto.
func TestRegistrarCompraSoloActualizaSiEsLaMasReciente(t *testing.T) {
	s, repo, cat, _ := entorno()
	repo.compras["nueva"] = domain.Compra{ID: "nueva", Fecha: ahora, CreatedAt: ahora, Version: 1,
		Items: []domain.Item{{ID: "n1", ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("1")}}}
	vieja := ahora.AddDate(0, 0, -10)
	precio := d("65000")
	_, err := s.RegistrarCompra(ctx, ComandoCompra{Fecha: &vieja, Items: []ItemCmd{
		{ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("100"), PrecioVentaNuevo: &precio},
		{ProductoID: "jabon", Cantidad: 1, CostoUnitario: d("50")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	// Kaiak ya tenía una compra más nueva: no se toca. Jabón sí.
	if len(cat.costos) != 1 || cat.costos[0].ProductoID != "jabon" || cat.costos[0].Costo.String() != "50.00" {
		t.Errorf("costos = %+v", cat.costos)
	}
}

func TestRegistrarCompraRenglonesRepetidosUsaElUltimo(t *testing.T) {
	s, _, cat, _ := entorno()
	p1, p2 := d("60000"), d("62000")
	_, err := s.RegistrarCompra(ctx, ComandoCompra{Items: []ItemCmd{
		{ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("100"), PrecioVentaNuevo: &p1},
		{ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("120"), PrecioVentaNuevo: &p2},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.costos) != 1 || cat.costos[0].Costo.String() != "120.00" || cat.costos[0].PrecioVenta.String() != "62000.00" {
		t.Errorf("costos = %+v", cat.costos)
	}
}

var _ = dinero.Cero

func TestObtenerCompraTraePrecioVentaActual(t *testing.T) {
	e := nuevoB(t)
	k := e.cat.productos["kaiak"]
	k.PrecioVenta = d("70000") // alguien cambió el precio a mano después de la compra
	e.cat.productos["kaiak"] = k
	e.cat.consultas = 0
	det, err := e.s.ObtenerCompra(ctx, e.id)
	if err != nil {
		t.Fatal(err)
	}
	if det.Items[0].PrecioVentaActual == nil || det.Items[0].PrecioVentaActual.String() != "70000.00" ||
		det.Items[1].PrecioVentaActual.String() != "3000.00" {
		t.Errorf("precios = %v, %v", det.Items[0].PrecioVentaActual, det.Items[1].PrecioVentaActual)
	}
	if e.cat.consultas != 1 {
		t.Errorf("debería ser una sola consulta al catálogo para todos los productos, fueron %d", e.cat.consultas)
	}
	// Producto desconocido para el catálogo: null.
	delete(e.cat.productos, "jabon")
	det, _ = e.s.ObtenerCompra(ctx, e.id)
	if det.Items[1].PrecioVentaActual != nil {
		t.Error("producto sin ficha debería dar null")
	}
}

// OBS-1 / RN-EC-15: "más reciente" compara el DÍA argentino de la fecha; el mismo día se desempata
// por created_at (no por la hora de la fecha) y después por id.
func TestMasRecientePorDiaArgentino(t *testing.T) {
	// C1: 25/10 12:00 AR (15:00Z), creada en `ahora`. Otra compra del mismo día pero con la hora
	// más tarde (18:00Z = 15:00 AR) y creada ANTES: por día igual, gana C1 (creada después).
	e := nuevoB(t)
	e.otraCompra("otra", ahora.Add(3*time.Hour), ahora.Add(-time.Hour))
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.CostoUnitario = d("36000") }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 1 {
		t.Errorf("mismo día: C1 es la más reciente por created_at (%+v)", e.cat.costos)
	}
	// Una compra del día siguiente en Argentina (aunque sea 02:00Z, todavía 23:00 AR del mismo día
	// no cuenta): 26/10 04:00Z = 01:00 AR del 26 -> día posterior, esa es la más reciente.
	e = nuevoB(t)
	e.otraCompra("sig", time.Date(2026, 10, 26, 4, 0, 0, 0, time.UTC), ahora.Add(-time.Hour))
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.CostoUnitario = d("36000") }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 0 {
		t.Errorf("día posterior: no es la más reciente (%+v)", e.cat.costos)
	}
	// 26/10 02:00Z = 23:00 AR del 25: sigue siendo el mismo día argentino; se desempata por created_at.
	e = nuevoB(t)
	e.otraCompra("tarde", time.Date(2026, 10, 26, 2, 0, 0, 0, time.UTC), ahora.Add(-time.Hour))
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.CostoUnitario = d("36000") }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 1 {
		t.Errorf("23:00 AR es el mismo día: gana C1 por created_at (%+v)", e.cat.costos)
	}
}

func TestEditarSinLoteEsErrorTipado(t *testing.T) {
	e := nuevoB(t)
	delete(e.inv.lotes, e.kaiak)
	_, err := e.editar(func(k, _ *ItemEdicionCmd) { k.CostoUnitario = d("36000") })
	if ee, ok := errores.Como(err); !ok || ee.Codigo != "renglon_sin_lote" {
		t.Errorf("err = %v", err)
	}
	nadaEscrito(t, e)
}

func TestVencimientoVacioNoGeneraCambioFalso(t *testing.T) {
	e := nuevoB(t)
	var cero fecha.Fecha
	det, err := e.editar(func(_, j *ItemEdicionCmd) { j.Vencimiento = &cero })
	if err != nil || det.Version != 1 {
		t.Fatalf("err=%v version=%d", err, det.Version)
	}
	nadaEscrito(t, e)
}
