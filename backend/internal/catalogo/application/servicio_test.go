package application

import (
	"context"
	"errors"
	"testing"

	"magui/internal/catalogo/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/paginacion"
)

var ctx = context.Background()

func ptr[T any](v T) *T { return &v }

// debe devuelve el valor o hace fallar el test (con pánico) si hubo error.
func debe[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func simple(sku, nombre, precio string, cats ...string) ComandoProducto {
	return ComandoProducto{Tipo: domain.TipoSimple, Datos: domain.DatosProducto{
		SKU: sku, Nombre: nombre, PrecioVenta: dinero.MustParse(precio), CategoriaIDs: cats, MostrarEnTienda: true,
	}}
}

// --- Empresas ---

func TestEmpresas(t *testing.T) {
	e := nuevoEntorno()
	natura := debe(e.svc.CrearEmpresa(ctx, "Natura"))
	if _, err := e.svc.CrearEmpresa(ctx, " natura "); !errors.Is(err, domain.ErrEmpresaDuplicada) {
		t.Errorf("duplicada: %v", err)
	}
	act := debe(e.svc.ActualizarEmpresa(ctx, natura.ID, "Natura Cosméticos", false, natura.Version))
	if act.Nombre != "Natura Cosméticos" || act.Activo || act.Version != 2 {
		t.Errorf("actualizada = %+v", act)
	}
	if _, err := e.svc.ActualizarEmpresa(ctx, natura.ID, "X", true, 1); !errors.Is(err, domain.ErrVersionDesactualizada) {
		t.Errorf("versión vieja: %v", err)
	}
	pag := debe(e.svc.ListarEmpresas(ctx, FiltroEmpresas{SoloActivas: true}, paginacion.Nueva(1, 10)))
	if pag.Total != 0 {
		t.Errorf("solo activas: %+v", pag)
	}
}

// --- Categorías ---

func TestCategorias(t *testing.T) {
	e := nuevoEntorno()
	perf := debe(e.svc.CrearCategoria(ctx, "PERFUMERÍA", nil))
	fem := debe(e.svc.CrearCategoria(ctx, "Femenina", &perf.ID))
	kits := debe(e.svc.CrearCategoria(ctx, "Kits", &fem.ID))

	if _, err := e.svc.CrearCategoria(ctx, "perfumeria", nil); !errors.Is(err, domain.ErrCategoriaDuplicada) {
		t.Errorf("duplicada sin importar mayúsculas/tildes: %v", err)
	}
	debe(e.svc.CrearCategoria(ctx, "Kits", nil)) // mismo nombre en otro nivel: permitido
	if _, err := e.svc.CrearCategoria(ctx, "X", ptr("no-existe")); !errors.Is(err, domain.ErrCategoriaNoEncontrada) {
		t.Errorf("padre inexistente: %v", err)
	}
	if _, err := e.svc.ActualizarCategoria(ctx, perf.ID, "PERFUMERÍA", &kits.ID, perf.Version); !errors.Is(err, domain.ErrCategoriaCiclo) {
		t.Errorf("mover dentro de un descendiente: %v", err)
	}
	if err := e.svc.EliminarCategoria(ctx, fem.ID); !errors.Is(err, domain.ErrCategoriaConHijos) {
		t.Errorf("borrar con hijos: %v", err)
	}
	debe(e.svc.CrearProducto(ctx, simple("P1", "Kaiak", "100", kits.ID)))
	if err := e.svc.EliminarCategoria(ctx, kits.ID); !errors.Is(err, domain.ErrCategoriaConProductos) {
		t.Errorf("borrar con productos: %v", err)
	}

	arbol := debe(e.svc.ArbolCategorias(ctx))
	if len(arbol) != 2 || arbol[0].Nombre != "Kits" || arbol[1].Nombre != "PERFUMERÍA" {
		t.Fatalf("raíces ordenadas = %+v", arbol)
	}
	if got := arbol[1].Hijos[0].Hijos[0]; got.Nombre != "Kits" || got.Productos != 1 {
		t.Errorf("nieto = %+v", got)
	}
}

// --- Productos ---

func TestCrearProductoYValidaciones(t *testing.T) {
	e := nuevoEntorno()
	cat := debe(e.svc.CrearCategoria(ctx, "REGALOS", nil))
	cmd := simple("P1", "Kaiak", "59200", cat.ID)
	cmd.Datos.CodigoBarras = "7791234"
	p := debe(e.svc.CrearProducto(ctx, cmd))
	if p.SKU != "P1" || len(p.Categorias) != 1 || p.Categorias[0].Ruta != "REGALOS" {
		t.Errorf("detalle = %+v", p)
	}

	casos := map[string]struct {
		cmd  ComandoProducto
		want error
	}{
		"sku repetido":            {simple("P1", "Otro", "1"), domain.ErrSKUDuplicado},
		"sku igual a un código":   {simple("7791234", "Otro", "1"), domain.ErrSKUDuplicado},
		"categoría inexistente":   {simple("P2", "Otro", "1", "nope"), domain.ErrCategoriaNoEncontrada},
		"empresa inexistente":     {func() ComandoProducto { c := simple("P3", "Otro", "1"); c.Datos.EmpresaID = "nope"; return c }(), domain.ErrEmpresaNoEncontrada},
		"código de barras en uso": {func() ComandoProducto { c := simple("P4", "Otro", "1"); c.Datos.CodigoBarras = "P1"; return c }(), domain.ErrCodigoBarrasDuplicado},
	}
	for nombre, c := range casos {
		if _, err := e.svc.CrearProducto(ctx, c.cmd); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", nombre, err, c.want)
		}
	}

	porCodigo := debe(e.svc.BuscarPorCodigo(ctx, " 7791234 "))
	if porCodigo.ID != p.ID {
		t.Error("buscar por código de barras")
	}
	if _, err := e.svc.BuscarPorCodigo(ctx, "nada"); !errors.Is(err, domain.ErrProductoNoEncontrado) {
		t.Errorf("código inexistente: %v", err)
	}
}

