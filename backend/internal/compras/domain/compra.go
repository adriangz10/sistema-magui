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

// MaxNotas es el largo máximo de las notas de una compra.
const MaxNotas = 500

// MaxCantidadItem es la cantidad máxima por renglón (la columna es INT; esto evita cargas absurdas).
const MaxCantidadItem = 1_000_000

// MaxMonto es el mayor importe que entra en las columnas DECIMAL(12,2) (costo, precio y total): 9.999.999.999,99.
var MaxMonto = dinero.DesdeCentavos(999_999_999_999)

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
	Version   int       // control optimista: sube 1 por edición efectiva
	UpdatedAt time.Time // última edición (igual a CreatedAt si nunca se editó)
	Empresa   string    // nombre de la empresa (solo lectura, para el historial de ediciones)
}

// Unidades devuelve la cantidad total de unidades compradas.
func (c Compra) Unidades() int {
	n := 0
	for _, it := range c.Items {
		n += it.Cantidad
	}
	return n
}

// validarCabecera valida lo que comparten el alta y la edición: notas y fecha no futura.
func validarCabecera(v *errores.Validacion, fechaCompra time.Time, notas string, ahora time.Time) {
	v.Si(fechaCompra.After(ahora.Add(time.Hour)), "fecha", "La fecha no puede ser futura")
	v.Si(utf8.RuneCountInString(notas) > MaxNotas, "notas", "Máximo 500 caracteres")
}

// validarItems valida los renglones y devuelve el total. Normaliza vencimientos vacíos.
func validarItems(v *errores.Validacion, items []Item) dinero.Dinero {
	v.Si(len(items) == 0, "items", "Agregá al menos un producto")
	v.Si(len(items) > MaxItems, "items", fmt.Sprintf("Máximo %d renglones por compra", MaxItems))

	total := dinero.Cero
	for i, it := range items {
		campo := func(n string) string { return fmt.Sprintf("items[%d].%s", i, n) }
		v.Si(strings.TrimSpace(it.ProductoID) == "", campo("productoId"), "Elegí un producto")
		v.Si(it.Cantidad <= 0, campo("cantidad"), "La cantidad debe ser mayor a cero")
		v.Si(it.Cantidad > MaxCantidadItem, campo("cantidad"), "La cantidad no puede superar 1.000.000 de unidades")
		v.Si(it.CostoUnitario.EsNegativo(), campo("costoUnitario"), "El costo no puede ser negativo")
		v.Si(it.CostoUnitario.MayorQue(MaxMonto), campo("costoUnitario"), "El costo es demasiado alto (máximo $9.999.999.999,99)")
		v.Si(it.PrecioVentaNuevo != nil && !it.PrecioVentaNuevo.EsPositivo(), campo("precioVentaNuevo"),
			"El precio de reventa debe ser mayor a cero")
		v.Si(it.PrecioVentaNuevo != nil && it.PrecioVentaNuevo.MayorQue(MaxMonto), campo("precioVentaNuevo"),
			"El precio de reventa es demasiado alto (máximo $9.999.999.999,99)")
		if it.Vencimiento != nil && it.Vencimiento.IsZero() {
			items[i].Vencimiento = nil
		}
		// Solo se suman renglones con valores acotados: así el total no puede desbordar.
		if it.Cantidad > 0 && it.Cantidad <= MaxCantidadItem && !it.CostoUnitario.EsNegativo() && !it.CostoUnitario.MayorQue(MaxMonto) {
			total = total.Sumar(it.Subtotal())
		}
		v.Si(total.MayorQue(MaxMonto), "items", "El total de la compra es demasiado alto (máximo $9.999.999.999,99)")
	}
	return total
}

// NuevaCompra valida y arma la compra; el total es la suma de los subtotales.
func NuevaCompra(id string, fechaCompra time.Time, empresaID, notas string, items []Item, ahora time.Time) (Compra, error) {
	var v errores.Validacion
	notas = strings.TrimSpace(notas)
	validarCabecera(&v, fechaCompra, notas, ahora)
	total := validarItems(&v, items)
	if err := v.Err(); err != nil {
		return Compra{}, err
	}

	c := Compra{ID: id, Fecha: fechaCompra.UTC(), Items: items, Total: total, CreatedAt: ahora, Version: 1, UpdatedAt: ahora}
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

	// ErrVersionDesactualizada: otra persona editó la compra después de que se leyó (US-001a, ADR 0002).
	ErrVersionDesactualizada = errores.Conflicto("version_desactualizada", "Otra persona modificó esta compra, recargá")
	// Funciones de US-001c que todavía no están disponibles.
	ErrRenglonesNuevosNoDisponibles = errores.Invalido("renglones_nuevos_no_disponibles", "Agregar renglones todavía no está disponible")
	ErrQuitarRenglonesNoDisponible  = errores.Invalido("quitar_renglones_no_disponible", "Quitar renglones todavía no está disponible")
)

// ErrCompraConsumida: se quiso bajar un renglón por debajo de lo que ya salió de su lote.
// Es un valor base: VerificarConsumo lo copia y le agrega los campos.
var ErrCompraConsumida = *errores.Conflicto("compra_consumida", "Hay unidades de esta compra que ya salieron del stock")

// ErrRenglonSinLote: el renglón no tiene lote (dato inconsistente); no es una regla de negocio
// que la usuaria pueda corregir, pero se informa con un error tipado en lugar de un 500.
func ErrRenglonSinLote(producto string) error {
	return errores.Conflicto("renglon_sin_lote", "No se encontró el lote de «"+producto+"»: no se puede editar ese renglón. Avisá a soporte.")
}

// ErrProductoNoComprable: los combos no se compran, se compran sus componentes.
func ErrProductoNoComprable(nombre string) error {
	return errores.Invalido("producto_no_comprable", "No se puede comprar un combo: "+nombre)
}
