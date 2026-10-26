package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"magui/internal/catalogo/application"
	"magui/internal/catalogo/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/id"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/platform/db"
)

// Integración: necesita MAGUI_TEST_MYSQL_DSN (ej. usuario:clave@tcp(localhost:3306)/).
// Prueba los casos de uso reales contra MySQL en una base temporal.

func baseTemporal(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("MAGUI_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("MAGUI_TEST_MYSQL_DSN no definido")
	}
	ctx := context.Background()
	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	nombre := fmt.Sprintf("magui_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + nombre + " CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec("DROP DATABASE " + nombre); admin.Close() })
	conn, err := db.OpenMySQL(ctx, dsn+nombre+"?parseTime=true&loc=UTC")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	m, _ := db.NewMigrator(conn, db.MySQL)
	if err := m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	return conn
}

func servicio(conn *sql.DB) *application.Servicio {
	return application.NuevoServicio(application.Dependencias{
		Empresas: Empresas{DB: conn}, Categorias: Categorias{DB: conn}, Productos: Productos{DB: conn},
		Tx: db.Transactor{DB: conn}, IDs: id.UUIDv7{}, Reloj: reloj.Sistema{},
	})
}

func debe[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func simple(sku, nombre, precio string, cats ...string) application.ComandoProducto {
	return application.ComandoProducto{Tipo: domain.TipoSimple, Datos: domain.DatosProducto{
		SKU: sku, Nombre: nombre, PrecioVenta: dinero.MustParse(precio), CategoriaIDs: cats, MostrarEnTienda: true,
	}}
}

// agregarStock inserta un movimiento como lo hará el módulo de inventario.
func agregarStock(t *testing.T, conn *sql.DB, productoID string, cant int) {
	t.Helper()
	if _, err := conn.Exec(`INSERT INTO movimiento_stock (id, producto_id, cantidad, tipo, fecha) VALUES (?, ?, ?, 'AJUSTE', NOW())`,
		id.Nuevo(), productoID, cant); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogoMySQL(t *testing.T) {
	conn := baseTemporal(t)
	s := servicio(conn)
	ctx := context.Background()
	pag := paginacion.Nueva(1, 10)

	// Empresas
	natura := debe(s.CrearEmpresa(ctx, "Natura"))
	if _, err := s.CrearEmpresa(ctx, "NATURA"); !errors.Is(err, domain.ErrEmpresaDuplicada) {
		t.Errorf("empresa duplicada: %v", err)
	}

	// Categorías
	perf := debe(s.CrearCategoria(ctx, "PERFUMERÍA", nil))
	fem := debe(s.CrearCategoria(ctx, "Perfumería femenina", &perf.ID))
	reg := debe(s.CrearCategoria(ctx, "REGALOS", nil))

	// Productos
	cmd := simple("P341355", "Kaiak Clásico femenino", "59200", fem.ID, reg.ID)
	cmd.Datos.EmpresaID, cmd.Datos.CodigoBarras, cmd.Datos.StockMinimo = natura.ID, "7790000000011", 2
	cmd.Datos.Costo = new(dinero.MustParse("30000"))
	kaiak := debe(s.CrearProducto(ctx, cmd))
	jabon := debe(s.CrearProducto(ctx, simple("P2", "Jabón Ekos", "5500", perf.ID)))
	if kaiak.Empresa == nil || *kaiak.Empresa != "Natura" || kaiak.Costo.String() != "30000.00" || len(kaiak.Categorias) != 2 {
		t.Fatalf("detalle = %+v", kaiak)
	}
	if _, err := s.CrearProducto(ctx, simple("7790000000011", "Otro", "1")); !errors.Is(err, domain.ErrSKUDuplicado) {
		t.Errorf("SKU igual a un código de barras: %v", err)
	}

	agregarStock(t, conn, kaiak.ID, 5)
	agregarStock(t, conn, jabon.ID, 3)

	// Combo: 1 Kaiak + 2 jabones -> con 5 y 3 en stock se pueden armar 1
	combo := debe(s.CrearProducto(ctx, application.ComandoProducto{Tipo: domain.TipoCombo, Datos: domain.DatosProducto{
		Nombre: "Combo Día de la Madre", PrecioVenta: dinero.MustParse("60000"), CategoriaIDs: []string{reg.ID},
		Items: []domain.ComboItem{{ProductoID: kaiak.ID, Cantidad: 1}, {ProductoID: jabon.ID, Cantidad: 2}},
	}}))
	stockComponente := map[string]int{}
	for _, it := range combo.Items {
		stockComponente[it.ProductoID] = it.Stock
	}
	if combo.Stock != 1 || len(combo.Items) != 2 || stockComponente[jabon.ID] != 3 || stockComponente[kaiak.ID] != 5 {
		t.Errorf("combo = stock %d items %+v", combo.Stock, combo.Items)
	}

	// Búsqueda por código (lector)
	if p := debe(s.BuscarPorCodigo(ctx, "7790000000011")); p.ID != kaiak.ID || p.Stock != 5 {
		t.Errorf("por código = %+v", p)
	}

	// Listados
	if r := debe(s.ListarProductos(ctx, application.ConsultaProductos{Texto: "kaiak clasico"}, pag)); r.Total != 1 {
		t.Errorf("búsqueda sin tildes = %d", r.Total)
	}
	if r := debe(s.ListarProductos(ctx, application.ConsultaProductos{CategoriaID: perf.ID}, pag)); r.Total != 2 {
		t.Errorf("PERFUMERÍA con subcategorías = %d", r.Total)
	}
	if r := debe(s.ListarProductos(ctx, application.ConsultaProductos{Tipo: domain.TipoCombo}, pag)); r.Total != 1 || r.Items[0].Stock != 1 {
		t.Errorf("combos = %+v", r)
	}
	// Copia completa para el escritorio
	exp := debe(s.ExportarCatalogo(ctx))
	if len(exp.Productos) != 3 || len(exp.Componentes) != 2 {
		t.Errorf("exportar: %d productos, %d componentes", len(exp.Productos), len(exp.Componentes))
	}
	for _, c := range exp.Componentes {
		if c.ComboID != combo.ID || (c.ProductoID == jabon.ID && c.Cantidad != 2) {
			t.Errorf("componente = %+v", c)
		}
	}

	agregarStock(t, conn, kaiak.ID, -4) // queda 1, mínimo 2 -> stock bajo
	if r := debe(s.ListarProductos(ctx, application.ConsultaProductos{StockBajo: true}, pag)); r.Total != 1 || r.Items[0].ID != kaiak.ID {
		t.Errorf("stock bajo = %+v", r)
	}
	if r := debe(s.ListarProductos(ctx, application.ConsultaProductos{Orden: application.OrdenPrecio}, pag)); r.Items[0].ID != jabon.ID {
		t.Errorf("orden por precio: primero %s", r.Items[0].Nombre)
	}

	// Edición con control de versión e historial
	datos := cmd.Datos
	datos.PrecioVenta = dinero.MustParse("61000")
	act := debe(s.ActualizarProducto(ctx, kaiak.ID, datos, kaiak.Version))
	if act.Version != 2 || act.PrecioVenta.String() != "61000.00" {
		t.Errorf("actualizado = %+v", act)
	}
	if _, err := s.ActualizarProducto(ctx, kaiak.ID, datos, kaiak.Version); !errors.Is(err, domain.ErrVersionDesactualizada) {
		t.Errorf("versión vieja: %v", err)
	}

	// Precios en cascada sobre PERFUMERÍA (+10 %, redondeo $100)
	res := debe(s.AplicarAjuste(ctx, application.ComandoAjuste{CategoriaID: perf.ID, IncluirSubcategorias: true,
		Ajuste: domain.AjustePrecio{Tipo: domain.AjustePorcentaje, Porcentaje: 1000, Redondeo: dinero.DesdePesos(100)}}))
	if res.Cantidad != 2 {
		t.Errorf("ajustados = %d", res.Cantidad)
	}
	if p := debe(s.ObtenerProducto(ctx, kaiak.ID)); p.PrecioVenta.String() != "67100.00" || p.Version != 3 {
		t.Errorf("kaiak tras cascada = %s v%d", p.PrecioVenta, p.Version)
	}
	hist := debe(s.HistorialPrecios(ctx, kaiak.ID, pag))
	if hist.Total != 2 || hist.Items[0].Origen != domain.OrigenCascada || hist.Items[1].Origen != domain.OrigenManual {
		t.Errorf("historial = %+v", hist.Items)
	}

	// Desactivar y filtrar
	debe(s.CambiarEstadoProducto(ctx, jabon.ID, false))
	if r := debe(s.ListarProductos(ctx, application.ConsultaProductos{Estado: application.EstadoInactivos}, pag)); r.Total != 1 {
		t.Errorf("inactivos = %d", r.Total)
	}

	// Árbol y borrado
	arbol := debe(s.ArbolCategorias(ctx))
	if len(arbol) != 2 || arbol[0].Nombre != "PERFUMERÍA" || arbol[0].Hijos[0].Productos != 1 {
		t.Errorf("árbol = %+v", arbol)
	}
	vacia := debe(s.CrearCategoria(ctx, "Vacía", nil))
	if err := s.EliminarCategoria(ctx, vacia.ID); err != nil {
		t.Errorf("borrar vacía: %v", err)
	}
	if err := s.EliminarCategoria(ctx, reg.ID); !errors.Is(err, domain.ErrCategoriaConProductos) {
		t.Errorf("borrar con productos: %v", err)
	}
}
