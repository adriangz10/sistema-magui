package domain

import (
	"errors"
	"testing"
	"time"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
)

var ahora = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

func ptr[T any](v T) *T { return &v }

func campos(err error) map[string]string {
	e, _ := errores.Como(err)
	if e == nil {
		return nil
	}
	return e.Campos
}

func datosValidos() DatosProducto {
	return DatosProducto{SKU: " P1 ", Nombre: "  Kaiak   Clásico ", PrecioVenta: dinero.MustParse("59200"),
		CategoriaIDs: []string{"c1", "c1", " ", "c2"}, MostrarEnTienda: true}
}

func TestNuevoProducto(t *testing.T) {
	p, err := NuevoProducto("id1", TipoSimple, datosValidos(), ahora)
	if err != nil {
		t.Fatal(err)
	}
	if p.SKU != "P1" || p.Nombre != "Kaiak Clásico" || p.Version != 1 || !p.Activo {
		t.Errorf("producto = %+v", p)
	}
	if len(p.CategoriaIDs) != 2 {
		t.Errorf("categorías sin duplicados = %v", p.CategoriaIDs)
	}
	if p.CodigoBarras != nil || p.Descripcion != nil || p.EmpresaID != nil {
		t.Error("los textos vacíos deberían quedar en nil")
	}
}

func TestNuevoProductoSKUAutomatico(t *testing.T) {
	d := datosValidos()
	d.SKU = ""
	p, _ := NuevoProducto("0192f0c1-aaaa-7bbb-8ccc-1234abcd5678", TipoSimple, d, ahora)
	if p.SKU != "MG-ABCD5678" {
		t.Errorf("sku = %q", p.SKU)
	}
}

func TestProductoValidaciones(t *testing.T) {
	d := DatosProducto{SKU: "con espacio", CodigoBarras: "77 9", PrecioVenta: 0, StockMinimo: -1,
		Costo: ptr(dinero.MustParse("-1")), Items: []ComboItem{{ProductoID: "x", Cantidad: 2}}}
	_, err := NuevoProducto("id", TipoSimple, d, ahora)
	c := campos(err)
	for _, campo := range []string{"nombre", "sku", "codigoBarras", "precioVenta", "stockMinimo", "costo", "items"} {
		if c[campo] == "" {
			t.Errorf("falta error en %s (errores: %v)", campo, c)
		}
	}
}

func TestComboValidaciones(t *testing.T) {
	casos := map[string][]ComboItem{
		"vacío":        nil,
		"sí mismo":     {{ProductoID: "combo", Cantidad: 2}},
		"cantidad 0":   {{ProductoID: "a", Cantidad: 0}, {ProductoID: "b", Cantidad: 1}},
		"repetido":     {{ProductoID: "a", Cantidad: 1}, {ProductoID: "a", Cantidad: 1}},
		"una unidad":   {{ProductoID: "a", Cantidad: 1}},
		"sin producto": {{ProductoID: "", Cantidad: 2}},
	}
	for nombre, items := range casos {
		d := datosValidos()
		d.Items = items
		_, err := NuevoProducto("combo", TipoCombo, d, ahora)
		if e, ok := errores.Como(err); !ok || e.Codigo != "combo_invalido" {
			t.Errorf("%s: err = %v", nombre, err)
		}
	}
	d := datosValidos()
	d.SKU = ""
	d.Items = []ComboItem{{ProductoID: "a", Cantidad: 2}}
	p, err := NuevoProducto("combo-1234", TipoCombo, d, ahora)
	if err != nil || p.SKU != "CB-OMBO1234" || len(p.Items) != 1 { // últimos 8 caracteres del ID
		t.Errorf("combo válido: %+v, %v", p, err)
	}
}

func TestModificarDevuelvePrecioAnterior(t *testing.T) {
	p, _ := NuevoProducto("id", TipoSimple, datosValidos(), ahora)
	d := datosValidos()
	if ant, _ := p.Modificar(d, ahora); ant != nil {
		t.Error("sin cambio de precio no debería devolver anterior")
	}
	d.PrecioVenta = dinero.MustParse("60000")
	ant, err := p.Modificar(d, ahora)
	if err != nil || ant == nil || ant.String() != "59200.00" || p.Version != 3 {
		t.Errorf("anterior = %v, versión = %d, err = %v", ant, p.Version, err)
	}
}