func TestCombos(t *testing.T) {
	e := nuevoEntorno()
	crema := debe(e.svc.CrearProducto(ctx, simple("P1", "Crema", "100")))
	jabon := debe(e.svc.CrearProducto(ctx, simple("P2", "Jabón", "50")))

	combo := ComandoProducto{Tipo: domain.TipoCombo, Datos: domain.DatosProducto{
		Nombre: "Día de la Madre", PrecioVenta: dinero.MustParse("130"),
		Items: []domain.ComboItem{{ProductoID: crema.ID, Cantidad: 1}, {ProductoID: jabon.ID, Cantidad: 2}},
	}}
	c := debe(e.svc.CrearProducto(ctx, combo))
	if c.Tipo != domain.TipoCombo || len(c.Items) != 2 || c.SKU == "" {
		t.Fatalf("combo = %+v", c)
	}

	// Un combo no puede contener otro combo
	otro := combo
	otro.Datos.Items = []domain.ComboItem{{ProductoID: c.ID, Cantidad: 2}}
	if _, err := e.svc.CrearProducto(ctx, otro); err == nil {
		t.Error("combo dentro de combo debería fallar")
	}
	// Ni productos desactivados o inexistentes
	debe(e.svc.CambiarEstadoProducto(ctx, jabon.ID, false))
	if _, err := e.svc.CrearProducto(ctx, combo); err == nil {
		t.Error("componente desactivado debería fallar")
	}
	combo.Datos.Items = []domain.ComboItem{{ProductoID: "fantasma", Cantidad: 2}}
	if _, err := e.svc.CrearProducto(ctx, combo); err == nil {
		t.Error("componente inexistente debería fallar")
	}
}

