package application

import (
	"errors"
	"strings"
	"testing"

	"magui/internal/compras/domain"
	"magui/internal/kernel/errores"
)

// QA US-001b: casos borde adicionales sobre el entorno de la historia (compra C1, Kaiak 4 y Jabón 10).

// Venta anulada: las unidades vuelven al lote (stock 4), así que se puede bajar de 4 a 1 con un -3.
func TestQAVentaAnuladaDevuelveLasUnidades(t *testing.T) {
	e := nuevoB(t)
	e.consumir(e.kaiak, 4)
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 1 }); err != nil {
		t.Fatal(err)
	}
	if len(e.inv.ajustes) != 1 || e.inv.ajustes[0].Diferencia != -3 {
		t.Errorf("ajustes = %+v", e.inv.ajustes)
	}
}

// Consumo por ajuste y combo (2 + 1 = 3 unidades): 4 -> 2 se rechaza diciendo que salieron 3;
// el mensaje trae el mínimo permitido (RN-EC-07).
func TestQAMensajeDeConsumoIndicaCuantasSalieron(t *testing.T) {
	e := nuevoB(t)
	e.consumir(e.kaiak, 1)
	_, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 2 })
	ee, ok := errores.Como(err)
	if !ok || ee.Codigo != "compra_consumida" {
		t.Fatalf("err = %v", err)
	}
	m := ee.Campos["items[0].cantidad"]
	for _, quiero := range []string{"Kaiak Clásico", "las 4 unidades", "salieron 3", "hasta 3 unidades"} {
		if !strings.Contains(m, quiero) {
			t.Errorf("el mensaje %q debería contener %q", m, quiero)
		}
	}
	// Todo el lote consumido: el mínimo permitido es 4.
	e = nuevoB(t)
	e.consumir(e.kaiak, 0)
	_, err = e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 3 })
	if ee, ok := errores.Como(err); !ok || !strings.Contains(ee.Campos["items[0].cantidad"], "hasta 4 unidades") {
		t.Errorf("err = %v", err)
	}
}

// Lote totalmente consumido: subir la cantidad y cambiar el costo siguen permitidos.
func TestQALoteConsumidoPermiteSubirYCambiarCosto(t *testing.T) {
	e := nuevoB(t)
	e.consumir(e.kaiak, 0)
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 6; k.CostoUnitario = d("36000") }); err != nil {
		t.Fatalf("subir con el lote consumido debe pasar: %v", err)
	}
	if len(e.inv.ajustes) != 1 || e.inv.ajustes[0].Diferencia != 2 {
		t.Errorf("ajustes = %+v", e.inv.ajustes)
	}
}

// Compra en versión 3, cantidad y costo a la vez: total recalculado, versión 4 y auditoría por renglón.
func TestQATotalVersionYAuditoriaDeCantidadYCosto(t *testing.T) {
	e := nuevoB(t)
	c := e.repo.compras[e.id]
	c.Version = 3
	e.repo.compras[e.id] = c
	e.version = 3
	det, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 6; k.CostoUnitario = d("36000") })
	if err != nil {
		t.Fatal(err)
	}
	// 6 x 36.000 + 10 x 2.100
	if det.Version != 4 || det.Total.String() != "237000.00" {
		t.Errorf("version=%d total=%s", det.Version, det.Total)
	}
	if len(e.repo.ediciones) != 1 || e.repo.ediciones[0].Version != 4 {
		t.Fatalf("ediciones = %+v", e.repo.ediciones)
	}
	rs := e.repo.ediciones[0].Cambios.Renglones
	if len(rs) != 1 {
		t.Fatalf("solo el renglón modificado va a la auditoría: %+v", rs)
	}
	got := map[string][2]any{}
	for _, c := range rs[0].Cambios {
		got[c.Campo] = [2]any{c.Antes, c.Despues}
	}
	if rs[0].Accion != "MODIFICADO" || got["cantidad"] != [2]any{4, 6} ||
		got["costoUnitario"] != [2]any{"35520.00", "36000.00"} || len(got) != 2 {
		t.Errorf("auditoría = %+v", rs[0])
	}
}

