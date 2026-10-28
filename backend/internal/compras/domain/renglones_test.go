package domain

import (
	"encoding/json"
	"strings"
	"testing"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
)

func dd(s string) dinero.Dinero  { return dinero.MustParse(s) }
func pd(s string) *dinero.Dinero { x := dd(s); return &x }

// Compra de la historia: Kaiak 4 x 35.520 (vence 2027-09-30, precio 62.000) y Ekos 10 x 2.100.
func compraB() Compra {
	return Compra{ID: "c1", Fecha: base().Fecha, Version: 3, Total: dd("163080"), Items: []Item{
		{ID: "i1", ProductoID: "kaiak", Cantidad: 4, CostoUnitario: dd("35520"), PrecioVentaNuevo: pd("62000"), Vencimiento: f("2027-09-30")},
		{ID: "i2", ProductoID: "ekos", Cantidad: 10, CostoUnitario: dd("2100")},
	}}
}

func pedidoB(mod func(rs []RenglonEdicion)) CambiosCabecera {
	rs := []RenglonEdicion{
		{ID: "i1", ProductoID: "kaiak", Cantidad: 4, CostoUnitario: dd("35520"), PrecioVentaNuevo: pd("62000"), Vencimiento: f("2027-09-30")},
		{ID: "i2", ProductoID: "ekos", Cantidad: 10, CostoUnitario: dd("2100")},
	}
	if mod != nil {
		mod(rs)
	}
	c := CambiosCabecera{Version: 3, Dia: f("2026-10-20"), NombresProductos: map[string]string{"kaiak": "Kaiak Clásico", "ekos": "Ekos Jabón"}}
	c.Renglones = &rs
	return c
}

