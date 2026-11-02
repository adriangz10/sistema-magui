// Package application contiene los casos de uso de ventas.
package application

import (
	"context"
	"time"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/paginacion"
	"magui/internal/ventas/domain"
)

// Repositorio persiste ventas.
type Repositorio interface {
	// Guardar inserta la venta con sus renglones y pagos.
	Guardar(ctx context.Context, v domain.Venta) error
	// Existe indica si ya hay una venta con ese ID (para no duplicar ventas sincronizadas).
	Existe(ctx context.Context, id string) (bool, error)
	// Obtener trae los datos básicos (sin renglones ni pagos) para operar sobre la venta.
	Obtener(ctx context.Context, id string) (domain.Venta, error)
	// MarcarAnulada falla con domain.ErrVentaYaAnulada si ya estaba anulada.
	MarcarAnulada(ctx context.Context, v domain.Venta) error
	Detalle(ctx context.Context, id string) (VentaDetalle, error)
	Listar(ctx context.Context, f Filtro, p paginacion.Solicitud) ([]VentaResumen, int, error)
}

// Componente de un combo.
type Componente struct {
	ProductoID string
	Cantidad   int
}

// ProductoVenta son los datos del catálogo que necesita una venta.
type ProductoVenta struct {
	ID          string
	Nombre      string
	Combo       bool
	Activo      bool
	Precio      dinero.Dinero
	CostoUltimo *dinero.Dinero
	Componentes []Componente
}

// Catalogo es el puerto hacia el catálogo.
type Catalogo interface {
	Productos(ctx context.Context, ids []string) (map[string]ProductoVenta, error)
}

// SalidaVenta descuenta stock de un producto simple.
type SalidaVenta struct {
	ProductoID       string
	Cantidad         int
	VentaID          string
	PermitirSinStock bool
}

// ParteSalida indica cuántas unidades salieron de un lote y a qué costo (nil si el lote no lo tiene).
type ParteSalida struct {
	Cantidad int
	Costo    *dinero.Dinero
}

// Caja verifica que haya una caja abierta antes de cobrar (US-002, D-1).
// Es opcional: sin Caja no se verifica (la app de escritorio cobra sin conexión).
type Caja interface {
	ExigirAbierta(ctx context.Context) error
}

// Inventario es el puerto hacia el inventario.
type Inventario interface {
	Descontar(ctx context.Context, s SalidaVenta) ([]ParteSalida, error)
	// Reponer devuelve el stock que salió por la venta (anulación).
	Reponer(ctx context.Context, ventaID, nota string) error
}

// --- Modelos de lectura ---

type VentaResumen struct {
	ID        string              `json:"id"`
	Fecha     time.Time           `json:"fecha"`
	Total     dinero.Dinero       `json:"total"`
	Estado    domain.Estado       `json:"estado"`
	Origen    domain.Origen       `json:"origen"`
	Productos int                 `json:"productos"` // renglones
	Unidades  int                 `json:"unidades"`
	Metodos   []domain.MetodoPago `json:"metodos"`
}

type ItemDTO struct {
	ProductoID     string        `json:"productoId"`
	SKU            string        `json:"sku"`
	Producto       string        `json:"producto"`
	Tipo           string        `json:"tipo"`
	Cantidad       int           `json:"cantidad"`
	PrecioUnitario dinero.Dinero `json:"precioUnitario"`
	Subtotal       dinero.Dinero `json:"subtotal"`
}

type PagoDTO struct {
	Metodo        domain.MetodoPago `json:"metodo"`
	Monto         dinero.Dinero     `json:"monto"`
	MontoRecibido *dinero.Dinero    `json:"montoRecibido"`
	Vuelto        *dinero.Dinero    `json:"vuelto"`
}

type VentaDetalle struct {
	VentaResumen
	CostoTotal dinero.Dinero `json:"costoTotal"`
	Ganancia   dinero.Dinero `json:"ganancia"`
	// CostoCompleto es false si algún producto vendido no tenía costo: la ganancia es aproximada.
	CostoCompleto   bool          `json:"costoCompleto"`
	Vuelto          dinero.Dinero `json:"vuelto"`
	AnuladaEn       *time.Time    `json:"anuladaEn"`
	MotivoAnulacion *string       `json:"motivoAnulacion"`
	Items           []ItemDTO     `json:"items"`
	Pagos           []PagoDTO     `json:"pagos"`
}

// Filtro del listado de ventas.
type Filtro struct {
	Desde  *time.Time
	Hasta  *time.Time // exclusivo
	Estado domain.Estado
	Metodo domain.MetodoPago
}

// --- Comandos ---

type LineaCmd struct {
	ProductoID string
	Cantidad   int
}

// ComandoVenta registra una venta.
type ComandoVenta struct {
	Items  []LineaCmd
	Pagos  []domain.PagoIngresado
	Origen domain.Origen
	// PermitirSinStock: la cajera confirmó vender aunque el sistema no tenga stock registrado.
	PermitirSinStock bool
}
