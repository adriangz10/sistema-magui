// Package application arma los reportes a partir de consultas de solo lectura.
package application

import (
	"context"
	"time"

	"magui/internal/estadisticas/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/reloj"
)

// --- Puerto ---

type TotalesVentas struct {
	Cantidad          int
	Total             dinero.Dinero
	Costo             dinero.Dinero
	Anuladas          int
	RenglonesSinCosto int
}

type MontoPorMetodo struct {
	Metodo   string
	Monto    dinero.Dinero
	Cantidad int // ventas que usaron el método
}

type TotalesGastos struct {
	Cantidad int
	Total    dinero.Dinero
	Efectivo dinero.Dinero
}

type FilaDia struct {
	Fecha    fecha.Fecha
	Cantidad int
	Total    dinero.Dinero
	Ganancia dinero.Dinero
}

type FilaHora struct {
	Hora     int
	Cantidad int
	Total    dinero.Dinero
}

// Consultas resuelve los números contra la base. Los rangos son [desde, hasta).
// offset es el desfase horario para agrupar por día u hora local (ej. "-03:00").
type Consultas interface {
	Ventas(ctx context.Context, desde, hasta time.Time) (TotalesVentas, error)
	VentasPorMetodo(ctx context.Context, desde, hasta time.Time) ([]MontoPorMetodo, error)
	Compras(ctx context.Context, desde, hasta time.Time) (cantidad int, total dinero.Dinero, err error)
	Gastos(ctx context.Context, desde, hasta time.Time) (TotalesGastos, error)
	VentasPorDia(ctx context.Context, desde, hasta time.Time, offset string) ([]FilaDia, error)
	VentasPorHora(ctx context.Context, desde, hasta time.Time, offset string) ([]FilaHora, error)
	MasVendidos(ctx context.Context, desde, hasta time.Time, limite int, orden OrdenRanking) ([]ProductoVendido, error)
}

// --- Modelos ---

type PeriodoDTO struct {
	Tipo  domain.TipoPeriodo `json:"tipo"`
	Desde time.Time          `json:"desde"`
	Hasta time.Time          `json:"hasta"` // exclusivo
}

type VentasDTO struct {
	Cantidad          int           `json:"cantidad"`
	Total             dinero.Dinero `json:"total"`
	Costo             dinero.Dinero `json:"costo"`
	Ganancia          dinero.Dinero `json:"ganancia"`
	Margen            float64       `json:"margen"` // % de ganancia sobre lo vendido
	TicketPromedio    dinero.Dinero `json:"ticketPromedio"`
	Anuladas          int           `json:"anuladas"`
	RenglonesSinCosto int           `json:"renglonesSinCosto"` // si es > 0, la ganancia es aproximada
}

type MetodoDTO struct {
	Metodo     string        `json:"metodo"`
	Monto      dinero.Dinero `json:"monto"`
	Cantidad   int           `json:"cantidad"`
	Porcentaje float64       `json:"porcentaje"`
}

type TotalDTO struct {
	Cantidad int           `json:"cantidad"`
	Total    dinero.Dinero `json:"total"`
}

type ComparacionDTO struct {
	Total             dinero.Dinero `json:"total"`
	Ganancia          dinero.Dinero `json:"ganancia"`
	VariacionTotal    *float64      `json:"variacionTotal"` // % respecto del período anterior
	VariacionGanancia *float64      `json:"variacionGanancia"`
}

// Resumen es el tablero principal.
type Resumen struct {
	Periodo   PeriodoDTO  `json:"periodo"`
	Ventas    VentasDTO   `json:"ventas"`
	PorMetodo []MetodoDTO `json:"porMetodo"`
	Compras   TotalDTO    `json:"compras"`
	Gastos    TotalDTO    `json:"gastos"`
	// GananciaNeta = ganancia de las ventas − gastos de caja.
	GananciaNeta dinero.Dinero `json:"gananciaNeta"`
	// EfectivoNeto = ventas cobradas en efectivo − gastos pagados en efectivo (lo que debería haber en caja).
	EfectivoNeto dinero.Dinero  `json:"efectivoNeto"`
	Anterior     ComparacionDTO `json:"anterior"`
}

