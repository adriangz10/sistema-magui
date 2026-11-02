// Package application contiene los casos de uso de los datos del negocio (US-003).
package application

import (
	"context"

	"magui/internal/negocio/domain"
)

// Repositorio guarda la única fila de datos del negocio.
type Repositorio interface {
	Obtener(ctx context.Context) (domain.Negocio, error)
	Guardar(ctx context.Context, n domain.Negocio) error
	// GuardarLogo reemplaza el logo (datos nil lo quita).
	GuardarLogo(ctx context.Context, datos []byte, tipo string) error
	Logo(ctx context.Context) (datos []byte, tipo string, err error)
}

type TicketDTO struct {
	Habilitado       bool    `json:"habilitado"`
	ImprimirAlCobrar bool    `json:"imprimirAlCobrar"`
	AnchoPapel       int     `json:"anchoPapel"`
	MostrarLogo      bool    `json:"mostrarLogo"`
	Datos            *string `json:"datos"`
	Pie              *string `json:"pie"`
}

// MarcaDTO es lo público: se muestra también en la pantalla de ingreso.
type MarcaDTO struct {
	Nombre      string `json:"nombre"`
	TieneLogo   bool   `json:"tieneLogo"`
	LogoVersion int    `json:"logoVersion"`
}

type NegocioDTO struct {
	MarcaDTO
	Ticket TicketDTO `json:"ticket"`
}

func dto(n domain.Negocio) NegocioDTO {
	t := n.Ticket
	return NegocioDTO{
		MarcaDTO: MarcaDTO{Nombre: n.Nombre, TieneLogo: n.TieneLogo, LogoVersion: n.LogoVersion},
		Ticket: TicketDTO{Habilitado: t.Habilitado, ImprimirAlCobrar: t.ImprimirAlCobrar, AnchoPapel: t.AnchoPapel,
			MostrarLogo: t.MostrarLogo, Datos: t.Datos, Pie: t.Pie},
	}
}

// ComandoNegocio son los datos editables (todo junto, como en el formulario).
type ComandoNegocio struct {
	Nombre string
	Ticket TicketDTO
}

type Servicio struct{ repo Repositorio }

func NuevoServicio(repo Repositorio) *Servicio { return &Servicio{repo: repo} }

func (s *Servicio) Obtener(ctx context.Context) (NegocioDTO, error) {
	n, err := s.repo.Obtener(ctx)
	if err != nil {
		return NegocioDTO{}, err
	}
	return dto(n), nil
}

func (s *Servicio) Marca(ctx context.Context) (MarcaDTO, error) {
	n, err := s.Obtener(ctx)
	return n.MarcaDTO, err
}

func texto(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (s *Servicio) Guardar(ctx context.Context, cmd ComandoNegocio) (NegocioDTO, error) {
	n, err := s.repo.Obtener(ctx)
	if err != nil {
		return NegocioDTO{}, err
	}
	t := cmd.Ticket
	if err := n.Aplicar(domain.Cambios{Nombre: cmd.Nombre, Datos: texto(t.Datos), Pie: texto(t.Pie), Ticket: domain.Ticket{
		Habilitado: t.Habilitado, ImprimirAlCobrar: t.ImprimirAlCobrar, AnchoPapel: t.AnchoPapel, MostrarLogo: t.MostrarLogo}}); err != nil {
		return NegocioDTO{}, err
	}
	if err := s.repo.Guardar(ctx, n); err != nil {
		return NegocioDTO{}, err
	}
	return dto(n), nil
}

// CambiarLogo valida y guarda la imagen. datos nil quita el logo.
func (s *Servicio) CambiarLogo(ctx context.Context, datos []byte) (MarcaDTO, error) {
	tipo := ""
	if datos != nil {
		var err error
		if tipo, err = domain.ValidarLogo(datos); err != nil {
			return MarcaDTO{}, err
		}
	}
	if err := s.repo.GuardarLogo(ctx, datos, tipo); err != nil {
		return MarcaDTO{}, err
	}
	return s.Marca(ctx)
}

func (s *Servicio) Logo(ctx context.Context) ([]byte, string, error) { return s.repo.Logo(ctx) }
