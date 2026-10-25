// Package domain contiene las reglas de una compra de mercadería.
package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
)

// MaxItems por compra (evita cargas accidentales enormes).
const MaxItems = 300

// Item es un renglón de la compra: un producto con su cantidad y costo.
type Item struct {
	ID               string
	ProductoID       string
	Cantidad         int
	CostoUnitario    dinero.Dinero
	PrecioVentaNuevo *dinero.Dinero // nuevo precio de reventa, cargado a mano (opcional)
	Vencimiento      *fecha.Fecha
}

// Subtotal = cantidad × costo unitario.
func (i Item) Subtotal() dinero.Dinero { return i.CostoUnitario.Multiplicar(i.Cantidad) }

// Compra es la llegada de mercadería.
type Compra struct {
	ID        string
	Fecha     time.Time
	EmpresaID *string
	Notas     *string
	Items     []Item
	Total     dinero.Dinero
	CreatedAt time.Time
}

// Unidades devuelve la cantidad total de unidades compradas.
func (c Compra) Unidades() int {
	n := 0
	for _, it := range c.Items {
		n += it.Cantidad
	}
	return n
}

// NuevaCompra valida y arma la compra; el total es la suma de los subtotales.
func NuevaCompra(id string, fechaCompra time.Time, empresaID, notas string, items []Item, ahora time.Time) (Compra, error) {
	var v errores.Validacion
	v.Si(fechaCompra.After(ahora.Add(time.Hour)), "fecha", "La fecha no puede ser futura")
	notas = strings.TrimSpace(notas)
	v.Si(utf8.RuneCountInString(notas) > 500, "notas", "Máximo 500 caracteres")
	v.Si(len(items) == 0, "items", "Agregá al menos un producto")
	v.Si(len(items) > MaxItems, "items", fmt.Sprintf("Máximo %d renglones por compra", MaxItems))

	total := dinero.Cero
	for i, it := range items {
		campo := func(n string) string { return fmt.Sprintf("items[%d].%s", i, n) }
		v.Si(strings.TrimSpace(it.ProductoID) == "", campo("productoId"), "Elegí un producto")
		v.Si(it.Cantidad <= 0, campo("cantidad"), "La cantidad debe ser mayor a cero")
		v.Si(it.CostoUnitario.EsNegativo(), campo("costoUnitario"), "El costo no puede ser negativo")
		v.Si(it.PrecioVentaNuevo != nil && !it.PrecioVentaNuevo.EsPositivo(), campo("precioVentaNuevo"),
			"El precio de reventa debe ser mayor a cero")
		if it.Vencimiento != nil && it.Vencimiento.IsZero() {
			items[i].Vencimiento = nil
		}
		total = total.Sumar(it.Subtotal())
	}
	if err := v.Err(); err != nil {
		return Compra{}, err
	}

	c := Compra{ID: id, Fecha: fechaCompra.UTC(), Items: items, Total: total, CreatedAt: ahora}
	if e := strings.TrimSpace(empresaID); e != "" {
		c.EmpresaID = &e
	}
	if notas != "" {
		c.Notas = &notas
	}
	return c, nil
}

// Errores de compras.
var (
	ErrCompraNoEncontrada   = errores.NoEncontrado("compra_no_encontrada", "La compra no existe")
	ErrProductoNoEncontrado = errores.NoEncontrado("producto_no_encontrado", "Uno de los productos no existe")
)

// ErrProductoNoComprable: los combos no se compran, se compran sus componentes.
func ErrProductoNoComprable(nombre string) error {
	return errores.Invalido("producto_no_comprable", "No se puede comprar un combo: "+nombre)
}