func TestRenglonesSinCambios(t *testing.T) {
	p, err := PlanificarEdicion(compraB(), pedidoB(nil), ahora)
	if err != nil || !p.SinCambios {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestRenglonesSubirCantidadYTotal(t *testing.T) {
	p, err := PlanificarEdicion(compraB(), pedidoB(func(rs []RenglonEdicion) { rs[0].Cantidad = 6 }), ahora)
	if err != nil || p.SinCambios || len(p.Modificados) != 1 {
		t.Fatalf("%+v %v", p, err)
	}
	m := p.Modificados[0]
	if m.Diferencia != 2 || m.CambioCosto || m.CambioVenc || m.CambioPrecio || m.Producto != "Kaiak Clásico" {
		t.Errorf("modificado = %+v", m)
	}
	if p.Compra.Total.String() != "234120.00" || p.Compra.Version != 4 {
		t.Errorf("compra = %+v", p.Compra)
	}
	if len(p.Cambios.Cabecera) != 0 || len(p.Cambios.Renglones) != 1 {
		t.Errorf("cambios = %+v", p.Cambios)
	}
}

func TestRenglonesAuditoriaGolden(t *testing.T) {
	p, err := PlanificarEdicion(compraB(), pedidoB(func(rs []RenglonEdicion) {
		rs[0].Cantidad = 6
		rs[0].CostoUnitario = dd("36000")
		rs[0].Vencimiento = nil
		rs[0].PrecioVentaNuevo = pd("65000")
	}), ahora)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(p.Cambios)
	want := `{"v":1,"cabecera":[],"renglones":[{"accion":"MODIFICADO","itemId":"i1","productoId":"kaiak","producto":"Kaiak Clásico","cambios":[` +
		`{"campo":"cantidad","antes":4,"despues":6},` +
		`{"campo":"costoUnitario","antes":"35520.00","despues":"36000.00"},` +
		`{"campo":"precioVentaNuevo","antes":"62000.00","despues":"65000.00"},` +
		`{"campo":"vencimiento","antes":"2027-09-30","despues":null}]}]}`
	if string(b) != want {
		t.Errorf("json =\n%s\nquería\n%s", b, want)
	}
	if _, err := LeerCambios(b); err != nil {
		t.Error(err)
	}
}

// El frontend manda vencimiento null para vaciarlo: el lote queda sin vencimiento.
func TestRenglonesVencimientoNullVaciaYVaciarPrecio(t *testing.T) {
	p, _ := PlanificarEdicion(compraB(), pedidoB(func(rs []RenglonEdicion) {
		rs[0].Vencimiento = nil
		rs[0].PrecioVentaNuevo = nil
	}), ahora)
	if len(p.Modificados) != 1 || !p.Modificados[0].CambioVenc || p.Modificados[0].Item.Vencimiento != nil ||
		!p.Modificados[0].CambioPrecio || p.Modificados[0].Item.PrecioVentaNuevo != nil {
		t.Fatalf("%+v", p.Modificados)
	}
	if p.Compra.Items[0].Vencimiento != nil {
		t.Error("la compra resultante debería quedar sin vencimiento")
	}
	// Sobre un renglón que ya no tenía vencimiento, null no es un cambio.
	p, _ = PlanificarEdicion(compraB(), pedidoB(func(rs []RenglonEdicion) { rs[1].Vencimiento = nil }), ahora)
	if !p.SinCambios {
		t.Error("null sobre un renglón sin vencimiento es sin cambios")
	}
}

func TestRenglonesValidaciones(t *testing.T) {
	c := pedidoB(func(rs []RenglonEdicion) {
		rs[0].Cantidad = 0
		rs[1].CostoUnitario = dd("-1")
		rs[1].PrecioVentaNuevo = pd("0")
	})
	cs := campos(ValidarEdicion(c, ahora))
	for _, k := range []string{"items[0].cantidad", "items[1].costoUnitario", "items[1].precioVentaNuevo"} {
		if cs[k] == "" {
			t.Errorf("falta %s: %v", k, cs)
		}
	}
	c = pedidoB(func(rs []RenglonEdicion) { rs[1].ID = "i1" })
	if campos(ValidarEdicion(c, ahora))["items[1].id"] == "" {
		t.Error("id repetido")
	}
	big := make([]RenglonEdicion, 301)
	for i := range big {
		big[i] = RenglonEdicion{ID: "x" + string(rune('a'+i%26)) + string(rune('a'+i/26)), ProductoID: "p", Cantidad: 1}
	}
	c = pedidoB(nil)
	c.Renglones = &big
	if campos(ValidarEdicion(c, ahora))["items"] == "" {
		t.Error("máximo 300")
	}
}

func TestRenglonesAjenosYProductoDistinto(t *testing.T) {
	_, err := PlanificarEdicion(compraB(), pedidoB(func(rs []RenglonEdicion) { rs[1].ID = "otro" }), ahora)
	if !strings.Contains(campos(err)["items[1].id"], "no pertenece") {
		t.Errorf("id ajeno: %v", err)
	}
	_, err = PlanificarEdicion(compraB(), pedidoB(func(rs []RenglonEdicion) { rs[0].ProductoID = "ekos" }), ahora)
	if !strings.Contains(campos(err)["items[0].productoId"], "quitalo y agregá otro") {
		t.Errorf("producto: %v", err)
	}
}

func lotes(s1, s2 int) map[string]LoteRenglon {
	return map[string]LoteRenglon{"i1": {LoteID: "l1", Stock: s1}, "i2": {LoteID: "l2", Stock: s2}}
}

func TestVerificarConsumo(t *testing.T) {
	bajar := func(a, b int) PlanEdicion {
		p, err := PlanificarEdicion(compraB(), pedidoB(func(rs []RenglonEdicion) {
			rs[0].Cantidad, rs[1].Cantidad = a, b
		}), ahora)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	// Se vendió 1 (stock 3): bajar 4 -> 3 se acepta.
	if err := VerificarConsumo(bajar(3, 10), lotes(3, 10)); err != nil {
		t.Errorf("sin consumo excedido: %v", err)
	}
	// Se consumieron 3 (stock 1): bajar a 3 (justo) se acepta, a 2 no.
	if err := VerificarConsumo(bajar(3, 10), lotes(1, 10)); err != nil {
		t.Errorf("justo hasta lo consumido: %v", err)
	}
	err := VerificarConsumo(bajar(2, 10), lotes(1, 10))
	e, ok := errores.Como(err)
	if !ok || e.Codigo != "compra_consumida" || e.Tipo != errores.TipoConflicto {
		t.Fatalf("err = %v", err)
	}
	if e.Campos["items[0].cantidad"] != "Kaiak Clásico: de las 4 unidades de esta compra ya salieron 3 (ventas o ajustes). Solo podés bajar hasta 3 unidades." {
		t.Errorf("mensaje = %q", e.Campos["items[0].cantidad"])
	}
	// Varios bloqueados en un único error (RN-B-02).
	err = VerificarConsumo(bajar(2, 1), lotes(1, 3))
	if e, _ := errores.Como(err); len(e.Campos) != 2 || !strings.Contains(e.Campos["items[1].cantidad"], "Ekos Jabón") ||
		!strings.Contains(e.Campos["items[1].cantidad"], "hasta 7 unidades") {
		t.Errorf("varios = %v", err)
	}
	// Subir nunca se bloquea, aunque el lote esté agotado (o negativo por ventas offline).
	if err := VerificarConsumo(bajar(8, 10), lotes(0, -2)); err != nil {
		t.Errorf("subir: %v", err)
	}
	// Venta anulada devuelve las unidades: stock 4 permite bajar a 1.
	if err := VerificarConsumo(bajar(1, 10), lotes(4, 10)); err != nil {
		t.Errorf("venta anulada: %v", err)
	}
	// Invariante: sin lote es un error tipado (409 renglon_sin_lote), no un 500.
	err = VerificarConsumo(bajar(2, 10), map[string]LoteRenglon{})
	if e, ok := errores.Como(err); !ok || e.Codigo != "renglon_sin_lote" {
		t.Errorf("sin lote: %v", err)
	}
}

// "vencimiento": "" llega como fecha cero: es "sin vencimiento" y no un cambio falso.
func TestRenglonesVencimientoVacioNoEsCambio(t *testing.T) {
	var cero fecha.Fecha
	c := pedidoB(func(rs []RenglonEdicion) { rs[1].Vencimiento = &cero })
	p, err := PlanificarEdicion(compraB(), c, ahora)
	if err != nil || !p.SinCambios {
		t.Fatalf("plan = %+v, %v", p, err)
	}
	// Sobre un renglón con vencimiento, "" lo vacía (y queda normalizado a nil).
	c = pedidoB(func(rs []RenglonEdicion) { rs[0].Vencimiento = &cero })
	p, _ = PlanificarEdicion(compraB(), c, ahora)
	if len(p.Modificados) != 1 || !p.Modificados[0].CambioVenc || p.Modificados[0].Item.Vencimiento != nil {
		t.Errorf("modificados = %+v", p.Modificados)
	}
}

func TestTopesDeCantidadYTotal(t *testing.T) {
	// Cantidad por encima del tope.
	c := pedidoB(func(rs []RenglonEdicion) { rs[0].Cantidad = MaxCantidadItem + 1 })
	if campos(ValidarEdicion(c, ahora))["items[0].cantidad"] == "" {
		t.Error("cantidad sobre el tope")
	}
	// En el tope exacto y con costo chico: pasa.
	c = pedidoB(func(rs []RenglonEdicion) { rs[0].Cantidad = MaxCantidadItem; rs[0].CostoUnitario = dd("1") })
	if err := ValidarEdicion(c, ahora); err != nil {
		t.Errorf("en el tope: %v", err)
	}
	// Costo que no entra en DECIMAL(12,2).
	c = pedidoB(func(rs []RenglonEdicion) { rs[0].CostoUnitario = dd("10000000000") })
	if campos(ValidarEdicion(c, ahora))["items[0].costoUnitario"] == "" {
		t.Error("costo sobre el tope")
	}
	c = pedidoB(func(rs []RenglonEdicion) { rs[0].PrecioVentaNuevo = pd("10000000000") })
	if campos(ValidarEdicion(c, ahora))["items[0].precioVentaNuevo"] == "" {
		t.Error("precio sobre el tope")
	}
	// Cada renglón entra, pero el total no (1.000.000 x $10.000 = $10.000.000.000).
	c = pedidoB(func(rs []RenglonEdicion) { rs[0].Cantidad = MaxCantidadItem; rs[0].CostoUnitario = dd("10000") })
	if campos(ValidarEdicion(c, ahora))["items"] == "" {
		t.Error("total sobre el tope")
	}
	// Mismo criterio en el alta.
	_, err := NuevaCompra("c", ahora, "", "", []Item{{ID: "i", ProductoID: "p", Cantidad: 3_000_000_000, CostoUnitario: dd("1000")}}, ahora)
	if campos(err)["items[0].cantidad"] == "" {
		t.Errorf("alta: %v", err)
	}
	_, err = NuevaCompra("c", ahora, "", "", []Item{
		{ID: "i", ProductoID: "p", Cantidad: MaxCantidadItem, CostoUnitario: dd("9000")},
		{ID: "j", ProductoID: "p", Cantidad: MaxCantidadItem, CostoUnitario: dd("9000")}}, ahora)
	if campos(err)["items"] == "" {
		t.Errorf("alta total: %v", err)
	}
	// El total máximo exacto entra.
	if _, err := NuevaCompra("c", ahora, "", "", []Item{{ID: "i", ProductoID: "p", Cantidad: 1, CostoUnitario: MaxMonto}}, ahora); err != nil {
		t.Errorf("total máximo: %v", err)
	}
}

func TestErrCompraConsumidaBaseNoSeContamina(t *testing.T) {
	p, _ := PlanificarEdicion(compraB(), pedidoB(func(rs []RenglonEdicion) { rs[0].Cantidad = 1 }), ahora)
	_ = VerificarConsumo(p, lotes(0, 10))
	if ErrCompraConsumida.Campos != nil {
		t.Error("el valor base no debe acumular campos entre pedidos")
	}
}
