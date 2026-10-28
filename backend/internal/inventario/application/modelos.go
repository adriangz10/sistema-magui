package application

import (
	"time"

	"magui/internal/inventario/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
)

// --- Modelos de lectura ---

type LoteDTO struct {
	ID            string                   `json:"id"`
	Vencimiento   *fecha.Fecha             `json:"vencimiento"`
	Costo         *dinero.Dinero           `json:"costo"`
	Stock         int                      `json:"stock"`
	Estado        domain.EstadoVencimiento `json:"estado"`
	DiasRestantes *int                     `json:"diasRestantes"`
}

type StockProductoDTO struct {
	ProductoID string    `json:"productoId"`
	SKU        string    `json:"sku"`
	Nombre     string    `json:"nombre"`
	Stock      int       `json:"stock"`
	Lotes      []LoteDTO `json:"lotes"`
}

type MovimientoDTO struct {
	ID              string                `json:"id"`
	Fecha           time.Time             `json:"fecha"`
	ProductoID      string                `json:"productoId"`
	SKU             string                `json:"sku"`
	Producto        string                `json:"producto"`
	Cantidad        int                   `json:"cantidad"`
	Tipo            domain.TipoMovimiento `json:"tipo"`
	ReferenciaID    *string               `json:"referenciaId"`
	Nota            string                `json:"nota"`
	LoteVencimiento *fecha.Fecha          `json:"loteVencimiento"`
}

type VencimientoDTO struct {
	LoteID        string                   `json:"loteId"`
	ProductoID    string                   `json:"productoId"`
	SKU           string                   `json:"sku"`
	Producto      string                   `json:"producto"`
	Vencimiento   fecha.Fecha              `json:"vencimiento"`
	Stock         int                      `json:"stock"`
	Estado        domain.EstadoVencimiento `json:"estado"`
	DiasRestantes int                      `json:"diasRestantes"`
}

// AlertasDTO resume lo que requiere atención (para el inicio).
type AlertasDTO struct {
	Vencidos   int `json:"vencidos"`   // lotes vencidos con stock
	PorVencer  int `json:"porVencer"`  // lotes que vencen dentro del plazo de aviso
	SinStock   int `json:"sinStock"`   // productos activos con stock <= 0
	StockBajo  int `json:"stockBajo"`  // productos activos con 0 < stock <= mínimo
	DiasAlerta int `json:"diasAlerta"` // plazo de aviso configurado
}

// --- Comandos ---

// Ingreso de stock (compra o ajuste positivo). Crea un lote nuevo.
type Ingreso struct {
	ProductoID   string
	Cantidad     int
	Costo        *dinero.Dinero
	Vencimiento  *fecha.Fecha
	CompraItemID *string
	Tipo         domain.TipoMovimiento
	ReferenciaID *string
	Nota         string
}

// Salida de stock (venta o ajuste negativo). Se reparte entre lotes por FEFO.
type Salida struct {
	ProductoID   string
	Cantidad     int // positiva
	Tipo         domain.TipoMovimiento
	ReferenciaID *string
	Nota         string
	// PermitirNegativo deja vender sin stock (ventas offline que se sincronizan después).
	PermitirNegativo bool
}

// ComandoAjuste es una corrección manual de stock.
type ComandoAjuste struct {
	ProductoID  string
	Cantidad    int // positivo suma, negativo resta
	Motivo      string
	LoteID      string // opcional: lote al que se aplica
	Vencimiento *fecha.Fecha
	Costo       *dinero.Dinero
}

// AjusteLoteCompra corrige un lote desde la edición de la compra que lo originó (US-001b):
// una diferencia de cantidad (movimiento EDICION_COMPRA sobre el mismo lote) y el costo y
// vencimiento nuevos (se actualizan en el lugar, sin movimiento, si cambiaron).
type AjusteLoteCompra struct {
	LoteID      string
	Diferencia  int // positiva suma, negativa resta; 0 = sin movimiento
	Costo       dinero.Dinero
	Vencimiento *fecha.Fecha
	CompraID    string
	Nota        string
}

// EstadoVencimientos para filtrar la lista de vencimientos.
type EstadoVencimientos string

const (
	FiltroVencidos  EstadoVencimientos = "vencidos"
	FiltroPorVencer EstadoVencimientos = "por_vencer"
	FiltroTodos     EstadoVencimientos = "todos"
)
