package application

import (
	"testing"
	"time"

	"magui/internal/compras/domain"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
)

// US-001c: agregar y quitar renglones (casos de uso con fakes). Parte de la compra C1 de nuevoB:
// Kaiak 4 x 35.520 (precio 62.000) y Ekos Jabón 10 x 2.100, total 163.080.

func (e *entornoB) conChronos() {
	c := d("12000")
	e.cat.productos["chronos"] = ProductoCompra{Nombre: "Chronos Crema", Costo: &c, PrecioVenta: d("20000")}
}

func chronos() ItemEdicionCmd {
	p := d("24000")
	v, _ := fecha.Parse("2028-01-31")
	return ItemEdicionCmd{ProductoID: "chronos", Cantidad: 2, CostoUnitario: d("15000"), PrecioVentaNuevo: &p, Vencimiento: &v}
}

// editarLista edita con la lista completa que arma el test a partir de la actual.
func (e *entornoB) editarLista(armar func(k, j ItemEdicionCmd) []ItemEdicionCmd) (CompraDetalle, error) {
	l := *e.items(nil)
	items := armar(l[0], l[1])
	return e.s.EditarCompra(ctx, ComandoEditarCompra{ID: e.id, Version: e.version, Fecha: FechaEditada{Instante: &ahora},
		Items: &items, Origen: "WEB"})
}

func (e *entornoB) editarFecha(dia string, armar func(k, j ItemEdicionCmd) []ItemEdicionCmd) (CompraDetalle, error) {
	f, _ := fecha.Parse(dia)
	l := *e.items(nil)
	cmd := ComandoEditarCompra{ID: e.id, Version: e.version, Fecha: FechaEditada{Dia: &f}}
	if armar != nil {
		items := armar(l[0], l[1])
		cmd.Items = &items
	}
	return e.s.EditarCompra(ctx, cmd)
}

func TestAgregarRenglonCreaLoteYMovimientoCompra(t *testing.T) {
	e := nuevoB(t)
	e.conChronos()
	det, err := e.editarLista(func(k, j ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k, j, chronos()} })
	if err != nil {
		t.Fatal(err)
	}
	if det.Version != 2 || det.Total.String() != "193080.00" || det.Productos != 3 {
		t.Errorf("detalle = %+v", det.CompraResumen)
	}
	if len(e.repo.agregados) != 1 || len(e.inv.ingresos) != 3 { // 2 del alta + 1 nuevo
		t.Fatalf("agregados=%d ingresos=%d", len(e.repo.agregados), len(e.inv.ingresos))
	}
	in := e.inv.ingresos[2]
	if in.CompraID != e.id || in.CompraItemID != e.repo.agregados[0].ID || in.ProductoID != "chronos" || in.Cantidad != 2 ||
		in.Costo.String() != "15000.00" || in.Vencimiento.String() != "2028-01-31" || in.Nota != "Edición de compra del 25/10/2026" {
		t.Errorf("ingreso = %+v", in)
	}
	if len(e.inv.ajustes) != 0 {
		t.Errorf("no hay ajustes de lotes existentes: %+v", e.inv.ajustes)
	}
	// Es la única compra de Chronos: toma costo y precio.
	if len(e.cat.costos) != 1 || e.cat.costos[0].ProductoID != "chronos" || e.cat.costos[0].Costo.String() != "15000.00" ||
		e.cat.costos[0].PrecioVenta == nil || e.cat.costos[0].PrecioVenta.String() != "24000.00" {
		t.Errorf("costos = %+v", e.cat.costos)
	}
	r := e.repo.ediciones[0].Cambios.Renglones
	if len(r) != 1 || r[0].Accion != "AGREGADO" || r[0].Producto != "Chronos Crema" {
		t.Errorf("auditoría = %+v", r)
	}
}

