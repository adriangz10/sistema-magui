// Package application contiene los casos de uso del inventario.
package application

import (
	"context"
	"time"

	"magui/internal/inventario/domain"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/paginacion"
)

// Repositorio persiste lotes y movimientos y resuelve las consultas de stock.
type Repositorio interface {
	CrearLote(ctx context.Context, l domain.Lote) error
	ObtenerLote(ctx context.Context, id string) (domain.LoteStock, error)
	RegistrarMovimientos(ctx context.Context, m []domain.Movimiento) error
	// LotesConStock devuelve los lotes del producto con stock distinto de cero.
	LotesConStock(ctx context.Context, productoID string) ([]domain.LoteStock, error)
	StockProducto(ctx context.Context, productoID string) (int, error)

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
