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
// versión, valida los productos de los renglones nuevos, bloquea los lotes de los renglones
// modificados y quitados, verifica el consumo (409 compra_consumida con todos los rechazados),
// escribe cabecera, renglones, lotes, movimientos (EDICION_COMPRA / COMPRA), último costo/precio
// del catálogo y la auditoría. Una edición sin cambios devuelve el detalle con la misma versión y
// no escribe nada.
func (s *Servicio) EditarCompra(ctx context.Context, cmd ComandoEditarCompra) (CompraDetalle, error) {
	ahora := s.reloj.Ahora()
	cmd.EmpresaID = strings.TrimSpace(cmd.EmpresaID)
	cab := domain.CambiosCabecera{
		Version: cmd.Version, Dia: cmd.Fecha.Dia, Instante: cmd.Fecha.Instante, EmpresaID: cmd.EmpresaID,
		Notas: cmd.Notas, Origen: cmd.Origen, Motivo: cmd.Motivo, NuevoID: s.ids.Nuevo,
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
			if prods, err = s.productosDeEdicion(ctx, actual, *cab.Renglones); err != nil {
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

		// Lotes de los renglones modificados y quitados, bloqueados antes de decidir (ADR 0002).
		lotes, err := s.bloquearYVerificar(ctx, plan)
		if err != nil {
			return err
		}

		// Qué compra era la más reciente de cada producto afectado, antes de escribir (RN-EC-19).
		cambioFecha := !plan.Compra.Fecha.Equal(actual.Fecha)
		afectados := plan.ProductosAfectados(actual.Items, cambioFecha)
		if prods, err = s.completarProductos(ctx, prods, afectados); err != nil {
			return err
		}
		antes := make(map[string]*CompraReciente, len(afectados))
		for _, pid := range afectados {
			if antes[pid], err = s.repo.CompraMasReciente(ctx, pid); err != nil {
				return err
			}
		}

		if err := s.repo.ActualizarCompra(ctx, plan.Compra, actual.Version); err != nil {
			return err
		}
		if err := s.escribirRenglones(ctx, actual.ID, plan, lotes, ahora, nota); err != nil {
			return err
		}
		if err := s.actualizarCostos(ctx, plan, afectados, antes, prods, nota); err != nil {
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

// productosDeEdicion consulta el catálogo una sola vez por los productos de los renglones actuales y
// de los nuevos, y valida los de los renglones nuevos: deben existir (404) y no ser combos (422).
func (s *Servicio) productosDeEdicion(ctx context.Context, actual domain.Compra, pedidos []domain.RenglonEdicion) (map[string]ProductoCompra, error) {
	ids := make([]string, 0, len(actual.Items)+len(pedidos))
	vistos := map[string]bool{}
	sumar := func(id string) {
		if !vistos[id] {
			vistos[id] = true
			ids = append(ids, id)
		}
	}
	for _, it := range actual.Items {
		sumar(it.ProductoID)
	}
	for _, r := range pedidos {
		if strings.TrimSpace(r.ID) == "" {
			sumar(r.ProductoID)
		}
	}
	prods, err := s.catalogo.Productos(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i, r := range pedidos {
		if strings.TrimSpace(r.ID) != "" {
			continue
		}
		p, ok := prods[r.ProductoID]
		if !ok {
			return nil, domain.ErrProductoNoEncontrado
		}
		if p.Combo {
			return nil, domain.ErrProductoNoComprable(p.Nombre)
		}
		if p.Inactivo {
			return nil, domain.ErrProductoInactivo(p.Nombre, i)
		}
	}
	return prods, nil
}

// completarProductos agrega a prods los productos afectados que todavía no se consultaron (p. ej.
// cuando solo cambió la fecha y el pedido no trae renglones).
func (s *Servicio) completarProductos(ctx context.Context, prods map[string]ProductoCompra, ids []string) (map[string]ProductoCompra, error) {
	var faltan []string
	for _, id := range ids {
		if _, ok := prods[id]; !ok {
			faltan = append(faltan, id)
		}
	}
	if len(faltan) == 0 {
		return prods, nil
	}
	mas, err := s.catalogo.Productos(ctx, faltan)
	if err != nil {
		return nil, err
	}
	out := make(map[string]ProductoCompra, len(prods)+len(mas))
	for id, p := range prods {
		out[id] = p
	}
	for id, p := range mas {
		out[id] = p
	}
	return out, nil
}

// bloquearYVerificar toma los lotes de los renglones modificados y quitados y rechaza la edición si
// alguno quedaría por debajo de lo consumido. No escribe nada.
func (s *Servicio) bloquearYVerificar(ctx context.Context, plan domain.PlanEdicion) (map[string]domain.LoteRenglon, error) {
	ids := make([]string, 0, len(plan.Modificados)+len(plan.Quitados))
	for _, m := range plan.Modificados {
		ids = append(ids, m.Item.ID)
	}
	for _, q := range plan.Quitados {
		ids = append(ids, q.Item.ID)
	}
	if len(ids) == 0 {
		return nil, nil
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
	for _, q := range plan.Quitados {
		if l, ok := lotes[q.Item.ID]; !ok || l.LoteID == "" {
			return nil, domain.ErrRenglonSinLote(q.Producto)
		}
	}
	return lotes, domain.VerificarConsumo(plan, lotes)
}

// escribirRenglones graba renglones modificados, quitados y agregados y los movimientos de stock.
func (s *Servicio) escribirRenglones(ctx context.Context, compraID string, plan domain.PlanEdicion,
	lotes map[string]domain.LoteRenglon, ahora time.Time, nota string) error {
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
			if err := s.inventario.AjustarLoteDeRenglon(ctx, AjusteRenglon{LoteID: lotes[m.Item.ID].LoteID,
				Diferencia: m.Diferencia, Costo: m.Item.CostoUnitario, Vencimiento: m.Item.Vencimiento,
				CompraID: compraID, Nota: nota}); err != nil {
				return err
			}
		}
	}
	if len(plan.Quitados) > 0 {
		ids := make([]string, len(plan.Quitados))
		for i, q := range plan.Quitados {
			ids[i] = q.Item.ID
		}
		if err := s.repo.QuitarItems(ctx, ids, ahora); err != nil {
			return err
		}
		for _, q := range plan.Quitados {
			// Solo la cantidad que trajo el renglón (P-A2); el lote y su vínculo quedan como historia.
			if err := s.inventario.AjustarLoteDeRenglon(ctx, AjusteRenglon{LoteID: lotes[q.Item.ID].LoteID,
				Diferencia: -q.Item.Cantidad, CompraID: compraID, Nota: nota, SoloCantidad: true}); err != nil {
				return err
			}
		}
	}
	if len(plan.Agregados) > 0 {
		items := make([]domain.Item, len(plan.Agregados))
		for i, a := range plan.Agregados {
			items[i] = a.Item
		}
		if err := s.repo.AgregarItems(ctx, compraID, items); err != nil {
			return err
		}
		for _, a := range plan.Agregados {
			if err := s.inventario.Ingresar(ctx, IngresoCompra{CompraID: compraID, CompraItemID: a.Item.ID,
				ProductoID: a.Item.ProductoID, Cantidad: a.Item.Cantidad, Costo: a.Item.CostoUnitario,
				Vencimiento: a.Item.Vencimiento, Nota: nota}); err != nil {
				return err
			}
		}
	}
	return nil
}

// actualizarCostos aplica el último costo y precio de los productos afectados (RN-EC-15 a RN-EC-19),
// con la compra ya escrita:
//   - si esta compra es la más reciente del producto: toma el costo de su último renglón vigente y, si
//     el precioVentaNuevo de un renglón modificado o agregado cambió y difiere del actual, el precio;
//   - si dejó de serlo (quitar el renglón o cambiar la fecha): el costo pasa al de la nueva más
//     reciente; el precio no se toca (D-6);
//   - si no queda ninguna compra del producto, el costo se mantiene.
func (s *Servicio) actualizarCostos(ctx context.Context, plan domain.PlanEdicion, afectados []string,
	antes map[string]*CompraReciente, prods map[string]ProductoCompra, nota string) error {
	var costos []CostoProducto
	directosSet := map[string]bool{} // productos con renglón modificado en costo/precio, agregado o quitado
	for _, id := range plan.ProductosAfectados(nil, false) {
		directosSet[id] = true
	}
	for _, pid := range afectados {
		rec, err := s.repo.CompraMasReciente(ctx, pid)
		if err != nil {
			return err
		}
		if rec == nil {
			continue
		}
		var precio *dinero.Dinero
		directos := directosSet
		switch {
		case rec.CompraID == plan.Compra.ID && !directos[pid] && antes[pid] != nil && antes[pid].CompraID == plan.Compra.ID:
			// Era y sigue siendo la más reciente y el producto no cambió en esta edición (p. ej. solo la fecha): no se toca el catálogo.
			continue
		case rec.CompraID == plan.Compra.ID:
			for _, o := range plan.Modificados { // el último renglón del producto cuyo precio cambió
				if o.Item.ProductoID == pid && o.CambioPrecio && o.Item.PrecioVentaNuevo != nil {
					precio = o.Item.PrecioVentaNuevo
				}
			}
			for _, a := range plan.Agregados {
				if a.Item.ProductoID == pid && a.Item.PrecioVentaNuevo != nil {
					precio = a.Item.PrecioVentaNuevo
				}
			}
		case antes[pid] != nil && antes[pid].CompraID == plan.Compra.ID:
			// Dejó de ser la más reciente: solo se recalcula el costo.
		default:
			continue
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
