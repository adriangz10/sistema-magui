package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
)

// Estado de una venta.
type Estado string

const (
	Confirmada Estado = "CONFIRMADA"
	Anulada    Estado = "ANULADA"
)

// Origen indica desde qué versión del sistema se hizo la venta.
type Origen string

const (
	OrigenWeb     Origen = "WEB"
	OrigenMovil   Origen = "MOVIL"
	OrigenDesktop Origen = "DESKTOP"
)

// ParseOrigen normaliza el origen (vacío = WEB).
func ParseOrigen(s string) (Origen, bool) {
	o := Origen(strings.ToUpper(strings.TrimSpace(s)))
	switch o {
	case "":
		return OrigenWeb, true
	case OrigenWeb, OrigenMovil, OrigenDesktop:
		return o, true
	}
	return "", false
}

// MaxItems por venta.
const MaxItems = 200

// Item es un renglón vendido. Si el producto es un combo, el stock se descuenta de sus componentes.
type Item struct {
	ID             string
	ProductoID     string
	Cantidad       int
	PrecioUnitario dinero.Dinero
	// CostoTotal del renglón según los lotes de donde salió (nil si no se conoce).
	CostoTotal *dinero.Dinero
}

// Subtotal = cantidad × precio.
func (i Item) Subtotal() dinero.Dinero { return i.PrecioUnitario.Multiplicar(i.Cantidad) }

// CostoUnitario aproximado (redondeado al centavo) para guardar en el renglón.
func (i Item) CostoUnitario() *dinero.Dinero {
	if i.CostoTotal == nil || i.Cantidad == 0 {
		return nil
	}
	c := dinero.DesdeCentavos((i.CostoTotal.Centavos()*2 + int64(i.Cantidad)) / (2 * int64(i.Cantidad)))
	return &c
}

// Venta es una venta del punto de venta.
type Venta struct {
	ID              string
	Fecha           time.Time
	Items           []Item
	Pagos           []Pago
	Total           dinero.Dinero
	Estado          Estado
	Origen          Origen
	AnuladaEn       *time.Time
	MotivoAnulacion *string
	CreatedAt       time.Time
}

// LineaVenta es un producto con la cantidad pedida y su precio actual.
type LineaVenta struct {
	ProductoID string
	Cantidad   int
	Precio     dinero.Dinero
}

// AgruparLineas suma las cantidades de un mismo producto (si se escaneó dos veces).
func AgruparLineas(lineas []LineaVenta) []LineaVenta {
	idx := map[string]int{}
	var out []LineaVenta
	for _, l := range lineas {
		if i, ok := idx[l.ProductoID]; ok {
			out[i].Cantidad += l.Cantidad
			continue
		}
		idx[l.ProductoID] = len(out)
		out = append(out, l)
	}
	return out
}

// NuevaVenta arma una venta confirmada y valida que los pagos cubran exactamente el total.
func NuevaVenta(id string, lineas []LineaVenta, pagos []PagoIngresado, origen Origen, ahora time.Time, nuevoID func() string) (Venta, ResumenPagos, error) {
	var v errores.Validacion
	v.Si(len(lineas) == 0, "items", "Agregá al menos un producto")
	v.Si(len(lineas) > MaxItems, "items", fmt.Sprintf("Máximo %d productos distintos por venta", MaxItems))
	for i, l := range lineas {
		v.Si(l.Cantidad <= 0, fmt.Sprintf("items[%d].cantidad", i), "La cantidad debe ser mayor a cero")
		v.Si(!l.Precio.EsPositivo(), fmt.Sprintf("items[%d].precio", i), "El producto no tiene precio")
	}
	if err := v.Err(); err != nil {
		return Venta{}, ResumenPagos{}, err
	}

	venta := Venta{ID: id, Fecha: ahora, Estado: Confirmada, Origen: origen, CreatedAt: ahora}
	for _, l := range AgruparLineas(lineas) {
		it := Item{ID: nuevoID(), ProductoID: l.ProductoID, Cantidad: l.Cantidad, PrecioUnitario: l.Precio}
		venta.Items = append(venta.Items, it)
		venta.Total = venta.Total.Sumar(it.Subtotal())
	}
	ps, res, err := CalcularPagos(venta.Total, pagos, nuevoID)
	if err != nil {
		return Venta{}, res, err
	}
	venta.Pagos = ps
	return venta, res, nil
}

// CostoTotal suma el costo conocido de los renglones. Completo es false si a algún
// renglón le falta el costo (la ganancia de esa venta es aproximada).
func (v Venta) CostoTotal() (total dinero.Dinero, completo bool) {
	completo = true
	for _, it := range v.Items {
		if it.CostoTotal == nil {
			completo = false
			continue
		}
		total = total.Sumar(*it.CostoTotal)
	}
	return total, completo
}

// Anular marca la venta como anulada.
func (v *Venta) Anular(motivo string, ahora time.Time) error {
	if v.Estado == Anulada {
		return ErrVentaYaAnulada
	}
	motivo = strings.TrimSpace(motivo)
	var val errores.Validacion
	val.Si(motivo == "", "motivo", "Contá por qué se anula (ej. error de carga, devolución)")
	val.Si(utf8.RuneCountInString(motivo) > 255, "motivo", "Máximo 255 caracteres")
	if err := val.Err(); err != nil {
		return err
	}
	v.Estado, v.AnuladaEn, v.MotivoAnulacion = Anulada, &ahora, &motivo
	return nil
}

// Errores de ventas.
var (
	ErrVentaNoEncontrada    = errores.NoEncontrado("venta_no_encontrada", "La venta no existe")
	ErrVentaYaAnulada       = errores.Conflicto("venta_ya_anulada", "La venta ya estaba anulada")
	ErrProductoNoEncontrado = errores.NoEncontrado("producto_no_encontrado", "Uno de los productos no existe")
)

// ErrProductoInactivo: no se venden productos dados de baja.
func ErrProductoInactivo(nombre string) error {
	return errores.Conflicto("producto_inactivo", "El producto está desactivado: "+nombre)
}