func TestAgregarMismoProductoEnOtroRenglonEsLoteIndependiente(t *testing.T) {
	e := nuevoB(t)
	v, _ := fecha.Parse("2028-03-31")
	det, err := e.editarLista(func(k, j ItemEdicionCmd) []ItemEdicionCmd {
		return []ItemEdicionCmd{k, j, {ProductoID: "kaiak", Cantidad: 2, CostoUnitario: d("35520"), Vencimiento: &v}}
	})
	if err != nil || det.Productos != 3 {
		t.Fatalf("%v %+v", err, det.CompraResumen)
	}
	if len(e.inv.ingresos) != 3 || e.inv.ingresos[2].ProductoID != "kaiak" {
		t.Errorf("ingresos = %+v", e.inv.ingresos)
	}
}

func TestAgregarProductoInexistenteComboYNadaSeAplica(t *testing.T) {
	e := nuevoB(t)
	_, err := e.editarLista(func(k, j ItemEdicionCmd) []ItemEdicionCmd {
		k.Cantidad = 5 // cambio válido junto con el inválido
		return []ItemEdicionCmd{k, j, {ProductoID: "nope", Cantidad: 1, CostoUnitario: d("1")}}
	})
	if ee, ok := errores.Como(err); !ok || ee.Codigo != "producto_no_encontrado" || ee.Tipo != errores.TipoNoEncontrado {
		t.Fatalf("inexistente: %v", err)
	}
	nadaEscrito(t, e)
	_, err = e.editarLista(func(k, j ItemEdicionCmd) []ItemEdicionCmd {
		return []ItemEdicionCmd{k, j, {ProductoID: "combo", Cantidad: 1, CostoUnitario: d("1")}}
	})
	if ee, ok := errores.Como(err); !ok || ee.Codigo != "producto_no_comprable" || ee.Tipo != errores.TipoInvalido {
		t.Fatalf("combo: %v", err)
	}
	nadaEscrito(t, e)
	if len(e.repo.agregados) != 0 || len(e.inv.ingresos) != 2 {
		t.Error("no debería haberse creado ningún lote")
	}
}

func TestQuitarRenglonSinConsumo(t *testing.T) {
	e := nuevoB(t)
	det, err := e.editarLista(func(k, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k} })
	if err != nil {
		t.Fatal(err)
	}
	if det.Version != 2 || det.Total.String() != "142080.00" || det.Productos != 1 {
		t.Errorf("detalle = %+v", det.CompraResumen)
	}
	if len(e.repo.quitados) != 1 || e.repo.quitados[0] != e.jabon {
		t.Errorf("quitados = %v", e.repo.quitados)
	}
	if len(e.inv.ajustes) != 1 {
		t.Fatalf("ajustes = %+v", e.inv.ajustes)
	}
	a := e.inv.ajustes[0]
	if a.LoteID != "lote-"+e.jabon || a.Diferencia != -10 || !a.SoloCantidad || a.CompraID != e.id || a.Nota != "Edición de compra del 25/10/2026" {
		t.Errorf("ajuste = %+v", a)
	}
	r := e.repo.ediciones[0].Cambios.Renglones
	if len(r) != 1 || r[0].Accion != "QUITADO" || r[0].Producto != "Ekos Jabón" || r[0].Cambios[0].Antes != 10 || r[0].Cambios[0].Despues != nil {
		t.Errorf("auditoría = %+v", r)
	}
	// Ekos no tiene otra compra: su último costo se mantiene (no se emite nada).
	if len(e.cat.costos) != 0 {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

// P-A2: lo sumado por ajustes manuales queda en el lote; solo se descuenta lo que trajo el renglón.
func TestQuitarConAjustePositivoDescuentaSoloLaCantidadDelRenglon(t *testing.T) {
	e := nuevoB(t)
	e.consumir(e.jabon, 12) // 10 de la compra + 2 por ajuste manual
	if _, err := e.editarLista(func(k, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k} }); err != nil {
		t.Fatal(err)
	}
	if len(e.inv.ajustes) != 1 || e.inv.ajustes[0].Diferencia != -10 {
		t.Errorf("ajustes = %+v", e.inv.ajustes)
	}
}

