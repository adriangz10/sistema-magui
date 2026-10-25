// Package domain contiene las reglas del inventario: lotes con vencimiento, movimientos
// de stock (el stock es la suma de movimientos), asignación FEFO y estados de vencimiento.
package domain

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
)

// TipoMovimiento indica el origen de un cambio de stock.
type TipoMovimiento string

const (
	MovInicial   TipoMovimiento = "INICIAL"
	MovCompra    TipoMovimiento = "COMPRA"
	MovVenta     TipoMovimiento = "VENTA"
	MovAnulacion TipoMovimiento = "ANULACION"
	MovAjuste    TipoMovimiento = "AJUSTE"
)

// Valido indica si es un tipo conocido.
func (t TipoMovimiento) Valido() bool {
	switch t {
	case MovInicial, MovCompra, MovVenta, MovAnulacion, MovAjuste:
		return true
	}
	return false
}

// Lote son unidades de un producto que comparten vencimiento y costo.
type Lote struct {
	ID           string
	ProductoID   string
	Vencimiento  *fecha.Fecha
	Costo        *dinero.Dinero
	CompraItemID *string
	CreatedAt    time.Time
}

// LoteStock es un lote con su stock actual.
type LoteStock struct {
	Lote
	Stock int
}

// Movimiento es un cambio de stock: positivo entra, negativo sale.
type Movimiento struct {
	ID           string
	ProductoID   string
	LoteID       *string
	Cantidad     int
	Tipo         TipoMovimiento
	ReferenciaID *string
	Nota         string
	Fecha        time.Time
}

// Asignacion indica de qué lote sale una parte de una salida de stock.
// LoteID nil significa que no había stock en lotes (stock negativo, ej. venta offline).
type Asignacion struct {
	LoteID   *string
	Cantidad int
	Costo    *dinero.Dinero
}

// AsignarFEFO reparte una salida entre lotes: primero los vigentes que vencen antes,
// después los sin fecha y por último los ya vencidos (así lo vencido sigue en las alertas
// hasta que se descarte con un ajuste). A igual fecha, el lote más antiguo.
// Lo que no alcanza a cubrirse se devuelve en una asignación sin lote.
func AsignarFEFO(lotes []LoteStock, cantidad int, hoy fecha.Fecha) []Asignacion {
	disponibles := slices.Clone(lotes)
	vencido := func(l LoteStock) bool { return l.Vencimiento != nil && l.Vencimiento.Antes(hoy) }
	slices.SortStableFunc(disponibles, func(a, b LoteStock) int {
		if va, vb := vencido(a), vencido(b); va != vb {
			if va {
				return 1
			}
			return -1
		}
		switch {
		case a.Vencimiento == nil && b.Vencimiento != nil:
			return 1
		case a.Vencimiento != nil && b.Vencimiento == nil:
			return -1
		case a.Vencimiento != nil && b.Vencimiento != nil && !a.Vencimiento.Time().Equal(b.Vencimiento.Time()):
			return a.Vencimiento.Time().Compare(b.Vencimiento.Time())
		}
		return cmp.Compare(a.CreatedAt.UnixNano(), b.CreatedAt.UnixNano())
	})

	var out []Asignacion
	resto := cantidad
	for _, l := range disponibles {
		if resto == 0 {
			break
		}
		if l.Stock <= 0 {
			continue
		}
		n := min(l.Stock, resto)
		out = append(out, Asignacion{LoteID: &l.ID, Cantidad: n, Costo: l.Costo})
		resto -= n
	}
	if resto > 0 {
		out = append(out, Asignacion{Cantidad: resto})
	}
	return out
}

// EstadoVencimiento de un lote.
type EstadoVencimiento string

const (
	Vencido   EstadoVencimiento = "VENCIDO"
	PorVencer EstadoVencimiento = "POR_VENCER"
	Vigente   EstadoVencimiento = "VIGENTE"
	SinFecha  EstadoVencimiento = "SIN_FECHA"
)

// Clasificar devuelve el estado y los días que faltan (negativo si ya venció).
func Clasificar(venc *fecha.Fecha, hoy fecha.Fecha, diasAlerta int) (EstadoVencimiento, *int) {
	if venc == nil || venc.IsZero() {
		return SinFecha, nil
	}
	dias := hoy.DiasHasta(*venc)
	switch {
	case dias < 0:
		return Vencido, &dias
	case dias < diasAlerta:
		return PorVencer, &dias
	default:
		return Vigente, &dias
	}
}

// Límites del aviso de vencimiento, en días.
const (
	DiasAlertaMinimo = 1
	DiasAlertaMaximo = 365
)

// Errores del inventario.
var (
	ErrLoteNoEncontrado = errores.NoEncontrado("lote_no_encontrado", "El lote no existe")
	ErrComboSinStock    = errores.Invalido("combo_sin_stock",
		"Los combos no tienen stock propio: se mueve el stock de sus productos")
)

// ErrStockInsuficiente informa cuánto hay disponible.
func ErrStockInsuficiente(disponible int) error {
	return errores.Conflicto("stock_insuficiente",
		"No hay stock suficiente (disponible: "+strconv.Itoa(disponible)+")")
}

// ValidarAjuste revisa un ajuste manual de stock.
func ValidarAjuste(cantidad int, motivo string) error {
	var v errores.Validacion
	v.Si(cantidad == 0, "cantidad", "La cantidad no puede ser cero")
	m := strings.TrimSpace(motivo)
	v.Si(m == "", "motivo", "Contá el motivo del ajuste (ej. rotura, vencido, conteo)")
	v.Si(utf8.RuneCountInString(m) > 255, "motivo", "Máximo 255 caracteres")
	return v.Err()
}

// ValidarDiasAlerta revisa la configuración del aviso de vencimiento.
func ValidarDiasAlerta(dias int) error {
	var v errores.Validacion
	v.Si(dias < DiasAlertaMinimo || dias > DiasAlertaMaximo, "diasAlertaVencimiento", "Debe estar entre 1 y 365 días")
	return v.Err()
}
