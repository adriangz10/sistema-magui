package application

import (
	"context"
	"testing"
	"time"

	"magui/internal/estadisticas/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/reloj"
)

func d(s string) dinero.Dinero { return dinero.MustParse(s) }

// consultasFijas devuelve números distintos para el período actual y el anterior.
type consultasFijas struct {
	desdeActual time.Time
	offset      string
}

func (c *consultasFijas) Ventas(_ context.Context, desde, _ time.Time) (TotalesVentas, error) {
	if desde.Equal(c.desdeActual) {
		return TotalesVentas{Cantidad: 4, Total: d("100000"), Costo: d("60000"), Anuladas: 1, RenglonesSinCosto: 2}, nil
	}
	return TotalesVentas{Cantidad: 2, Total: d("80000"), Costo: d("50000")}, nil
}
func (c *consultasFijas) VentasPorMetodo(context.Context, time.Time, time.Time) ([]MontoPorMetodo, error) {
	return []MontoPorMetodo{{"EFECTIVO", d("30000"), 2}, {"QR", d("70000"), 3}}, nil
}
func (c *consultasFijas) Compras(context.Context, time.Time, time.Time) (int, dinero.Dinero, error) {
	return 1, d("200000"), nil
}
func (c *consultasFijas) Gastos(context.Context, time.Time, time.Time) (TotalesGastos, error) {
	return TotalesGastos{Cantidad: 2, Total: d("15000"), Efectivo: d("10000")}, nil
}
func (c *consultasFijas) VentasPorDia(_ context.Context, _, _ time.Time, offset string) ([]FilaDia, error) {
	c.offset = offset
	return []FilaDia{{Fecha: fecha.Nueva(2026, 10, 21), Cantidad: 2, Total: d("5000"), Ganancia: d("2000")}}, nil
}
func (c *consultasFijas) VentasPorHora(context.Context, time.Time, time.Time, string) ([]FilaHora, error) {
	return []FilaHora{{Hora: 17, Cantidad: 3, Total: d("9000")}}, nil
}
func (c *consultasFijas) MasVendidos(context.Context, time.Time, time.Time, int, OrdenRanking) ([]ProductoVendido, error) {
	return nil, nil
}

var art = time.FixedZone("ART", -3*3600)
var ahora = time.Date(2026, 10, 25, 18, 0, 0, 0, time.UTC) // domingo 15:00 ART

func entorno() (*Servicio, *consultasFijas) {
	c := &consultasFijas{desdeActual: time.Date(2026, 10, 19, 0, 0, 0, 0, art)}
	return NuevoServicio(c, reloj.Fijo{T: ahora}, art), c
}

var ctx = context.Background()

func TestResumen(t *testing.T) {
	s, _ := entorno()
	r, err := s.Resumen(ctx, ConsultaPeriodo{Tipo: domain.Semana})
	if err != nil {
		t.Fatal(err)
	}
	v := r.Ventas
	if v.Ganancia.String() != "40000.00" || v.Margen != 40 || v.TicketPromedio.String() != "25000.00" || v.Anuladas != 1 {
		t.Errorf("ventas = %+v", v)
	}
	if r.GananciaNeta.String() != "25000.00" || r.EfectivoNeto.String() != "20000.00" || r.Compras.Total.String() != "200000.00" {
		t.Errorf("neta %s efectivo %s compras %s", r.GananciaNeta, r.EfectivoNeto, r.Compras.Total)
	}
	if len(r.PorMetodo) != 4 || r.PorMetodo[0].Metodo != "EFECTIVO" || r.PorMetodo[2].Porcentaje != 70 || !r.PorMetodo[3].Monto.EsCero() {
		t.Errorf("por método = %+v", r.PorMetodo)
	}
	if *r.Anterior.VariacionTotal != 25 || *r.Anterior.VariacionGanancia != 33.33 {
		t.Errorf("comparación = %+v", r.Anterior)
	}
}

func TestVentasPorDiaCompletaCeros(t *testing.T) {
	s, c := entorno()
	dias, err := s.VentasPorDia(ctx, ConsultaPeriodo{Tipo: domain.Semana})
	if err != nil {
		t.Fatal(err)
	}
	if len(dias) != 7 || dias[0].Fecha.String() != "2026-10-19" || dias[2].Total.String() != "5000.00" || !dias[3].Total.EsCero() {
		t.Errorf("días = %+v", dias)
	}
	if c.offset != "-03:00" {
		t.Errorf("offset = %q", c.offset)
	}
	if _, err := s.VentasPorDia(ctx, ConsultaPeriodo{Tipo: domain.Anio}); err == nil {
		t.Error("año completo por día debería pedir un rango")
	}
}

func TestVentasPorHoraYRanking(t *testing.T) {
	s, _ := entorno()
	horas, _ := s.VentasPorHora(ctx, ConsultaPeriodo{Tipo: domain.Dia})
	if len(horas) != 24 || horas[17].Cantidad != 3 || horas[3].Hora != 3 {
		t.Errorf("horas = %+v", horas)
	}
	if r, err := s.MasVendidos(ctx, ConsultaPeriodo{}, 0, ""); err != nil || r == nil {
		t.Errorf("ranking vacío debería ser [] y no nil: %v %v", r, err)
	}
	if _, err := s.MasVendidos(ctx, ConsultaPeriodo{}, 5, "otro"); err == nil {
		t.Error("orden inválido")
	}
}