type VentaDiaDTO struct {
	Fecha    fecha.Fecha   `json:"fecha"`
	Cantidad int           `json:"cantidad"`
	Total    dinero.Dinero `json:"total"`
	Ganancia dinero.Dinero `json:"ganancia"`
}

type VentaHoraDTO struct {
	Hora     int           `json:"hora"`
	Cantidad int           `json:"cantidad"`
	Total    dinero.Dinero `json:"total"`
}

// OrdenRanking del ranking de productos.
type OrdenRanking string

const (
	PorUnidades OrdenRanking = "unidades"
	PorImporte  OrdenRanking = "importe"
)

type ProductoVendido struct {
	ProductoID string         `json:"productoId"`
	SKU        string         `json:"sku"`
	Nombre     string         `json:"nombre"`
	Tipo       string         `json:"tipo"`
	Unidades   int            `json:"unidades"`
	Total      dinero.Dinero  `json:"total"`
	Ganancia   *dinero.Dinero `json:"ganancia"` // nil si no se conoce el costo de ninguna unidad
}

// ConsultaPeriodo es lo que pide la pantalla.
type ConsultaPeriodo struct {
	Tipo  domain.TipoPeriodo
	Desde *fecha.Fecha
	Hasta *fecha.Fecha
}

// Metodos de pago, en el orden en que se muestran.
var Metodos = []string{"EFECTIVO", "TRANSFERENCIA", "QR", "TARJETA"}

// --- Servicio ---

type Servicio struct {
	consultas Consultas
	reloj     reloj.Reloj
	zona      *time.Location
}

func NuevoServicio(c Consultas, r reloj.Reloj, zona *time.Location) *Servicio {
	return &Servicio{consultas: c, reloj: r, zona: zona}
}

func (s *Servicio) periodo(c ConsultaPeriodo) (domain.Periodo, error) {
	return domain.CalcularPeriodo(c.Tipo, s.reloj.Ahora(), s.zona, c.Desde, c.Hasta)
}

// offset devuelve el desfase horario de la zona en ese momento ("-03:00").
func (s *Servicio) offset(t time.Time) string { return t.In(s.zona).Format("-07:00") }

// Resumen arma el tablero del período.
func (s *Servicio) Resumen(ctx context.Context, c ConsultaPeriodo) (Resumen, error) {
	p, err := s.periodo(c)
	if err != nil {
		return Resumen{}, err
	}
	v, err := s.consultas.Ventas(ctx, p.Desde, p.Hasta)
	if err != nil {
		return Resumen{}, err
	}
	metodos, err := s.consultas.VentasPorMetodo(ctx, p.Desde, p.Hasta)
	if err != nil {
		return Resumen{}, err
	}
	nCompras, totCompras, err := s.consultas.Compras(ctx, p.Desde, p.Hasta)
	if err != nil {
		return Resumen{}, err
	}
	g, err := s.consultas.Gastos(ctx, p.Desde, p.Hasta)
	if err != nil {
		return Resumen{}, err
	}
	ant, err := s.consultas.Ventas(ctx, p.AnteriorDesde, p.AnteriorHasta)
	if err != nil {
		return Resumen{}, err
	}

	ganancia := v.Total.Restar(v.Costo)
	gananciaAnt := ant.Total.Restar(ant.Costo)
	r := Resumen{
		Periodo: PeriodoDTO{Tipo: p.Tipo, Desde: p.Desde, Hasta: p.Hasta},
		Ventas: VentasDTO{Cantidad: v.Cantidad, Total: v.Total, Costo: v.Costo, Ganancia: ganancia,
			Margen: domain.Porcentaje(ganancia, v.Total), TicketPromedio: domain.Promedio(v.Total, v.Cantidad),
			Anuladas: v.Anuladas, RenglonesSinCosto: v.RenglonesSinCosto},
		Compras:      TotalDTO{Cantidad: nCompras, Total: totCompras},
		Gastos:       TotalDTO{Cantidad: g.Cantidad, Total: g.Total},
		GananciaNeta: ganancia.Restar(g.Total),
		Anterior: ComparacionDTO{Total: ant.Total, Ganancia: gananciaAnt,
			VariacionTotal: domain.Variacion(v.Total, ant.Total), VariacionGanancia: domain.Variacion(ganancia, gananciaAnt)},
	}

	porMetodo := map[string]MontoPorMetodo{}
	for _, m := range metodos {
		porMetodo[m.Metodo] = m
	}
	for _, nombre := range Metodos {
		m := porMetodo[nombre]
		r.PorMetodo = append(r.PorMetodo, MetodoDTO{Metodo: nombre, Monto: m.Monto, Cantidad: m.Cantidad,
			Porcentaje: domain.Porcentaje(m.Monto, v.Total)})
	}
	r.EfectivoNeto = porMetodo["EFECTIVO"].Monto.Restar(g.Efectivo)
	return r, nil
}

