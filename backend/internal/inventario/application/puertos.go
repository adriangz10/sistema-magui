// Package application contiene los casos de uso del inventario.
package application

import (
	"context"
	"time"

	"magui/internal/inventario/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/paginacion"
)

// Repositorio persiste lotes y movimientos y resuelve las consultas de stock.
type Repositorio interface {
	CrearLote(ctx context.Context, l domain.Lote) error
	// ObtenerLote devuelve el lote con su stock. Con bloquear, toma el lote con FOR UPDATE y lee el
	// stock con una lectura bloqueante (lo último confirmado, ADR 0002).
	ObtenerLote(ctx context.Context, id string, bloquear bool) (domain.LoteStock, error)
	RegistrarMovimientos(ctx context.Context, m []domain.Movimiento) error
	// MovimientosPorReferencia devuelve los movimientos de un tipo originados por una operación.
	MovimientosPorReferencia(ctx context.Context, referenciaID string, tipo domain.TipoMovimiento) ([]domain.Movimiento, error)
	// LotesConStock devuelve los lotes del producto con stock distinto de cero. Con bloquear,
	// primero bloquea todos los lotes del producto (ordenados por id) y lee el stock con una
	// lectura bloqueante, así dos salidas del mismo producto quedan serializadas (ADR 0002).
	LotesConStock(ctx context.Context, productoID string, bloquear bool) ([]domain.LoteStock, error)
	// BloquearLotesDeProducto toma FOR UPDATE sobre todos los lotes del producto, ordenados por id.
	BloquearLotesDeProducto(ctx context.Context, productoID string) error
	// LotesPorCompraItem devuelve los lotes (con su stock) de los renglones de compra indicados.
	// Con bloquear los bloquea ordenados por id y lee el stock de forma bloqueante.
	LotesPorCompraItem(ctx context.Context, itemIDs []string, bloquear bool) ([]domain.LoteStock, error)
	// ActualizarLote cambia el costo y el vencimiento de un lote en el lugar (sin movimiento).
	ActualizarLote(ctx context.Context, id string, costo *dinero.Dinero, vencimiento *fecha.Fecha) error
	StockProducto(ctx context.Context, productoID string) (int, error)
	// StockSinLote suma los movimientos del producto que no pertenecen a ningún lote (ventas offline sin stock).
	StockSinLote(ctx context.Context, productoID string) (int, error)

	ListarMovimientos(ctx context.Context, f FiltroMovimientos, p paginacion.Solicitud) ([]MovimientoDTO, int, error)
	ListarVencimientos(ctx context.Context, f FiltroVencimientos, p paginacion.Solicitud) ([]VencimientoDTO, int, error)
	ContarAlertas(ctx context.Context, hoy, limitePorVencer fecha.Fecha) (AlertasDTO, error)

	DiasAlerta(ctx context.Context) (int, error)
	GuardarDiasAlerta(ctx context.Context, dias int) error
}

// InfoProducto son los datos del catálogo que necesita el inventario.
type InfoProducto struct {
	ID     string
	SKU    string
	Nombre string
	Combo  bool
	Activo bool
}

// Productos es el puerto hacia el catálogo.
type Productos interface {
	// Info devuelve los productos encontrados (los inexistentes no aparecen en el mapa).
	Info(ctx context.Context, ids []string) (map[string]InfoProducto, error)
}

// FiltroMovimientos para el historial de movimientos.
type FiltroMovimientos struct {
	ProductoID string
	Tipo       domain.TipoMovimiento
	Desde      *time.Time
	Hasta      *time.Time // exclusivo
}

// FiltroVencimientos: lotes con stock y fecha de vencimiento en el rango.
type FiltroVencimientos struct {
	Desde *fecha.Fecha // inclusivo
	Hasta *fecha.Fecha // exclusivo
}
