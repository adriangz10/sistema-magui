// Package domain contiene las reglas de los gastos y retiros de caja
// (dinero que sale y que no es compra de mercadería).
package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
)

// Metodo con el que salió el dinero.
type Metodo string

const (
	Efectivo      Metodo = "EFECTIVO"
	Transferencia Metodo = "TRANSFERENCIA"
	QR            Metodo = "QR"
	Tarjeta       Metodo = "TARJETA"
)

// ParseMetodo normaliza el método (vacío = EFECTIVO, lo habitual en un retiro de caja).
func ParseMetodo(s string) (Metodo, bool) {
	m := Metodo(strings.ToUpper(strings.TrimSpace(s)))
	switch m {
	case "":
		return Efectivo, true
	case Efectivo, Transferencia, QR, Tarjeta:
		return m, true
	}
	return "", false
}

// CategoriasSugeridas se ofrecen en el formulario; se puede escribir cualquier otra.
var CategoriasSugeridas = []string{"Servicios", "Alquiler", "Transporte", "Insumos", "Impuestos", "Retiro personal", "Otros"}

// Gasto es una salida de dinero de la caja.
type Gasto struct {
	ID              string
	Fecha           time.Time
	Monto           dinero.Dinero
	Motivo          string
	Categoria       *string
	Metodo          Metodo
	AnuladoEn       *time.Time
	MotivoAnulacion *string
	CreatedAt       time.Time
}

// NuevoGasto valida y crea un gasto.
func NuevoGasto(id string, fecha time.Time, monto dinero.Dinero, motivo, categoria string, metodo Metodo, ahora time.Time) (Gasto, error) {
	motivo = strings.Join(strings.Fields(motivo), " ")
	categoria = strings.Join(strings.Fields(categoria), " ")
	var v errores.Validacion
	v.Si(!monto.EsPositivo(), "monto", "El monto debe ser mayor a cero")
	v.Si(motivo == "", "motivo", "Contá en qué se gastó")
	v.Si(utf8.RuneCountInString(motivo) > 255, "motivo", "Máximo 255 caracteres")
	v.Si(utf8.RuneCountInString(categoria) > 60, "categoria", "Máximo 60 caracteres")
	v.Si(fecha.After(ahora.Add(time.Hour)), "fecha", "La fecha no puede ser futura")
	if err := v.Err(); err != nil {
		return Gasto{}, err
	}
	g := Gasto{ID: id, Fecha: fecha.UTC(), Monto: monto, Motivo: motivo, Metodo: metodo, CreatedAt: ahora}
	if categoria != "" {
		g.Categoria = &categoria
	}
	return g, nil
}

// Anular marca el gasto como anulado (no se borra).
func (g *Gasto) Anular(motivo string, ahora time.Time) error {
	if g.AnuladoEn != nil {
		return ErrGastoYaAnulado
	}
	motivo = strings.TrimSpace(motivo)
	var v errores.Validacion
	v.Si(motivo == "", "motivo", "Contá por qué se anula")
	v.Si(utf8.RuneCountInString(motivo) > 255, "motivo", "Máximo 255 caracteres")
	if err := v.Err(); err != nil {
		return err
	}
	g.AnuladoEn, g.MotivoAnulacion = &ahora, &motivo
	return nil
}

var (
	ErrGastoNoEncontrado = errores.NoEncontrado("gasto_no_encontrado", "El gasto no existe")
	ErrGastoYaAnulado    = errores.Conflicto("gasto_ya_anulado", "El gasto ya estaba anulado")
)
