package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"magui/internal/caja/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
)

type repoMem struct{ m map[string]domain.Gasto }

func (r *repoMem) Guardar(_ context.Context, g domain.Gasto) error { r.m[g.ID] = g; return nil }
func (r *repoMem) Obtener(_ context.Context, id string) (domain.Gasto, error) {
	g, ok := r.m[id]
	if !ok {
		return g, domain.ErrGastoNoEncontrado
	}
	return g, nil
}
func (r *repoMem) MarcarAnulado(_ context.Context, g domain.Gasto) error { r.m[g.ID] = g; return nil }
func (r *repoMem) Listar(_ context.Context, f Filtro, _ paginacion.Solicitud) ([]domain.Gasto, int, dinero.Dinero, error) {
	var out []domain.Gasto
	suma := dinero.Cero
	for _, g := range r.m {
		if g.AnuladoEn != nil && !f.IncluirAnulados {
			continue
		}
		out = append(out, g)
		suma = suma.Sumar(g.Monto)
	}
	return out, len(out), suma, nil
}
func (r *repoMem) CategoriasUsadas(context.Context) ([]string, error) {
	return []string{"servicios", "Peluquería"}, nil
}

type idsSec struct{ n int }

func (g *idsSec) Nuevo() string { g.n++; return fmt.Sprintf("g-%d", g.n) }

var ctx = context.Background()
var ahora = time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)

func entorno() (*Servicio, *repoMem) {
	repo := &repoMem{m: map[string]domain.Gasto{}}
	return NuevoServicio(repo, &idsSec{}, reloj.Fijo{T: ahora}), repo
}

func TestRegistrarYAnularGasto(t *testing.T) {
	s, _ := entorno()
	g, err := s.RegistrarGasto(ctx, ComandoGasto{Monto: dinero.MustParse("15000"), Motivo: "  Boleta   de luz ", Categoria: "Servicios"})
	if err != nil {
		t.Fatal(err)
	}
	if g.Motivo != "Boleta de luz" || g.Metodo != domain.Efectivo || !g.Fecha.Equal(ahora) || *g.Categoria != "Servicios" {
		t.Errorf("gasto = %+v", g)
	}
	_, _ = s.RegistrarGasto(ctx, ComandoGasto{Monto: dinero.MustParse("5000"), Motivo: "Remis", Metodo: domain.Transferencia})
	l, _ := s.ListarGastos(ctx, Filtro{}, paginacion.Nueva(1, 10))
	if l.Total != 2 || l.TotalMonto.String() != "20000.00" {
		t.Errorf("listado = %+v", l)
	}

	an, err := s.AnularGasto(ctx, g.ID, "Cargado dos veces")
	if err != nil || !an.Anulado || *an.MotivoAnulacion != "Cargado dos veces" {
		t.Fatalf("anular: %+v %v", an, err)
	}
	if _, err := s.AnularGasto(ctx, g.ID, "otra"); !errors.Is(err, domain.ErrGastoYaAnulado) {
		t.Errorf("doble anulación: %v", err)
	}
	l, _ = s.ListarGastos(ctx, Filtro{}, paginacion.Nueva(1, 10))
	if l.Total != 1 || l.TotalMonto.String() != "5000.00" {
		t.Errorf("sin anulados = %+v", l)
	}
}

func TestValidacionesGasto(t *testing.T) {
	s, _ := entorno()
	futuro := ahora.Add(48 * time.Hour)
	_, err := s.RegistrarGasto(ctx, ComandoGasto{Monto: dinero.Cero, Motivo: "", Fecha: &futuro})
	e, _ := errores.Como(err)
	if e == nil || e.Campos["monto"] == "" || e.Campos["motivo"] == "" || e.Campos["fecha"] == "" {
		t.Errorf("validación = %v", err)
	}
	if _, err := s.AnularGasto(ctx, "nope", "x"); !errors.Is(err, domain.ErrGastoNoEncontrado) {
		t.Errorf("inexistente: %v", err)
	}
}

func TestCategorias(t *testing.T) {
	s, _ := entorno()
	cats, _ := s.Categorias(ctx)
	if len(cats) != len(domain.CategoriasSugeridas)+1 || cats[len(cats)-1] != "Peluquería" {
		t.Errorf("categorías = %v (servicios no debería repetirse)", cats)
	}
	if m, ok := domain.ParseMetodo(""); !ok || m != domain.Efectivo {
		t.Error("método vacío = efectivo")
	}
}
