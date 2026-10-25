package application

import (
	"context"

	"magui/internal/catalogo/domain"
	"magui/internal/kernel/paginacion"
)

// ListarEmpresas devuelve una página de empresas.
func (s *Servicio) ListarEmpresas(ctx context.Context, f FiltroEmpresas, p paginacion.Solicitud) (paginacion.Pagina[EmpresaDTO], error) {
	lista, total, err := s.empresas.Listar(ctx, f, p)
	if err != nil {
		return paginacion.Pagina[EmpresaDTO]{}, err
	}
	out := make([]EmpresaDTO, len(lista))
	for i, e := range lista {
		out[i] = empresaDTO(e)
	}
	return paginacion.NuevaPagina(out, p, total), nil
}

// CrearEmpresa da de alta una empresa (Natura, Avon, ...).
func (s *Servicio) CrearEmpresa(ctx context.Context, nombre string) (EmpresaDTO, error) {
	e, err := domain.NuevaEmpresa(s.ids.Nuevo(), nombre, s.reloj.Ahora())
	if err != nil {
		return EmpresaDTO{}, err
	}
	if err := s.validarNombreEmpresa(ctx, e.Nombre, ""); err != nil {
		return EmpresaDTO{}, err
	}
	if err := s.empresas.Crear(ctx, e); err != nil {
		return EmpresaDTO{}, err
	}
	return empresaDTO(e), nil
}

// ActualizarEmpresa cambia nombre o estado. version es la que vio el usuario al abrir el formulario.
func (s *Servicio) ActualizarEmpresa(ctx context.Context, id, nombre string, activo bool, version int) (EmpresaDTO, error) {
	e, err := s.empresas.ObtenerPorID(ctx, id)
	if err != nil {
		return EmpresaDTO{}, err
	}
	if e.Version != version {
		return EmpresaDTO{}, domain.ErrVersionDesactualizada
	}
	if err := e.Modificar(nombre, activo, s.reloj.Ahora()); err != nil {
		return EmpresaDTO{}, err
	}
	if err := s.validarNombreEmpresa(ctx, e.Nombre, e.ID); err != nil {
		return EmpresaDTO{}, err
	}
	if err := s.empresas.Actualizar(ctx, e, version); err != nil {
		return EmpresaDTO{}, err
	}
	return empresaDTO(e), nil
}

func (s *Servicio) validarNombreEmpresa(ctx context.Context, nombre, excluirID string) error {
	existe, err := s.empresas.ExisteNombre(ctx, nombre, excluirID)
	if err != nil {
		return err
	}
	if existe {
		return domain.ErrEmpresaDuplicada
	}
	return nil
}
