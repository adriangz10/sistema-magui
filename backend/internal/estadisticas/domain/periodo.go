// Package domain contiene las reglas de los reportes: períodos y cálculos de margen y variación.
package domain

import (
	"strings"
	"time"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
)

// TipoPeriodo de un reporte.
type TipoPeriodo string

const (
	Dia    TipoPeriodo = "dia"
	Semana TipoPeriodo = "semana" // de lunes a domingo
	Mes    TipoPeriodo = "mes"
	Anio   TipoPeriodo = "anio"
	Rango  TipoPeriodo = "rango" // desde/hasta elegidos
)

// MaxDiasRango limita los rangos personalizados.
const MaxDiasRango = 366

// Periodo es un intervalo [Desde, Hasta) y el intervalo anterior equivalente para comparar.
type Periodo struct {
	Tipo          TipoPeriodo
	Desde         time.Time
	Hasta         time.Time // exclusivo (fin del período)
	AnteriorDesde time.Time
	AnteriorHasta time.Time
}

// Dias devuelve las fechas que abarca el período (para series diarias).
func (p Periodo) Dias(zona *time.Location) []fecha.Fecha {
	var out []fecha.Fecha
	fin := fecha.DelInstante(p.Hasta.Add(-time.Nanosecond), zona)
	for f := fecha.DelInstante(p.Desde, zona); !f.Despues(fin); f = f.SumarDias(1) {
		out = append(out, f)
	}
	return out
}

// CalcularPeriodo arma el período. Para día/semana/mes/año en curso, el anterior es el
// mismo tramo transcurrido del período previo (ej. "lunes a hoy" contra "lunes a mismo día"
// de la semana pasada), para que la comparación sea justa.
func CalcularPeriodo(tipo TipoPeriodo, ahora time.Time, zona *time.Location, desde, hasta *fecha.Fecha) (Periodo, error) {
	tipo = TipoPeriodo(strings.ToLower(string(tipo)))
	if tipo == "" {
		tipo = Dia
	}
	hoy := fecha.DelInstante(ahora, zona)
	p := Periodo{Tipo: tipo}
	var inicio, inicioAnterior fecha.Fecha

	switch tipo {
	case Dia:
		inicio, inicioAnterior = hoy, hoy.SumarDias(-1)
		p.Hasta = hoy.SumarDias(1).InicioEn(zona)
	case Semana:
		lunes := hoy.SumarDias(-((int(hoy.Time().Weekday()) + 6) % 7))
		inicio, inicioAnterior = lunes, lunes.SumarDias(-7)
		p.Hasta = lunes.SumarDias(7).InicioEn(zona)
	case Mes:
		t := hoy.Time()
		inicio = fecha.Nueva(t.Year(), t.Month(), 1)
		inicioAnterior = fecha.Nueva(t.Year(), t.Month()-1, 1)
		p.Hasta = fecha.Nueva(t.Year(), t.Month()+1, 1).InicioEn(zona)
	case Anio:
		inicio = fecha.Nueva(hoy.Time().Year(), 1, 1)
		inicioAnterior = fecha.Nueva(hoy.Time().Year()-1, 1, 1)
		p.Hasta = fecha.Nueva(hoy.Time().Year()+1, 1, 1).InicioEn(zona)
	case Rango:
		var v errores.Validacion
		v.Si(desde == nil || desde.IsZero(), "desde", "Elegí desde qué fecha")
		v.Si(hasta == nil || hasta.IsZero(), "hasta", "Elegí hasta qué fecha")
		if err := v.Err(); err != nil {
			return Periodo{}, err
		}
		dias := desde.DiasHasta(*hasta) + 1
		v.Si(dias < 1, "hasta", "Debe ser igual o posterior a desde")
		v.Si(dias > MaxDiasRango, "hasta", "El rango no puede superar un año")
		if err := v.Err(); err != nil {
			return Periodo{}, err
		}
		p.Desde = desde.InicioEn(zona)
		p.Hasta = hasta.SumarDias(1).InicioEn(zona)
		p.AnteriorDesde = desde.SumarDias(-dias).InicioEn(zona)
		p.AnteriorHasta = p.Desde
		return p, nil
	default:
		var v errores.Validacion
		v.Agregar("periodo", "Use dia, semana, mes, anio o rango")
		return Periodo{}, v.Err()
	}

	p.Desde = inicio.InicioEn(zona)
	p.AnteriorDesde = inicioAnterior.InicioEn(zona)
	transcurrido := ahora.Sub(p.Desde)
	p.AnteriorHasta = p.AnteriorDesde.Add(transcurrido)
	if p.AnteriorHasta.After(p.Desde) {
		p.AnteriorHasta = p.Desde
	}
	return p, nil
}

// Porcentaje devuelve parte/total en % con 2 decimales (0 si total es cero).
func Porcentaje(parte, total dinero.Dinero) float64 {
	return float64(parte.PorcentajeDe(total)) / 100
}

// Variacion devuelve el cambio porcentual de anterior a actual; nil si no hay base para comparar.
func Variacion(actual, anterior dinero.Dinero) *float64 {
	if anterior.EsCero() {
		return nil
	}
	v := float64(actual.Restar(anterior).PorcentajeDe(anterior)) / 100
	return &v
}

// Promedio divide un monto por una cantidad, redondeando al centavo.
func Promedio(total dinero.Dinero, cantidad int) dinero.Dinero {
	if cantidad == 0 {
		return dinero.Cero
	}
	return dinero.DesdeCentavos((total.Centavos()*2 + int64(cantidad)) / (2 * int64(cantidad)))
}