func TestQuitarRenglonConsumidoRechazaTodoJunto(t *testing.T) {
	e := nuevoB(t)
	e.conChronos()
	e.consumir(e.jabon, 9)                                                                                     // salió 1
	e.consumir(e.kaiak, 1)                                                                                     // salieron 3
	_, err := e.editarLista(func(_, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{chronos()} }) // quita los dos, agrega uno válido
	ee, ok := errores.Como(err)
	if !ok || ee.Codigo != "compra_consumida" || ee.Tipo != errores.TipoConflicto || len(ee.Campos) != 2 {
		t.Fatalf("err = %v", err)
	}
	if ee.Campos["quitados["+e.jabon+"]"] != "Ekos Jabón: ya salió 1 de las 10 unidades; no se puede quitar. Podés bajar la cantidad hasta 1." ||
		ee.Campos["quitados["+e.kaiak+"]"] == "" {
		t.Errorf("campos = %v", ee.Campos)
	}
	// Todo o nada: ni el renglón nuevo, ni la baja, ni la versión.
	nadaEscrito(t, e)
	if len(e.repo.agregados) != 0 || len(e.repo.quitados) != 0 || len(e.inv.ingresos) != 2 || e.repo.compras[e.id].Version != 1 {
		t.Error("no debería haberse escrito nada")
	}
}

func TestNoSePuedeDejarLaCompraVacia(t *testing.T) {
	e := nuevoB(t)
	_, err := e.editarLista(func(_, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{} })
	if ee, ok := errores.Como(err); !ok || ee.Campos["items"] != "Agregá al menos un producto" {
		t.Errorf("err = %v", err)
	}
	nadaEscrito(t, e)
}

func TestQuitarRenglonYaQuitadoNoSePuedeReenviar(t *testing.T) {
	e := nuevoB(t)
	if _, err := e.editarLista(func(k, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k} }); err != nil {
		t.Fatal(err)
	}
	e.version = 2
	jabon := ItemEdicionCmd{ID: e.jabon, ProductoID: "jabon", Cantidad: 10, CostoUnitario: d("2100")}
	_, err := e.editarLista(func(k, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k, jabon} })
	if ee, _ := errores.Como(err); ee == nil || ee.Campos["items[1].id"] == "" {
		t.Errorf("err = %v", err)
	}
}