func TestCambiarPrecioYEstado(t *testing.T) {
	p, _ := NuevoProducto("id", TipoSimple, datosValidos(), ahora)
	if _, err := p.CambiarPrecio(0, ahora); err == nil {
		t.Error("precio cero debería rechazarse")
	}
	ant, err := p.CambiarPrecio(dinero.MustParse("65000"), ahora)
	if err != nil || ant.String() != "59200.00" || p.PrecioVenta.String() != "65000.00" {
		t.Errorf("cambiar precio: %v %v", ant, err)
	}
	v := p.Version
	p.CambiarEstado(true, ahora) // ya activo: no cambia versión
	if p.Version != v {
		t.Error("no debería incrementar versión si no cambia")
	}
	p.CambiarEstado(false, ahora)
	if p.Activo || p.Version != v+1 {
		t.Error("debería desactivarse")
	}
}

func TestEmpresaYCategoria(t *testing.T) {
	if _, err := NuevaEmpresa("e", "  ", ahora); campos(err)["nombre"] == "" {
		t.Error("empresa sin nombre")
	}
	e, _ := NuevaEmpresa("e", " Natura ", ahora)
	if e.Nombre != "Natura" || e.Version != 1 || !e.Activo {
		t.Errorf("empresa = %+v", e)
	}
	if _, err := NuevaCategoria("c", "X", ptr("c"), ahora); !errors.Is(err, ErrCategoriaCiclo) {
		t.Errorf("padre de sí misma: %v", err)
	}
	c, _ := NuevaCategoria("c", "X", ptr(""), ahora)
	if c.PadreID != nil {
		t.Error("padre vacío debería quedar nil")
	}
}

func TestArbol(t *testing.T) {
	// PERFUMERIA > Femenina > Kits ; REGALOS
	a := NuevoArbol([]Categoria{
		{ID: "perf", Nombre: "PERFUMERÍA"},
		{ID: "fem", Nombre: "Femenina", PadreID: ptr("perf")},
		{ID: "kits", Nombre: "Kits", PadreID: ptr("fem")},
		{ID: "reg", Nombre: "REGALOS"},
	})
	if got := a.ConDescendientes("perf"); len(got) != 3 {
		t.Errorf("descendientes = %v", got)
	}
	if a.PuedeMoverA("perf", ptr("kits")) {
		t.Error("mover dentro de un descendiente debería formar ciclo")
	}
	if !a.PuedeMoverA("kits", ptr("reg")) || !a.PuedeMoverA("kits", nil) {
		t.Error("movimientos válidos rechazados")
	}
	if r := a.Ruta("kits"); len(r) != 3 || r[0] != "PERFUMERÍA" || r[2] != "Kits" {
		t.Errorf("ruta = %v", r)
	}
	if len(a.Raices()) != 2 {
		t.Errorf("raíces = %v", a.Raices())
	}
}

func TestAjustePrecio(t *testing.T) {
	pct := AjustePrecio{Tipo: AjustePorcentaje, Porcentaje: 1000, Redondeo: dinero.DesdePesos(100)}
	if err := pct.Validar(); err != nil {
		t.Fatal(err)
	}
	if got := pct.Aplicar(dinero.MustParse("59200")).String(); got != "65100.00" { // 65120 -> 65100
		t.Errorf("porcentaje = %s", got)
	}
	monto := AjustePrecio{Tipo: AjusteMonto, Monto: dinero.MustParse("-200")}
	if got := monto.Aplicar(dinero.MustParse("5500")).String(); got != "5300.00" {
		t.Errorf("monto = %s", got)
	}
	invalidos := []AjustePrecio{
		{Tipo: AjustePorcentaje},
		{Tipo: AjustePorcentaje, Porcentaje: 100001},
		{Tipo: AjusteMonto},
		{Tipo: "OTRO"},
		{Tipo: AjusteMonto, Monto: 1, Redondeo: -1},
	}
	for _, a := range invalidos {
		if a.Validar() == nil {
			t.Errorf("debería ser inválido: %+v", a)
		}
	}
}
