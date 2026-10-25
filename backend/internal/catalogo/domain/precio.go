package domain

import (
	"time"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
)

// OrigenPrecio indica qué provocó un cambio de precio.
type OrigenPrecio string

const (
	OrigenManual  OrigenPrecio = "MANUAL"
	OrigenCascada OrigenPrecio = "CASCADA"
	OrigenCompra  OrigenPrecio = "COMPRA"
)

// CambioPrecio es una entrada del historial de precios.
type CambioPrecio struct {
	ID             string
	ProductoID     string
	PrecioAnterior dinero.Dinero
	PrecioNuevo    dinero.Dinero
	Origen         OrigenPrecio
	Motivo         string
	Fecha          time.Time
}

// TipoAjuste de un cambio de precio masivo.
type TipoAjuste string

const (
	AjustePorcentaje TipoAjuste = "PORCENTAJE" // +10 % / -5 %
	AjusteMonto      TipoAjuste = "MONTO"      // +$500 / -$200
)

// Límites razonables para evitar errores de tipeo (ej. 1000 en vez de 10).
const (
	PorcentajeMinimo = -9000  // -90,00 %
	PorcentajeMaximo = 100000 // +1000,00 %
)

// AjustePrecio describe un cambio masivo de precios.
type AjustePrecio struct {
	Tipo TipoAjuste
	// Porcentaje en centésimas de punto (1050 = 10,50 %) si Tipo es PORCENTAJE.
	Porcentaje int64
	// Monto a sumar (o restar si es negativo) si Tipo es MONTO.
	Monto dinero.Dinero
	// Redondeo opcional del resultado al múltiplo más cercano (ej. $100). Cero = sin redondeo.
	Redondeo dinero.Dinero
}

// Validar revisa que el ajuste tenga sentido.
func (a AjustePrecio) Validar() error {
	var v errores.Validacion
	switch a.Tipo {
	case AjustePorcentaje:
		v.Si(a.Porcentaje == 0, "valor", "El porcentaje no puede ser cero")
		v.Si(a.Porcentaje < PorcentajeMinimo || a.Porcentaje > PorcentajeMaximo, "valor",
			"El porcentaje debe estar entre -90 y 1000")
	case AjusteMonto:
		v.Si(a.Monto.EsCero(), "valor", "El monto no puede ser cero")
	default:
		v.Agregar("tipo", "El tipo debe ser PORCENTAJE o MONTO")
	}
	v.Si(a.Redondeo.EsNegativo(), "redondeo", "El redondeo no puede ser negativo")
	return v.Err()
}

// Aplicar calcula el precio nuevo a partir del actual.
func (a AjustePrecio) Aplicar(actual dinero.Dinero) dinero.Dinero {
	var nuevo dinero.Dinero
	if a.Tipo == AjustePorcentaje {
		nuevo = actual.AplicarPorcentaje(a.Porcentaje)
	} else {
		nuevo = actual.Sumar(a.Monto)
	}
	return nuevo.RedondearA(a.Redondeo)
}
