// Package domain contiene las reglas de una venta: renglones, pagos combinados y anulación.
package domain

import (
	"fmt"
	"strings"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
)

// MetodoPago con el que se cobra.
type MetodoPago string

const (
	Efectivo      MetodoPago = "EFECTIVO"
	Transferencia MetodoPago = "TRANSFERENCIA"
	QR            MetodoPago = "QR"
	Tarjeta       MetodoPago = "TARJETA"
)

// ParseMetodo normaliza y valida un método de pago.
func ParseMetodo(s string) (MetodoPago, bool) {
	m := MetodoPago(strings.ToUpper(strings.TrimSpace(s)))
	switch m {
	case Efectivo, Transferencia, QR, Tarjeta:
		return m, true
	}
	return "", false
}

// PagoIngresado es lo que carga la cajera. En efectivo, Monto es lo que entregó el cliente
// (puede ser más que lo que falta: la diferencia es el vuelto).
type PagoIngresado struct {
	Metodo MetodoPago
	Monto  dinero.Dinero
}

// Pago es un pago aplicado a la venta.
type Pago struct {
	ID            string
	Metodo        MetodoPago
	Monto         dinero.Dinero  // lo que se aplica al total
	MontoRecibido *dinero.Dinero // solo efectivo
	Vuelto        *dinero.Dinero // solo efectivo
}

// ResumenPagos sirve para mostrar "cuánto falta" mientras se carga el cobro.
type ResumenPagos struct {
	Total  dinero.Dinero
	Pagado dinero.Dinero // aplicado al total
	Falta  dinero.Dinero
	Vuelto dinero.Dinero
}

// CalcularPagos aplica los pagos al total. Reglas:
//   - todos los montos deben ser mayores a cero;
//   - los pagos que no son en efectivo no pueden superar el total (no hay vuelto con tarjeta);
//   - puede haber un solo pago en efectivo: cubre lo que falta y el excedente es vuelto;
//   - la suma aplicada debe ser exactamente el total.
func CalcularPagos(total dinero.Dinero, ingresados []PagoIngresado, nuevoID func() string) ([]Pago, ResumenPagos, error) {
	res := ResumenPagos{Total: total}
	if len(ingresados) == 0 {
		res.Falta = total
		return nil, res, ErrPagosInsuficientes(total)
	}
	var v errores.Validacion
	var noEfectivo dinero.Dinero
	efectivo := -1
	for i, p := range ingresados {
		campo := fmt.Sprintf("pagos[%d]", i)
		if _, ok := ParseMetodo(string(p.Metodo)); !ok {
			v.Agregar(campo+".metodo", "Use EFECTIVO, TRANSFERENCIA, QR o TARJETA")
			continue
		}
		v.Si(!p.Monto.EsPositivo(), campo+".monto", "El monto debe ser mayor a cero")
		if p.Metodo == Efectivo {
			if efectivo >= 0 {
				v.Agregar(campo+".metodo", "Cargá el efectivo en un solo pago")
			}
			efectivo = i
			continue
		}
		noEfectivo = noEfectivo.Sumar(p.Monto)
	}
	if err := v.Err(); err != nil {
		return nil, res, err
	}
	if noEfectivo.MayorQue(total) {
		return nil, res, errores.Conflicto("pagos_excedidos", fmt.Sprintf(
			"Los pagos con transferencia, QR o tarjeta ($%s) superan el total ($%s)", noEfectivo, total))
	}

	pagos := make([]Pago, 0, len(ingresados))
	for i, p := range ingresados {
		if i == efectivo {
			continue
		}
		pagos = append(pagos, Pago{ID: nuevoID(), Metodo: p.Metodo, Monto: p.Monto})
	}
	res.Pagado = noEfectivo
	resta := total.Restar(noEfectivo)

	if efectivo >= 0 {
		recibido := ingresados[efectivo].Monto
		aplicado := recibido
		if recibido.MayorQue(resta) {
			aplicado = resta
		}
		vuelto := recibido.Restar(aplicado)
		res.Pagado = res.Pagado.Sumar(aplicado)
		res.Vuelto = vuelto
		if aplicado.EsPositivo() {
			pagos = append(pagos, Pago{ID: nuevoID(), Metodo: Efectivo, Monto: aplicado, MontoRecibido: &recibido, Vuelto: &vuelto})
		} else {
			// Ya estaba todo cubierto con otros medios: el efectivo sobra por completo.
			return nil, res, errores.Conflicto("pagos_excedidos", "El total ya está cubierto: quitá el pago en efectivo")
		}
	}

	res.Falta = total.Restar(res.Pagado)
	if res.Falta.EsPositivo() {
		return nil, res, ErrPagosInsuficientes(res.Falta)
	}
	return pagos, res, nil
}

// ErrPagosInsuficientes informa cuánto falta cobrar.
func ErrPagosInsuficientes(falta dinero.Dinero) error {
	return errores.Conflicto("pagos_insuficientes", fmt.Sprintf("Faltan $%s para completar el cobro", falta))
}