// Quitar la compra más reciente de un producto: el último costo pasa a la nueva más reciente y el precio no se toca.
func TestQuitarRecalculaElUltimoCostoConLaNuevaMasReciente(t *testing.T) {
	e := nuevoB(t)
	e.otraCompra("c0", ahora.AddDate(0, 0, -10), ahora.Add(-240*time.Hour)) // Kaiak a 30.000
	if _, err := e.editarLista(func(_, j ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{j} }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 1 {
		t.Fatalf("costos = %+v", e.cat.costos)
	}
	c := e.cat.costos[0]
	if c.ProductoID != "kaiak" || c.Costo.String() != "30000.00" || c.PrecioVenta != nil {
		t.Errorf("costo = %+v", c)
	}
}

func TestQuitarUnoDeDosRenglonesDelMismoProductoTomaElQueQueda(t *testing.T) {
	e := nuevoB(t)
	if _, err := e.editarLista(func(k, j ItemEdicionCmd) []ItemEdicionCmd {
		extra := ItemEdicionCmd{ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("40000")}
		return []ItemEdicionCmd{k, j, extra}
	}); err != nil {
		t.Fatal(err)
	}
	e.cat.costos = nil
	e.version = 2
	// Se quita el segundo renglón de Kaiak (el último): rige el costo del primero.
	det := e.repo.compras[e.id]
	nuevoID := det.Items[2].ID
	l := *e.items(nil)
	if _, err := e.s.EditarCompra(ctx, ComandoEditarCompra{ID: e.id, Version: 2, Fecha: FechaEditada{Instante: &ahora},
		Items: &[]ItemEdicionCmd{l[0], l[1]}}); err != nil {
		t.Fatal(err)
	}
	if len(e.repo.quitados) != 1 || e.repo.quitados[0] != nuevoID {
		t.Fatalf("quitados = %v", e.repo.quitados)
	}
	// El producto tenía costo 35.520 (alta) y el renglón quitado traía 40.000, pero el catálogo fake
	// conserva 35.520: ya coincide con el renglón que queda, no se emite nada.
	if len(e.cat.costos) != 0 {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

func TestCambiarFechaCambiaCualEsLaMasReciente(t *testing.T) {
	// C1 (25/10) es la más reciente de Kaiak; C0 (15/10) tiene Kaiak a 30.000. Se mueve C1 al 05/10.
	e := nuevoB(t)
	e.otraCompra("c0", ahora.AddDate(0, 0, -10), ahora.Add(-240*time.Hour))
	det, err := e.editarFecha("2026-10-05", nil)
	if err != nil || det.Version != 2 {
		t.Fatalf("%v %+v", err, det.CompraResumen)
	}
	if len(e.cat.costos) != 1 || e.cat.costos[0].ProductoID != "kaiak" || e.cat.costos[0].Costo.String() != "30000.00" || e.cat.costos[0].PrecioVenta != nil {
		t.Errorf("costos = %+v", e.cat.costos)
	}
	if len(e.inv.ajustes) != 0 || len(e.repo.ediciones) != 1 {
		t.Errorf("solo cambia la fecha: ajustes=%+v ediciones=%d", e.inv.ajustes, len(e.repo.ediciones))
	}
}

func TestCambiarFechaALaMasReciente(t *testing.T) {
	// C1 (05/10) vieja; C0 (15/10) es la más reciente. Se mueve C1 al 20/10: pasa a serlo.
	e := nuevoB(t)
	c := e.repo.compras[e.id]
	c.Fecha = time.Date(2026, 10, 5, 15, 0, 0, 0, time.UTC)
	e.repo.compras[e.id] = c
	e.otraCompra("c0", time.Date(2026, 10, 15, 15, 0, 0, 0, time.UTC), ahora.Add(-240*time.Hour))
	if _, err := e.editarFecha("2026-10-20", nil); err != nil {
		t.Fatal(err)
	}
	// Kaiak: el costo vigente del catálogo ya es 35.520 (igual al renglón): no hay cambio. Ekos tampoco.
	if len(e.cat.costos) != 0 {
		t.Errorf("costos iguales: no se emite nada: %+v", e.cat.costos)
	}
	// Si el catálogo tenía el costo de C0 (30.000), pasa a 35.520.
	e = nuevoB(t)
	c = e.repo.compras[e.id]
	c.Fecha = time.Date(2026, 10, 5, 15, 0, 0, 0, time.UTC)
	e.repo.compras[e.id] = c
	e.otraCompra("c0", time.Date(2026, 10, 15, 15, 0, 0, 0, time.UTC), ahora.Add(-240*time.Hour))
	k := e.cat.productos["kaiak"]
	costo0 := d("30000")
	k.Costo = &costo0
	e.cat.productos["kaiak"] = k
	if _, err := e.editarFecha("2026-10-20", nil); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 1 || e.cat.costos[0].ProductoID != "kaiak" || e.cat.costos[0].Costo.String() != "35520.00" || e.cat.costos[0].PrecioVenta != nil {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

func TestCambiarFechaSinOtraCompraMantieneElCosto(t *testing.T) {
	e := nuevoB(t)
	if _, err := e.editarFecha("2026-10-05", nil); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 0 {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

func TestQuitarNuncaRevierteElPrecio(t *testing.T) {
	e := nuevoB(t)
	e.otraCompra("c0", ahora.AddDate(0, 0, -10), ahora.Add(-240*time.Hour))
	if _, err := e.editarLista(func(_, j ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{j} }); err != nil {
		t.Fatal(err)
	}
	for _, c := range e.cat.costos {
		if c.PrecioVenta != nil {
			t.Errorf("el precio no se revierte: %+v", c)
		}
	}
}

func TestAgregarYQuitarCombinadosConCostoYPrecio(t *testing.T) {
	e := nuevoB(t)
	e.conChronos()
	_, err := e.editarLista(func(k, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k, chronos()} })
	if err != nil {
		t.Fatal(err)
	}
	r := e.repo.ediciones[0].Cambios.Renglones
	if len(r) != 2 {
		t.Fatalf("renglones = %+v", r)
	}
	acciones := map[string]string{}
	for _, x := range r {
		acciones[x.Accion] = x.Producto
	}
	if acciones["AGREGADO"] != "Chronos Crema" || acciones["QUITADO"] != "Ekos Jabón" {
		t.Errorf("acciones = %v", acciones)
	}
	if e.repo.compras[e.id].Total.String() != "172080.00" {
		t.Errorf("total = %s", e.repo.compras[e.id].Total)
	}
}

func TestQuitarProductoInactivoSeAcepta(t *testing.T) {
	// El caso de uso no consulta el estado del producto (RN-EC-14): quitar un renglón de un producto inactivo pasa.
	e := nuevoB(t)
	if _, err := e.editarLista(func(k, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k} }); err != nil {
		t.Fatal(err)
	}
}

func TestQuitarFallaEnInventarioNoDejaRastroEnLaAuditoria(t *testing.T) {
	e := nuevoB(t)
	e.inv.falla = errorFalso("lote bloqueado")
	_, err := e.editarLista(func(k, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k} })
	if err == nil || err.Error() != "lote bloqueado" || len(e.repo.ediciones) != 0 {
		t.Errorf("err=%v ediciones=%d", err, len(e.repo.ediciones))
	}
}

type errorFalso string

func (e errorFalso) Error() string { return string(e) }

var _ = domain.MaxItems

// I-1: cambiar solo la fecha de la compra que era y sigue siendo la más reciente no pisa un costo editado a mano.
func TestCambiarSoloFechaNoPisaCostoDeCatalogoSiSigueSiendoLaMasReciente(t *testing.T) {
	e := nuevoB(t)
	k := e.cat.productos["kaiak"]
	manual := d("40000") // distinto al 35.520 del renglón
	k.Costo = &manual
	e.cat.productos["kaiak"] = k
	e.otraCompra("c0", ahora.AddDate(0, 0, -30), ahora.Add(-720*time.Hour))
	if _, err := e.editarFecha("2026-10-20", nil); err != nil { // sigue siendo más reciente que c0
		t.Fatal(err)
	}
	if len(e.cat.costos) != 0 {
		t.Errorf("no debería tocar el catálogo: %+v", e.cat.costos)
	}
}

func TestAgregarProductoInactivoSeRechazaPeroElExistenteInactivoSeEdita(t *testing.T) {
	e := nuevoB(t)
	e.conChronos()
	pc := e.cat.productos["chronos"]
	pc.Inactivo = true
	e.cat.productos["chronos"] = pc
	_, err := e.editarLista(func(k, j ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k, j, chronos()} })
	ee, ok := errores.Como(err)
	if !ok || ee.Codigo != "producto_inactivo" || ee.Tipo != errores.TipoInvalido || ee.Campos["items[2].productoId"] == "" {
		t.Fatalf("inactivo nuevo: %v", err)
	}
	nadaEscrito(t, e)
	// Un renglón existente cuyo producto se desactivó después se puede editar.
	k := e.cat.productos["kaiak"]
	k.Inactivo = true
	e.cat.productos["kaiak"] = k
	if _, err := e.editarLista(func(k, j ItemEdicionCmd) []ItemEdicionCmd { k.Cantidad = 5; return []ItemEdicionCmd{k, j} }); err != nil {
		t.Fatalf("editar existente inactivo: %v", err)
	}
}
