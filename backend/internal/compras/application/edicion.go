package application

import (
	"context"
	"strings"
	"time"

	"magui/internal/compras/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
)

// FechaEditada es la fecha pedida: un día (AAAA-MM-DD) o un instante; exactamente uno.
type FechaEditada struct {
	Dia      *fecha.Fecha
	Instante *time.Time
}

// ItemEdicionCmd es un renglón del pedido de edición. Se define completo para US-001b/c;
// en US-001a enviar renglones se rechaza.
type ItemEdicionCmd struct {
	ID               string // "" = renglón nuevo (US-001c)
	ProductoID       string
	Cantidad         int
	CostoUnitario    dinero.Dinero
	PrecioVentaNuevo *dinero.Dinero
	Vencimiento      *fecha.Fecha
}

// ComandoEditarCompra edita una compra (US-001a: cabecera). Items nil = renglones sin cambios.
type ComandoEditarCompra struct {
	ID        string
	Version   int
	Fecha     FechaEditada
	EmpresaID string // "" = sin empresa
	Notas     string
	Origen    string // "" = WEB
	Motivo    string
	Items     *[]ItemEdicionCmd
}

// EditarCompra corrige fecha, empresa y notas de una compra (US-001a). La fila se lee con
// bloqueo, se compara la versión y se graba cabecera + auditoría en una sola transacción.
// Una edición sin cambios devuelve el detalle con la misma versión y no escribe nada.
func (s *Servicio) EditarCompra(ctx context.Context, cmd ComandoEditarCompra) (CompraDetalle, error) {
	ahora := s.reloj.Ahora()
	cmd.EmpresaID = strings.TrimSpace(cmd.EmpresaID)
	cab := domain.CambiosCabecera{
		Version: cmd.Version, Dia: cmd.Fecha.Dia, Instante: cmd.Fecha.Instante, EmpresaID: cmd.EmpresaID,
		Notas: cmd.Notas, Origen: cmd.Origen, Motivo: cmd.Motivo, ConRenglones: cmd.Items != nil,
	}
	if err := domain.ValidarEdicion(cab, ahora); err != nil {
		return CompraDetalle{}, err
	}

	err := s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		actual, err := s.repo.ObtenerParaEditar(ctx, cmd.ID)
		if err != nil {
			return err
		}
		if actual.Version != cmd.Version {
			return domain.ErrVersionDesactualizada
		}
		if e := cab.EmpresaID; e != "" && (actual.EmpresaID == nil || *actual.EmpresaID != e) {
			if cab.EmpresaNombre, err = s.catalogo.NombreEmpresa(ctx, e); err != nil {
				return err
			}
		}
		plan, err := domain.PlanificarEdicion(actual, cab, ahora)
		if err != nil || plan.SinCambios {
			return err
		}
		if err := s.repo.ActualizarCompra(ctx, plan.Compra, actual.Version); err != nil {
			return err
		}
		return s.repo.GuardarEdicion(ctx, domain.Edicion{ID: s.ids.Nuevo(), CompraID: actual.ID, Fecha: ahora,
			Origen: plan.Origen, Motivo: plan.Motivo, Version: plan.Compra.Version, Cambios: plan.Cambios})
	})
	if err != nil {
		return CompraDetalle{}, err
	}
	return s.ObtenerCompra(ctx, cmd.ID)
}
