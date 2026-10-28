package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"magui/internal/kernel/errores"
)

// US-001c: agregar y quitar renglones (dominio puro).

func conIDs(c CambiosCabecera) CambiosCabecera {
	n := 0
	c.NuevoID = func() string { n++; return fmt.Sprintf("n%d", n) }
	c.NombresProductos["chronos"] = "Chronos Crema"
	return c
}

func soloKaiak(c CambiosCabecera) CambiosCabecera {
	rs := (*c.Renglones)[:1]
	c.Renglones = &rs
	return c
}

func TestAgregarRenglon(t *testing.T) {
	c := conIDs(pedidoB(nil))
	rs := append(*c.Renglones, RenglonEdicion{ProductoID: "chronos", Cantidad: 2, CostoUnitario: dd("15000"),
		PrecioVentaNuevo: pd("24000"), Vencimiento: f("2028-01-31")})
	c.Renglones = &rs
	p, err := PlanificarEdicion(compraB(), c, ahora)
	if err != nil || p.SinCambios {
		t.Fatalf("%+v %v", p, err)
	}
	if len(p.Agregados) != 1 || p.Agregados[0].Item.ID != "n1" || p.Agregados[0].Producto != "Chronos Crema" || len(p.Modificados) != 0 || len(p.Quitados) != 0 {
		t.Fatalf("plan = %+v", p)
	}
	if p.Compra.Total.String() != "193080.00" || p.Compra.Version != 4 || len(p.Compra.Items) != 3 {
		t.Errorf("compra = %+v", p.Compra)
	}
	b, _ := json.Marshal(p.Cambios)
	want := `{"v":1,"cabecera":[],"renglones":[{"accion":"AGREGADO","itemId":"n1","productoId":"chronos","producto":"Chronos Crema","cambios":[` +
		`{"campo":"cantidad","antes":null,"despues":2},` +
		`{"campo":"costoUnitario","antes":null,"despues":"15000.00"},` +
		`{"campo":"precioVentaNuevo","antes":null,"despues":"24000.00"},` +
		`{"campo":"vencimiento","antes":null,"despues":"2028-01-31"}]}]}`
	if string(b) != want {
		t.Errorf("json =\n%s\nquería\n%s", b, want)
	}
}