// Solo cambia el precio de reventa: no hay movimiento de stock ni cambio de lote y, siendo la compra
// más reciente, el precio del producto se actualiza.
func TestQASoloPrecioNoMueveStock(t *testing.T) {
	e := nuevoB(t)
	p := d("65000")
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.PrecioVentaNuevo = &p }); err != nil {
		t.Fatal(err)
	}
	if len(e.inv.ajustes) != 0 {
		t.Errorf("no debería haber ajuste de lote: %+v", e.inv.ajustes)
	}
	if len(e.cat.costos) != 1 || e.cat.costos[0].PrecioVenta == nil || e.cat.costos[0].PrecioVenta.String() != "65000.00" {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

// A guarda y después B guarda con la versión vieja: 409 y nada nuevo escrito.
func TestQADosPersonasEditanALaVez(t *testing.T) {
	e := nuevoB(t)
	if _, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 5 }); err != nil {
		t.Fatal(err)
	}
	ajustes, ediciones := len(e.inv.ajustes), len(e.repo.ediciones)
	_, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 8 }) // B sigue con version 1
	if !errors.Is(err, domain.ErrVersionDesactualizada) {
		t.Fatalf("err = %v", err)
	}
	if len(e.inv.ajustes) != ajustes || len(e.repo.ediciones) != ediciones {
		t.Error("B no debería haber escrito nada")
	}
}

// Cabecera y renglones juntos: una sola versión y una sola fila de auditoría.
func TestQACabeceraYRenglonesEnUnaSolaEdicion(t *testing.T) {
	e := nuevoB(t)
	det, err := e.s.EditarCompra(ctx, ComandoEditarCompra{ID: e.id, Version: 1, Fecha: FechaEditada{Instante: &ahora},
		Notas: "corregida", Items: e.items(func(k, _ *ItemEdicionCmd) { k.Cantidad = 5 })})
	if err != nil {
		t.Fatal(err)
	}
	if det.Version != 2 || len(e.repo.ediciones) != 1 {
		t.Fatalf("version=%d ediciones=%d", det.Version, len(e.repo.ediciones))
	}
	c := e.repo.ediciones[0].Cambios
	if len(c.Cabecera) != 1 || c.Cabecera[0].Campo != "notas" || len(c.Renglones) != 1 {
		t.Errorf("cambios = %+v", c)
	}
}

// BUG-001 (QA US-001b): no hay tope de cantidad ni de total. El total se guarda en DECIMAL(12,2)
// (máx. 9.999.999.999,99) y la cantidad en INT: valores mayores pasan la validación de dominio y recién
// fallarían en la base (500 en lugar de 422). Este test documenta el comportamiento sin fallar la suite.
func TestQASinTopeDeCantidadNiDeTotal(t *testing.T) {
	// BUG-001 corregido: cantidad y total acotados, con 422 por campo (antes daba 500 al guardar).
	e := nuevoB(t)
	_, err := e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = 3_000_000_000; k.CostoUnitario = d("1000") })
	ee, ok := errores.Como(err)
	if !ok || ee.Tipo != errores.TipoInvalido || ee.Campos["items[0].cantidad"] == "" {
		t.Fatalf("debería ser 422 en items[0].cantidad: %v", err)
	}
	nadaEscrito(t, e)
	// Total que no entra en DECIMAL(12,2).
	_, err = e.editar(func(k, _ *ItemEdicionCmd) { k.Cantidad = domain.MaxCantidadItem; k.CostoUnitario = d("90000") })
	if ee, ok := errores.Como(err); !ok || ee.Tipo != errores.TipoInvalido || ee.Campos["items"] == "" {
		t.Fatalf("total: %v", err)
	}
	nadaEscrito(t, e)
}
