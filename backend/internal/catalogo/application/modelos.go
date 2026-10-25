// Package application contiene los casos de uso del catálogo y los puertos que necesitan.
package application

import (
	"time"

	"magui/internal/catalogo/domain"
	"magui/internal/kernel/dinero"
)

// --- Modelos de lectura (lo que devuelve la API) ---

type EmpresaDTO struct {
	ID      string `json:"id"`
	Nombre  string `json:"nombre"`
	Activo  bool   `json:"activo"`
	Version int    `json:"version"`
}

func empresaDTO(e domain.Empresa) EmpresaDTO {
	return EmpresaDTO{ID: e.ID, Nombre: e.Nombre, Activo: e.Activo, Version: e.Version}
}

type CategoriaDTO struct {
	ID      string  `json:"id"`
	Nombre  string  `json:"nombre"`
	PadreID *string `json:"padreId"`
	Version int     `json:"version"`
}

func categoriaDTO(c domain.Categoria) CategoriaDTO {
	return CategoriaDTO{ID: c.ID, Nombre: c.Nombre, PadreID: c.PadreID, Version: c.Version}
}

// CategoriaNodo es una categoría con sus subcategorías, para mostrar el árbol.
type CategoriaNodo struct {
	CategoriaDTO
	Productos int             `json:"productos"` // asignados directamente
	Hijos     []CategoriaNodo `json:"hijos"`
}

// CategoriaRef identifica una categoría con su ruta legible ("PERFUMERÍA > Femenina").
type CategoriaRef struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Ruta   string `json:"ruta"`
}

// ProductoResumen es una fila del listado de productos.
type ProductoResumen struct {
	ID           string         `json:"id"`
	SKU          string         `json:"sku"`
	CodigoBarras *string        `json:"codigoBarras"`
	Nombre       string         `json:"nombre"`
	Tipo         domain.Tipo    `json:"tipo"`
	EmpresaID    *string        `json:"empresaId"`
	Empresa      *string        `json:"empresa"`
	PrecioVenta  dinero.Dinero  `json:"precioVenta"`
	Costo        *dinero.Dinero `json:"costo"`
	// Stock: en combos, cuántos se pueden armar con el stock de sus componentes.
	Stock       int  `json:"stock"`
	StockMinimo int  `json:"stockMinimo"`
	Activo      bool `json:"activo"`
	Version     int  `json:"version"`
}

// ComboItemDTO es un componente de combo con sus datos.
type ComboItemDTO struct {
	ProductoID  string        `json:"productoId"`
	SKU         string        `json:"sku"`
	Nombre      string        `json:"nombre"`
	Cantidad    int           `json:"cantidad"`
	PrecioVenta dinero.Dinero `json:"precioVenta"`
	Stock       int           `json:"stock"`
}

// ProductoDetalle es la ficha completa de un producto.
type ProductoDetalle struct {
	ProductoResumen
	Descripcion     *string        `json:"descripcion"`
	MostrarEnTienda bool           `json:"mostrarEnTienda"`
	Categorias      []CategoriaRef `json:"categorias"`
	Items           []ComboItemDTO `json:"items"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type CambioPrecioDTO struct {
	ID             string              `json:"id"`
	PrecioAnterior dinero.Dinero       `json:"precioAnterior"`
	PrecioNuevo    dinero.Dinero       `json:"precioNuevo"`
	Origen         domain.OrigenPrecio `json:"origen"`
	Motivo         string              `json:"motivo"`
	Fecha          time.Time           `json:"fecha"`
}

// PrecioPrevisto es una fila de la vista previa de un cambio masivo.
type PrecioPrevisto struct {
	ProductoID     string        `json:"productoId"`
	SKU            string        `json:"sku"`
	Nombre         string        `json:"nombre"`
	PrecioAnterior dinero.Dinero `json:"precioAnterior"`
	PrecioNuevo    dinero.Dinero `json:"precioNuevo"`
}

// ResultadoAjuste resume un cambio masivo de precios.
type ResultadoAjuste struct {
	Cantidad  int              `json:"cantidad"`
	Aplicado  bool             `json:"aplicado"`
	Productos []PrecioPrevisto `json:"productos"`
}

// --- Filtros y comandos ---

type FiltroEmpresas struct {
	Texto       string
	SoloActivas bool
}

// Estado de producto para filtrar.
type Estado string

const (
	EstadoActivos   Estado = "activos"
	EstadoInactivos Estado = "inactivos"
	EstadoTodos     Estado = "todos"
)

// Orden del listado de productos.
type Orden string

const (
	OrdenNombre    Orden = "nombre"
	OrdenRecientes Orden = "recientes"
	OrdenPrecio    Orden = "precio"
	OrdenStock     Orden = "stock"
)

// ConsultaProductos es lo que pide la pantalla de productos.
type ConsultaProductos struct {
	Texto       string // busca en nombre, SKU y código de barras
	CategoriaID string // incluye sus subcategorías
	EmpresaID   string
	Tipo        domain.Tipo
	Estado      Estado
	StockBajo   bool // stock <= stock mínimo (solo simples)
	Orden       Orden
}

// FiltroProductos es la consulta ya resuelta que recibe el repositorio.
type FiltroProductos struct {
	Texto        string
	CategoriaIDs []string // vacío = todas
	EmpresaID    string
	Tipo         domain.Tipo
	Estado       Estado
	StockBajo    bool
	Orden        Orden
}

// ComandoProducto crea o modifica un producto.
type ComandoProducto struct {
	Tipo  domain.Tipo // solo al crear
	Datos domain.DatosProducto
}

// ComandoAjuste es un cambio masivo de precios sobre una categoría.
type ComandoAjuste struct {
	CategoriaID          string
	Ajuste               domain.AjustePrecio
	IncluirSubcategorias bool
	Motivo               string
}