// VentasPorDia devuelve una fila por cada día del período (con ceros donde no hubo ventas).
func (s *Servicio) VentasPorDia(ctx context.Context, c ConsultaPeriodo) ([]VentaDiaDTO, error) {
	p, err := s.periodo(c)
	if err != nil {
		return nil, err
	}
	if p.Tipo == domain.Anio {
		var v errores.Validacion
		v.Agregar("periodo", "Para el año use un rango o el resumen mensual")
		return nil, v.Err()
	}
	filas, err := s.consultas.VentasPorDia(ctx, p.Desde, p.Hasta, s.offset(p.Desde))
	if err != nil {
		return nil, err
	}
	porDia := map[string]FilaDia{}
	for _, f := range filas {
		porDia[f.Fecha.String()] = f
	}
	dias := p.Dias(s.zona)
	out := make([]VentaDiaDTO, len(dias))
	for i, d := range dias {
		f := porDia[d.String()]
		out[i] = VentaDiaDTO{Fecha: d, Cantidad: f.Cantidad, Total: f.Total, Ganancia: f.Ganancia}
	}
	return out, nil
}

// VentasPorHora devuelve las 24 horas del día con lo vendido en cada una.
func (s *Servicio) VentasPorHora(ctx context.Context, c ConsultaPeriodo) ([]VentaHoraDTO, error) {
	p, err := s.periodo(c)
	if err != nil {
		return nil, err
	}
	filas, err := s.consultas.VentasPorHora(ctx, p.Desde, p.Hasta, s.offset(p.Desde))
	if err != nil {
		return nil, err
	}
	out := make([]VentaHoraDTO, 24)
	for h := range out {
		out[h].Hora = h
	}
	for _, f := range filas {
		if f.Hora >= 0 && f.Hora < 24 {
			out[f.Hora] = VentaHoraDTO{Hora: f.Hora, Cantidad: f.Cantidad, Total: f.Total}
		}
	}
	return out, nil
}

// LimiteRanking por defecto y máximo.
const (
	LimiteRankingPorDefecto = 10
	LimiteRankingMaximo     = 50
)

// MasVendidos devuelve el ranking de productos del período.
func (s *Servicio) MasVendidos(ctx context.Context, c ConsultaPeriodo, limite int, orden OrdenRanking) ([]ProductoVendido, error) {
	p, err := s.periodo(c)
	if err != nil {
		return nil, err
	}
	if limite <= 0 {
		limite = LimiteRankingPorDefecto
	}
	limite = min(limite, LimiteRankingMaximo)
	switch orden {
	case "":
		orden = PorUnidades
	case PorUnidades, PorImporte:
	default:
		var v errores.Validacion
		v.Agregar("orden", "Use unidades o importe")
		return nil, v.Err()
	}
	lista, err := s.consultas.MasVendidos(ctx, p.Desde, p.Hasta, limite, orden)
	if lista == nil {
		lista = []ProductoVendido{}
	}
	return lista, err
}
