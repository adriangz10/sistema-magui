package application

import (
	"context"
	"strings"
	"time"

	"magui/internal/compras/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/reloj"
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

// ComandoEditarCompra edita una compra: cabecera (US-001a) y renglones existentes (US-001b).
// Items nil = renglones sin cambios; si viene, es la lista completa (reemplazo) y un Vencimiento nil
// deja el lote sin vencimiento.
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

// EditarCompra corrige una compra en una sola transacción: lee la fila con bloqueo, compara la
// versión, bloquea los lotes de los renglones modificados, verifica el consumo (409 compra_consumida
// con todos los rechazados), escribe cabecera, renglones, lotes, movimientos EDICION_COMPRA, último
// costo/precio del catálogo y la auditoría. Una edición sin cambios devuelve el detalle con la misma
// versión y no escribe nada.
func (s *Servicio) EditarCompra(ctx context.Context, cmd ComandoEditarCompra) (CompraDetalle, error) {
	ahora := s.reloj.Ahora()
	cmd.EmpresaID = strings.TrimSpace(cmd.EmpresaID)
	cab := domain.CambiosCabecera{
		Version: cmd.Version, Dia: cmd.Fecha.Dia, Instante: cmd.Fecha.Instante, EmpresaID: cmd.EmpresaID,
		Notas: cmd.Notas, Origen: cmd.Origen, Motivo: cmd.Motivo,
	}
	if cmd.Items != nil {
		rs := make([]domain.RenglonEdicion, len(*cmd.Items))
		for i, it := range *cmd.Items {
			rs[i] = domain.RenglonEdicion{ID: it.ID, ProductoID: it.ProductoID, Cantidad: it.Cantidad,
				CostoUnitario: it.CostoUnitario, PrecioVentaNuevo: it.PrecioVentaNuevo, Vencimiento: it.Vencimiento}
		}
		cab.Renglones = &rs
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
		var prods map[string]ProductoCompra
		if cab.Renglones != nil {
			ids := make([]string, len(actual.Items))
			for i, it := range actual.Items {
				ids[i] = it.ProductoID
			}
			if prods, err = s.catalogo.Productos(ctx, ids); err != nil {
				return err
			}
			cab.NombresProductos = make(map[string]string, len(prods))
			for id, p := range prods {
				cab.NombresProductos[id] = p.Nombre
			}
		}
		plan, err := domain.PlanificarEdicion(actual, cab, ahora)
		if err != nil || plan.SinCambios {
			return err
		}
		nota := "Edición de compra del " + plan.Compra.Fecha.In(reloj.ZonaArgentina).Format("02/01/2006")

		// Lotes de los renglones modificados, bloqueados antes de decidir (ADR 0002).
		lotes, err := s.bloquearYVerificar(ctx, plan)
		if err != nil {
			return err
		}

		if err := s.repo.ActualizarCompra(ctx, plan.Compra, actual.Version); err != nil {
			return err
		}
		if len(plan.Modificados) > 0 {
			items := make([]domain.Item, len(plan.Modificados))
			for i, m := range plan.Modificados {
				items[i] = m.Item
			}
			if err := s.repo.ActualizarItems(ctx, items); err != nil {
				return err
			}
			for _, m := range plan.Modificados {
				if m.Diferencia == 0 && !m.CambioCosto && !m.CambioVenc {
					continue
				}
				lote, ok := lotes[m.Item.ID]
				if !ok || lote.LoteID == "" {
					return domain.ErrRenglonSinLote(m.Producto)
				}
				if err := s.inventario.AjustarLoteDeRenglon(ctx, AjusteRenglon{LoteID: lote.LoteID,
					Diferencia: m.Diferencia, Costo: m.Item.CostoUnitario, Vencimiento: m.Item.Vencimiento,
					CompraID: actual.ID, Nota: nota}); err != nil {
					return err
				}
			}
			if err := s.actualizarCostos(ctx, plan, prods, nota); err != nil {
				return err
			}
		}
		return s.repo.GuardarEdicion(ctx, domain.Edicion{ID: s.ids.Nuevo(), CompraID: actual.ID, Fecha: ahora,
			Origen: plan.Origen, Motivo: plan.Motivo, Version: plan.Compra.Version, Cambios: plan.Cambios})
	})
	if err != nil {
		return CompraDetalle{}, err
	}
	return s.ObtenerCompra(ctx, cmd.ID)
}

// bloquearYVerificar toma los lotes de los renglones modificados y rechaza la edición si alguno
// quedaría por debajo de lo consumido. No escribe nada.
func (s *Servicio) bloquearYVerificar(ctx context.Context, plan domain.PlanEdicion) (map[string]domain.LoteRenglon, error) {
	if len(plan.Modificados) == 0 {
		return nil, nil
	}
	ids := make([]string, len(plan.Modificados))
	for i, m := range plan.Modificados {
		ids[i] = m.Item.ID
	}
	lotes, err := s.inventario.LotesDeRenglones(ctx, ids, true)
	if err != nil {
		return nil, err
	}
	// Antes de escribir nada: todo renglón que toca su lote debe tenerlo (invariante de datos).
	for _, m := range plan.Modificados {
		if m.Diferencia == 0 && !m.CambioCosto && !m.CambioVenc {
			continue
		}
		if l, ok := lotes[m.Item.ID]; !ok || l.LoteID == "" {
			return nil, domain.ErrRenglonSinLote(m.Producto)
		}
	}
	return lotes, domain.VerificarConsumo(plan, lotes)
}

// actualizarCostos aplica el último costo y precio de los productos afectados, solo si esta compra
// es la más reciente del producto (después de escribir los renglones). El precio solo se aplica si
// el precioVentaNuevo del renglón cambió, y nunca se revierte (D-6).
func (s *Servicio) actualizarCostos(ctx context.Context, plan domain.PlanEdicion, prods map[string]ProductoCompra, nota string) error {
	var costos []CostoProducto
	vistos := map[string]bool{}
	for _, m := range plan.Modificados {
		pid := m.Item.ProductoID
		if vistos[pid] || (!m.CambioCosto && !m.CambioPrecio) {
			continue
		}
		vistos[pid] = true
		rec, err := s.repo.CompraMasReciente(ctx, pid)
		if err != nil {
			return err
		}
		if rec == nil || rec.CompraID != plan.Compra.ID {
			continue
		}
		var precio *dinero.Dinero
		for _, o := range plan.Modificados { // el último renglón del producto cuyo precio cambió
			if o.Item.ProductoID == pid && o.CambioPrecio && o.Item.PrecioVentaNuevo != nil {
				precio = o.Item.PrecioVentaNuevo
			}
		}
		p, ok := prods[pid]
		if !ok {
			continue
		}
		if precio != nil && *precio == p.PrecioVenta {
			precio = nil
		}
		if precio == nil && p.Costo != nil && *p.Costo == rec.Costo {
			continue
		}
		costos = append(costos, CostoProducto{ProductoID: pid, Costo: rec.Costo, PrecioVenta: precio})
	}
	if len(costos) == 0 {
		return nil
	}
	return s.catalogo.RegistrarCostos(ctx, costos, nota)
}
