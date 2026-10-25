// Package application contiene los casos de uso de gastos de caja.
package application

import (
	"context"
	"slices"
	"strings"
	"time"

	"magui/internal/caja/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/id"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
)

// Repositorio persiste gastos.
type Repositorio interface {
	Guardar(ctx context.Context, g domain.Gasto) error
	Obtener(ctx context.Context, id string) (domain.Gasto, error)
	MarcarAnulado(ctx context.Context, g domain.Gasto) error
	// Listar devuelve la página, la cantidad total y la suma de montos de los gastos filtrados.
	Listar(ctx context.Context, f Filtro, p paginacion.Solicitud) ([]domain.Gasto, int, dinero.Dinero, error)
	CategoriasUsadas(ctx context.Context) ([]string, error)
}

// Filtro del listado.
type Filtro struct {
	Desde           *time.Time
	Hasta           *time.Time // exclusivo
	Categoria       string
	IncluirAnulados bool
}

type GastoDTO struct {
	ID              string        `json:"id"`
	Fecha           time.Time     `json:"fecha"`
	Monto           dinero.Dinero `json:"monto"`
	Motivo          string        `json:"motivo"`
	Categoria       *string       `json:"categoria"`
	Metodo          domain.Metodo `json:"metodo"`
	Anulado         bool          `json:"anulado"`
	AnuladoEn       *time.Time    `json:"anuladoEn"`
	MotivoAnulacion *string       `json:"motivoAnulacion"`
}

func dto(g domain.Gasto) GastoDTO {
	return GastoDTO{ID: g.ID, Fecha: g.Fecha, Monto: g.Monto, Motivo: g.Motivo, Categoria: g.Categoria, Metodo: g.Metodo,
		Anulado: g.AnuladoEn != nil, AnuladoEn: g.AnuladoEn, MotivoAnulacion: g.MotivoAnulacion}
}

// ListadoGastos es una página de gastos más la suma de todos los filtrados (no solo la página).
type ListadoGastos struct {
	paginacion.Pagina[GastoDTO]
	TotalMonto dinero.Dinero `json:"totalMonto"`
}

// ComandoGasto registra un gasto. Fecha nil = ahora.
type ComandoGasto struct {
	Fecha     *time.Time
	Monto     dinero.Dinero
	Motivo    string
	Categoria string
	Metodo    domain.Metodo
}

type Servicio struct {
	repo  Repositorio
	ids   id.Generador
	reloj reloj.Reloj
}

func NuevoServicio(repo Repositorio, ids id.Generador, r reloj.Reloj) *Servicio {
	return &Servicio{repo: repo, ids: ids, reloj: r}
}

func (s *Servicio) RegistrarGasto(ctx context.Context, cmd ComandoGasto) (GastoDTO, error) {
	ahora := s.reloj.Ahora()
	fecha := ahora
	if cmd.Fecha != nil {
		fecha = *cmd.Fecha
	}
	if cmd.Metodo == "" {
		cmd.Metodo = domain.Efectivo
	}
	g, err := domain.NuevoGasto(s.ids.Nuevo(), fecha, cmd.Monto, cmd.Motivo, cmd.Categoria, cmd.Metodo, ahora)
	if err != nil {
		return GastoDTO{}, err
	}
	if err := s.repo.Guardar(ctx, g); err != nil {
		return GastoDTO{}, err
	}
	return dto(g), nil
}

func (s *Servicio) AnularGasto(ctx context.Context, id, motivo string) (GastoDTO, error) {
	g, err := s.repo.Obtener(ctx, id)
	if err != nil {
		return GastoDTO{}, err
	}
	if err := g.Anular(motivo, s.reloj.Ahora()); err != nil {
		return GastoDTO{}, err
	}
	if err := s.repo.MarcarAnulado(ctx, g); err != nil {
		return GastoDTO{}, err
	}
	return dto(g), nil
}

func (s *Servicio) ListarGastos(ctx context.Context, f Filtro, p paginacion.Solicitud) (ListadoGastos, error) {
	lista, total, suma, err := s.repo.Listar(ctx, f, p)
	if err != nil {
		return ListadoGastos{}, err
	}
	out := make([]GastoDTO, len(lista))
	for i, g := range lista {
		out[i] = dto(g)
	}
	return ListadoGastos{Pagina: paginacion.NuevaPagina(out, p, total), TotalMonto: suma}, nil
}

// Categorias devuelve las sugeridas más las que ya se usaron, sin repetir.
func (s *Servicio) Categorias(ctx context.Context) ([]string, error) {
	usadas, err := s.repo.CategoriasUsadas(ctx)
	if err != nil {
		return nil, err
	}
	out := slices.Clone(domain.CategoriasSugeridas)
	for _, c := range usadas {
		if !slices.ContainsFunc(out, func(x string) bool { return strings.EqualFold(x, c) }) {
			out = append(out, c)
		}
	}
	return out, nil
}