func TestActualizarProductoRegistraPrecio(t *testing.T) {
	e := nuevoEntorno()
	p := debe(e.svc.CrearProducto(ctx, simple("P1", "Kaiak", "100")))
	datos := simple("P1", "Kaiak Clásico", "120").Datos
	act := debe(e.svc.ActualizarProducto(ctx, p.ID, datos, p.Version))
	if act.Nombre != "Kaiak Clásico" || act.PrecioVenta.String() != "120.00" || act.Version != 2 {
		t.Errorf("actualizado = %+v", act)
	}
	hist := debe(e.svc.HistorialPrecios(ctx, p.ID, paginacion.Nueva(1, 10)))
	if hist.Total != 1 || hist.Items[0].Origen != domain.OrigenManual || hist.Items[0].PrecioAnterior.String() != "100.00" {
		t.Errorf("historial = %+v", hist)
	}
	if _, err := e.svc.ActualizarProducto(ctx, p.ID, datos, 1); !errors.Is(err, domain.ErrVersionDesactualizada) {
		t.Errorf("versión vieja: %v", err)
	}
	// Cambiar solo el nombre no agrega historial
	datos.Nombre = "Kaiak"
	debe(e.svc.ActualizarProducto(ctx, p.ID, datos, 2))
	if len(e.productos.historial) != 1 {
		t.Errorf("no debería registrar precio sin cambio: %d", len(e.productos.historial))
	}
}

func TestListarPorCategoriaIncluyeSubcategorias(t *testing.T) {
	e := nuevoEntorno()
	perf := debe(e.svc.CrearCategoria(ctx, "PERFUMERÍA", nil))
	fem := debe(e.svc.CrearCategoria(ctx, "Femenina", &perf.ID))
	debe(e.svc.CrearProducto(ctx, simple("P1", "A", "1", fem.ID)))
	debe(e.svc.CrearProducto(ctx, simple("P2", "B", "1", perf.ID)))
	debe(e.svc.CrearProducto(ctx, simple("P3", "C", "1")))

	pag := debe(e.svc.ListarProductos(ctx, ConsultaProductos{CategoriaID: perf.ID}, paginacion.Nueva(1, 10)))
	if pag.Total != 2 {
		t.Errorf("productos en PERFUMERÍA y subcategorías = %d", pag.Total)
	}
	if _, err := e.svc.ListarProductos(ctx, ConsultaProductos{CategoriaID: "nope"}, paginacion.Nueva(1, 10)); err == nil {
		t.Error("categoría inexistente debería fallar")
	}
}

// --- Precios en cascada ---

func TestAjustePrecioEnCascada(t *testing.T) {
	e := nuevoEntorno()
	perf := debe(e.svc.CrearCategoria(ctx, "PERFUMERÍA", nil))
	fem := debe(e.svc.CrearCategoria(ctx, "Femenina", &perf.ID))
	a := debe(e.svc.CrearProducto(ctx, simple("P1", "A", "59200", fem.ID)))
	b := debe(e.svc.CrearProducto(ctx, simple("P2", "B", "1000", perf.ID)))
	inactivo := debe(e.svc.CrearProducto(ctx, simple("P3", "C", "1000", perf.ID)))
	debe(e.svc.CambiarEstadoProducto(ctx, inactivo.ID, false))

	cmd := ComandoAjuste{CategoriaID: perf.ID, IncluirSubcategorias: true, Motivo: "Aumento Natura",
		Ajuste: domain.AjustePrecio{Tipo: domain.AjustePorcentaje, Porcentaje: 1000, Redondeo: dinero.DesdePesos(100)}}

	previa := debe(e.svc.VistaPreviaAjuste(ctx, cmd))
	if previa.Cantidad != 2 || previa.Aplicado {
		t.Fatalf("vista previa = %+v", previa)
	}
	if p, _ := e.productos.ObtenerPorID(ctx, a.ID); p.PrecioVenta.String() != "59200.00" {
		t.Fatal("la vista previa no debe guardar cambios")
	}

	res := debe(e.svc.AplicarAjuste(ctx, cmd))
	if !res.Aplicado || res.Cantidad != 2 {
		t.Fatalf("resultado = %+v", res)
	}
	pa, _ := e.productos.ObtenerPorID(ctx, a.ID)
	pb, _ := e.productos.ObtenerPorID(ctx, b.ID)
	pi, _ := e.productos.ObtenerPorID(ctx, inactivo.ID)
	if pa.PrecioVenta.String() != "65100.00" || pb.PrecioVenta.String() != "1100.00" || pi.PrecioVenta.String() != "1000.00" {
		t.Errorf("precios = %s %s %s", pa.PrecioVenta, pb.PrecioVenta, pi.PrecioVenta)
	}
	if len(e.productos.historial) != 2 || e.productos.historial[0].Origen != domain.OrigenCascada {
		t.Errorf("historial = %+v", e.productos.historial)
	}

	// Sin subcategorías: solo los asignados directamente
	cmd.IncluirSubcategorias = false
	if r := debe(e.svc.VistaPreviaAjuste(ctx, cmd)); r.Cantidad != 1 || r.Productos[0].ProductoID != b.ID {
		t.Errorf("solo directos = %+v", r)
	}
}

