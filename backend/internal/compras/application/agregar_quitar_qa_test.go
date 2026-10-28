package application

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"magui/internal/compras/domain"
	"magui/internal/kernel/errores"
)

// QA de US-001c (casos borde adicionales). Parte de la compra C1 de nuevoB: Kaiak 4 x 35.520 y Ekos Jabón 10 x 2.100.

// Quitar el último de dos renglones de Kaiak: el costo vuelve al del renglón que queda, y el catálogo
// (que tenía el del renglón quitado) lo recibe. El test del desarrollador no lo ejercita porque su
// catálogo ya coincidía con el renglón que queda.
func TestQAQuitarSegundoRenglonDelMismoProductoVuelveAlCostoDelQueQueda(t *testing.T) {
	e := nuevoB(t)
	if _, err := e.editarLista(func(k, j ItemEdicionCmd) []ItemEdicionCmd {
		return []ItemEdicionCmd{k, j, {ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("40000")}}
	}); err != nil {
		t.Fatal(err)
	}
	e.version = 2
	k := e.cat.productos["kaiak"]
	c40 := d("40000")
	k.Costo = &c40 // como quedaría el catálogo real tras la edición anterior
	e.cat.productos["kaiak"] = k
	e.cat.costos = nil
	l := *e.items(nil)
	if _, err := e.s.EditarCompra(ctx, ComandoEditarCompra{ID: e.id, Version: 2, Fecha: FechaEditada{Instante: &ahora},
		Items: &[]ItemEdicionCmd{l[0], l[1]}}); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 1 || e.cat.costos[0].ProductoID != "kaiak" || e.cat.costos[0].Costo.String() != "35520.00" || e.cat.costos[0].PrecioVenta != nil {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

// Quitar y agregar el mismo producto en una edición: baja el lote viejo, nace uno nuevo y el costo es el del nuevo.
func TestQAQuitarYAgregarMismoProductoEnUnaEdicion(t *testing.T) {
	e := nuevoB(t)
	det, err := e.editarLista(func(k, _ ItemEdicionCmd) []ItemEdicionCmd {
		return []ItemEdicionCmd{k, {ProductoID: "jabon", Cantidad: 3, CostoUnitario: d("2500")}}
	})
	if err != nil {
		t.Fatal(err)
	}
	if det.Total.String() != "149580.00" || det.Productos != 2 {
		t.Errorf("detalle = %+v", det.CompraResumen)
	}
	if len(e.inv.ajustes) != 1 || e.inv.ajustes[0].Diferencia != -10 || !e.inv.ajustes[0].SoloCantidad {
		t.Errorf("ajustes = %+v", e.inv.ajustes)
	}
	if len(e.inv.ingresos) != 3 || e.inv.ingresos[2].Cantidad != 3 {
		t.Errorf("ingresos = %+v", e.inv.ingresos)
	}
	if len(e.cat.costos) != 1 || e.cat.costos[0].ProductoID != "jabon" || e.cat.costos[0].Costo.String() != "2500.00" || e.cat.costos[0].PrecioVenta != nil {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

// Quitar los dos renglones originales con uno nuevo en la misma edición es válido (la compra no queda vacía).
func TestQAQuitarTodosLosOriginalesAgregandoUnoNuevo(t *testing.T) {
	e := nuevoB(t)
	e.conChronos()
	det, err := e.editarLista(func(_, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{chronos()} })
	if err != nil {
		t.Fatal(err)
	}
	if det.Total.String() != "30000.00" || det.Productos != 1 || len(e.repo.quitados) != 2 {
		t.Errorf("detalle = %+v quitados=%v", det.CompraResumen, e.repo.quitados)
	}
}

// Cambio de fecha y quitar en la misma edición: Kaiak vuelve a la compra anterior por la fecha; Ekos queda igual.
func TestQACambioDeFechaMasQuitarEnLaMismaEdicion(t *testing.T) {
	e := nuevoB(t)
	e.otraCompra("c0", ahora.AddDate(0, 0, -10), ahora.Add(-240*time.Hour)) // Kaiak a 30.000
	if _, err := e.editarFecha("2026-10-05", func(k, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k} }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 1 || e.cat.costos[0].ProductoID != "kaiak" || e.cat.costos[0].Costo.String() != "30000.00" || e.cat.costos[0].PrecioVenta != nil {
		t.Errorf("costos = %+v", e.cat.costos)
	}
	if len(e.inv.ajustes) != 1 || e.inv.ajustes[0].Diferencia != -10 {
		t.Errorf("ajustes = %+v", e.inv.ajustes)
	}
}

// Quitar el renglón de una compra que no es la más reciente no toca el catálogo.
func TestQAQuitarEnCompraQueNoEsLaMasRecienteNoTocaElCosto(t *testing.T) {
	e := nuevoB(t)
	e.otraCompra("c0", ahora, ahora.Add(time.Hour)) // mismo día, creada después: es la más reciente
	if _, err := e.editarLista(func(_, j ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{j} }); err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 0 {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

// Dos renglones nuevos del mismo producto: manda el último (costo y precio), como en el alta.
func TestQAAgregarDosRenglonesNuevosDelMismoProductoGanaElUltimo(t *testing.T) {
	e := nuevoB(t)
	e.conChronos()
	p1, p2 := d("24000"), d("26000")
	_, err := e.editarLista(func(k, j ItemEdicionCmd) []ItemEdicionCmd {
		return []ItemEdicionCmd{k, j,
			{ProductoID: "chronos", Cantidad: 1, CostoUnitario: d("15000"), PrecioVentaNuevo: &p1},
			{ProductoID: "chronos", Cantidad: 1, CostoUnitario: d("16000"), PrecioVentaNuevo: &p2}}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(e.cat.costos) != 1 || e.cat.costos[0].Costo.String() != "16000.00" || e.cat.costos[0].PrecioVenta == nil || e.cat.costos[0].PrecioVenta.String() != "26000.00" {
		t.Errorf("costos = %+v", e.cat.costos)
	}
}

// Importes con decimales y cantidades varias: el total es exacto.
func TestQATotalConDecimalesAlAgregar(t *testing.T) {
	e := nuevoB(t)
	e.conChronos()
	det, err := e.editarLista(func(k, j ItemEdicionCmd) []ItemEdicionCmd {
		return []ItemEdicionCmd{k, j, {ProductoID: "chronos", Cantidad: 3, CostoUnitario: d("1234.56")}}
	})
	if err != nil || det.Total.String() != "166783.68" {
		t.Errorf("err=%v total=%s", err, det.Total)
	}
}

// Versión vieja con agregar y quitar: 409 version_desactualizada y no se escribe nada.
func TestQAVersionVencidaConAgregarYQuitarNoEscribe(t *testing.T) {
	e := nuevoB(t)
	e.conChronos()
	e.version = 5
	_, err := e.editarLista(func(k, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k, chronos()} })
	if ee, ok := errores.Como(err); !ok || ee.Codigo != "version_desactualizada" || ee.Tipo != errores.TipoConflicto {
		t.Fatalf("err = %v", err)
	}
	nadaEscrito(t, e)
	if len(e.repo.agregados) != 0 || len(e.repo.quitados) != 0 || len(e.inv.ingresos) != 2 {
		t.Error("no debería haberse escrito nada")
	}
}

// Una sola unidad ya consumida: el mensaje no sugiere "bajar hasta" (variante de 1 unidad).
func TestQAQuitarRenglonDeUnaUnidadConsumida(t *testing.T) {
	e := nuevoB(t)
	l := *e.items(nil)
	c := e.repo.compras[e.id]
	c.Items = append([]domain.Item(nil), c.Items...)
	c.Items[0].Cantidad = 1
	e.repo.compras[e.id] = c
	e.consumir(e.kaiak, 0)
	_, err := e.s.EditarCompra(ctx, ComandoEditarCompra{ID: e.id, Version: 1, Fecha: FechaEditada{Instante: &ahora},
		Items: &[]ItemEdicionCmd{l[1]}})
	ee, ok := errores.Como(err)
	if !ok || ee.Codigo != "compra_consumida" || !strings.Contains(ee.Campos["quitados["+e.kaiak+"]"], "ya salió la única unidad") {
		t.Fatalf("err = %v", err)
	}
}

// Tope de 300: con 300 renglones, agregar uno más es 422; quitar uno y agregar otro (300) se acepta.
func TestQATope300RenglonesExistentes(t *testing.T) {
	s, repo, cat, inv := entorno()
	cat.productos["kaiak"] = ProductoCompra{Nombre: "Kaiak", PrecioVenta: d("1")}
	inv.lotes = map[string]domain.LoteRenglon{}
	items := make([]domain.Item, 300)
	cmd := make([]ItemEdicionCmd, 300)
	for i := range items {
		id := fmt.Sprintf("r%03d", i)
		items[i] = domain.Item{ID: id, ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("10")}
		cmd[i] = ItemEdicionCmd{ID: id, ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("10")}
		inv.lotes[id] = domain.LoteRenglon{LoteID: "l" + id, Stock: 1}
	}
	repo.compras["c300"] = domain.Compra{ID: "c300", Fecha: ahora, CreatedAt: ahora, Version: 1, Items: items, Total: d("3000")}
	nuevo := ItemEdicionCmd{ProductoID: "kaiak", Cantidad: 1, CostoUnitario: d("10")}

	mas := append(append([]ItemEdicionCmd(nil), cmd...), nuevo)
	_, err := s.EditarCompra(ctx, ComandoEditarCompra{ID: "c300", Version: 1, Fecha: FechaEditada{Instante: &ahora}, Items: &mas})
	if ee, ok := errores.Como(err); !ok || ee.Campos["items"] != "Máximo 300 renglones por compra" {
		t.Fatalf("301: %v", err)
	}
	if len(repo.agregados) != 0 || len(repo.ediciones) != 0 {
		t.Error("no debería haberse escrito nada")
	}
	igual := append(append([]ItemEdicionCmd(nil), cmd[1:]...), nuevo)
	det, err := s.EditarCompra(ctx, ComandoEditarCompra{ID: "c300", Version: 1, Fecha: FechaEditada{Instante: &ahora}, Items: &igual})
	if err != nil || det.Productos != 300 {
		t.Fatalf("quitar 1 y agregar 1: err=%v productos=%d", err, det.Productos)
	}
}

// El documento de auditoría de un QUITADO solo trae cantidad y costo (sin precioVentaNuevo ni vencimiento),
// con "despues" en null; un AGREGADO sin precio ni vencimiento tampoco los trae. La UI debe tolerarlo.
func TestQAAuditoriaJSONdeAgregadoYQuitado(t *testing.T) {
	e := nuevoB(t)
	e.conChronos()
	if _, err := e.editarLista(func(k, _ ItemEdicionCmd) []ItemEdicionCmd {
		return []ItemEdicionCmd{k, {ProductoID: "chronos", Cantidad: 2, CostoUnitario: d("15000")}}
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(e.repo.ediciones[0].Cambios)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Renglones []struct {
			Accion  string
			Cambios []map[string]any
		}
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Renglones) != 2 {
		t.Fatalf("doc = %s", raw)
	}
	for _, r := range doc.Renglones {
		campos := []string{}
		for _, c := range r.Cambios {
			campos = append(campos, c["campo"].(string))
		}
		if strings.Join(campos, ",") != "cantidad,costoUnitario" {
			t.Errorf("%s campos = %v", r.Accion, campos)
		}
		if r.Accion == "QUITADO" && (r.Cambios[0]["despues"] != nil || r.Cambios[0]["antes"] != float64(10)) {
			t.Errorf("QUITADO = %v", r.Cambios)
		}
	}
}

// El detalle tras quitar no muestra el renglón y el resumen (productos, unidades) lo refleja.
func TestQADetalleTrasQuitarNoMuestraElRenglon(t *testing.T) {
	e := nuevoB(t)
	det, err := e.editarLista(func(k, _ ItemEdicionCmd) []ItemEdicionCmd { return []ItemEdicionCmd{k} })
	if err != nil {
		t.Fatal(err)
	}
	if len(det.Items) != 1 || det.Items[0].ID != e.kaiak || det.Unidades != 4 {
		t.Errorf("detalle = %+v", det)
	}
}