func TestAgregarMismoProductoEnOtroRenglon(t *testing.T) {
	c := conIDs(pedidoB(nil))
	rs := append(*c.Renglones, RenglonEdicion{ProductoID: "kaiak", Cantidad: 2, CostoUnitario: dd("35520"), Vencimiento: f("2028-03-31")})
	c.Renglones = &rs
	p, err := PlanificarEdicion(compraB(), c, ahora)
	if err != nil || len(p.Agregados) != 1 || len(p.Compra.Items) != 3 {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestQuitarRenglon(t *testing.T) {
	p, err := PlanificarEdicion(compraB(), conIDs(soloKaiak(pedidoB(nil))), ahora)
	if err != nil || p.SinCambios || len(p.Quitados) != 1 || p.Quitados[0].Item.ID != "i2" {
		t.Fatalf("%+v %v", p, err)
	}
	if p.Compra.Total.String() != "142080.00" || len(p.Compra.Items) != 1 || p.Compra.Items[0].ID != "i1" {
		t.Errorf("compra = %+v", p.Compra)
	}
	b, _ := json.Marshal(p.Cambios)
	want := `{"v":1,"cabecera":[],"renglones":[{"accion":"QUITADO","itemId":"i2","productoId":"ekos","producto":"Ekos Jabón","cambios":[` +
		`{"campo":"cantidad","antes":10,"despues":null},{"campo":"costoUnitario","antes":"2100.00","despues":null}]}]}`
	if string(b) != want {
		t.Errorf("json =\n%s\nquería\n%s", b, want)
	}
	if got := p.ProductosAfectados(compraB().Items, false); len(got) != 1 || got[0] != "ekos" {
		t.Errorf("afectados = %v", got)
	}
}

func TestQuitarYAgregarEnLaMismaEdicionOrdenaPorId(t *testing.T) {
	c := conIDs(soloKaiak(pedidoB(nil)))
	rs := append(*c.Renglones, RenglonEdicion{ProductoID: "chronos", Cantidad: 2, CostoUnitario: dd("15000")})
	c.Renglones = &rs
	p, err := PlanificarEdicion(compraB(), c, ahora)
	if err != nil {
		t.Fatal(err)
	}
	r := p.Cambios.Renglones
	if len(r) != 2 || r[0].Accion != "QUITADO" || r[1].Accion != "AGREGADO" || r[1].Cambios[0].Despues != 2 {
		t.Errorf("renglones = %+v", r)
	}
	if p.Compra.Total.String() != "172080.00" { // 142.080 + 30.000
		t.Errorf("total = %s", p.Compra.Total)
	}
	got := p.ProductosAfectados(compraB().Items, false)
	if len(got) != 2 || got[0] != "chronos" || got[1] != "ekos" {
		t.Errorf("afectados = %v", got)
	}
}

func TestNoQuedaCompraVaciaNiMasDe300(t *testing.T) {
	c := pedidoB(nil)
	vacio := []RenglonEdicion{}
	c.Renglones = &vacio
	if campos(ValidarEdicion(c, ahora))["items"] != "Agregá al menos un producto" {
		t.Error("quitar todos debe dar 422 items")
	}
	// 300 sin problema, 301 con renglón nuevo no.
	c = pedidoB(nil)
	rs := make([]RenglonEdicion, 301)
	for i := range rs {
		rs[i] = RenglonEdicion{ProductoID: "kaiak", Cantidad: 1, CostoUnitario: dd("1")}
	}
	c.Renglones = &rs
	if campos(ValidarEdicion(c, ahora))["items"] != "Máximo 300 renglones por compra" {
		t.Error("301 debe dar 422")
	}
	rs = rs[:300]
	if err := ValidarEdicion(c, ahora); err != nil {
		t.Errorf("300: %v", err)
	}
}

func TestRenglonQuitadoNoSePuedeReenviar(t *testing.T) {
	// El caso de uso solo le pasa a la compra los renglones vigentes: un id quitado no pertenece.
	actual := compraB()
	actual.Items = actual.Items[:1]
	c := pedidoB(nil) // reenvía i2
	_, err := PlanificarEdicion(actual, c, ahora)
	if !strings.Contains(campos(err)["items[1].id"], "no pertenece") {
		t.Errorf("err = %v", err)
	}
}

func TestVerificarConsumoQuitar(t *testing.T) {
	p, err := PlanificarEdicion(compraB(), conIDs(soloKaiak(pedidoB(nil))), ahora)
	if err != nil {
		t.Fatal(err)
	}
	// Sin consumo (stock del lote = cantidad) o con stock de más por un ajuste positivo: se acepta.
	if err := VerificarConsumo(p, lotes(4, 10)); err != nil {
		t.Errorf("sin consumo: %v", err)
	}
	if err := VerificarConsumo(p, lotes(4, 12)); err != nil {
		t.Errorf("con ajuste positivo: %v", err)
	}
	// Salió 1 de 10: 409 con la clave quitados[i2] y el mensaje de la historia.
	err = VerificarConsumo(p, lotes(4, 9))
	e, ok := errores.Como(err)
	if !ok || e.Codigo != "compra_consumida" || e.Tipo != errores.TipoConflicto {
		t.Fatalf("err = %v", err)
	}
	if e.Campos["quitados[i2]"] != "Ekos Jabón: ya salió 1 de las 10 unidades; no se puede quitar. Podés bajar la cantidad hasta 1." {
		t.Errorf("mensaje = %q", e.Campos["quitados[i2]"])
	}
	// Varios rechazados juntos (modificado + quitado) en un solo error.
	c := conIDs(soloKaiak(pedidoB(func(rs []RenglonEdicion) { rs[0].Cantidad = 2 })))
	p, _ = PlanificarEdicion(compraB(), c, ahora)
	err = VerificarConsumo(p, lotes(1, 3))
	e, _ = errores.Como(err)
	if len(e.Campos) != 2 || e.Campos["items[0].cantidad"] == "" || !strings.Contains(e.Campos["quitados[i2]"], "ya salieron 7 de las 10") {
		t.Errorf("campos = %v", e.Campos)
	}
	// Sin lote: error tipado.
	if e, ok := errores.Como(VerificarConsumo(p, map[string]LoteRenglon{"i1": {LoteID: "l1", Stock: 4}})); !ok || e.Codigo != "renglon_sin_lote" {
		t.Errorf("sin lote: %v", e)
	}
}

func TestProductosAfectadosPorCambioDeFecha(t *testing.T) {
	c := pedidoB(func(rs []RenglonEdicion) { rs[0].Cantidad = 5 }) // solo cantidad: no afecta por sí sola
	p, _ := PlanificarEdicion(compraB(), c, ahora)
	if got := p.ProductosAfectados(compraB().Items, false); len(got) != 0 {
		t.Errorf("solo cantidad: %v", got)
	}
	if got := p.ProductosAfectados(compraB().Items, true); len(got) != 2 {
		t.Errorf("con cambio de fecha: %v", got)
	}
}