func TestAjusteQueDejaPrecioInvalidoNoCambiaNada(t *testing.T) {
	e := nuevoEntorno()
	cat := debe(e.svc.CrearCategoria(ctx, "X", nil))
	caro := debe(e.svc.CrearProducto(ctx, simple("P1", "Caro", "1000", cat.ID)))
	debe(e.svc.CrearProducto(ctx, simple("P2", "Barato", "100", cat.ID)))

	cmd := ComandoAjuste{CategoriaID: cat.ID, Ajuste: domain.AjustePrecio{Tipo: domain.AjusteMonto, Monto: dinero.MustParse("-500")}}
	if _, err := e.svc.AplicarAjuste(ctx, cmd); err == nil {
		t.Fatal("debería fallar: Barato quedaría negativo")
	}
	if p, _ := e.productos.ObtenerPorID(ctx, caro.ID); p.PrecioVenta.String() != "1000.00" {
		t.Errorf("no debería cambiar ningún precio: %s", p.PrecioVenta)
	}
}

func TestRegistrarCostosCompra(t *testing.T) {
	e := nuevoEntorno()
	a := debe(e.svc.CrearProducto(ctx, simple("P1", "A", "100")))
	b := debe(e.svc.CrearProducto(ctx, simple("P2", "B", "200")))
	nuevo := dinero.MustParse("120")
	err := e.svc.RegistrarCostosCompra(ctx, []CostoCompra{
		{ProductoID: a.ID, Costo: dinero.MustParse("60"), PrecioVenta: &nuevo},
		{ProductoID: b.ID, Costo: dinero.MustParse("90")}, // sin precio nuevo
	}, "Compra Natura")
	if err != nil {
		t.Fatal(err)
	}
	pa, _ := e.productos.ObtenerPorID(ctx, a.ID)
	pb, _ := e.productos.ObtenerPorID(ctx, b.ID)
	if pa.Costo.String() != "60.00" || pa.PrecioVenta.String() != "120.00" || pb.Costo.String() != "90.00" || pb.PrecioVenta.String() != "200.00" {
		t.Errorf("A = %s/%s, B = %s/%s", pa.Costo, pa.PrecioVenta, pb.Costo, pb.PrecioVenta)
	}
	if len(e.productos.historial) != 1 || e.productos.historial[0].Origen != domain.OrigenCompra {
		t.Errorf("historial = %+v", e.productos.historial)
	}
	info := debe(e.svc.InfoProductos(ctx, []string{a.ID, "nope"}))
	if len(info) != 1 || info[a.ID].Costo.String() != "60.00" {
		t.Errorf("info = %+v", info)
	}
}
