package application

import (
	"context"
	"strings"
	"time"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/id"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/kernel/tx"
	"magui/internal/ventas/domain"
)

type Servicio struct {
	repo       Repositorio
	catalogo   Catalogo
	inventario Inventario
	tx         tx.Transactor
	ids        id.Generador
	reloj      reloj.Reloj
}

type Dependencias struct {
	Repo       Repositorio
	Catalogo   Catalogo
	Inventario Inventario
	Tx         tx.Transactor
	IDs        id.Generador
	Reloj      reloj.Reloj
}

func NuevoServicio(d Dependencias) *Servicio {
	return &Servicio{repo: d.Repo, catalogo: d.Catalogo, inventario: d.Inventario, tx: d.Tx, ids: d.IDs, reloj: d.Reloj}
}

// RegistrarVenta cobra una venta: toma los precios actuales del catálogo, valida que los pagos
// cubran exactamente el total, descuenta stock (en combos, de sus componentes) y guarda
// el costo de lo vendido para calcular la ganancia. Todo o nada.
func (s *Servicio) RegistrarVenta(ctx context.Context, cmd ComandoVenta) (VentaDetalle, error) {
	if cmd.Origen == "" {
		cmd.Origen = domain.OrigenWeb
	}
	var ventaID string
	err := s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		prods, err := s.catalogo.Productos(ctx, idsDe(cmd.Items))
		if err != nil {
			return err
		}
		lineas := make([]domain.LineaVenta, len(cmd.Items))
		for i, l := range cmd.Items {
			p, ok := prods[l.ProductoID]
			if !ok {
				return domain.ErrProductoNoEncontrado
			}
			if !p.Activo {
				return domain.ErrProductoInactivo(p.Nombre)
			}
			lineas[i] = domain.LineaVenta{ProductoID: p.ID, Cantidad: l.Cantidad, Precio: p.Precio}
		}
		ventaID = s.ids.Nuevo()
		ahora := s.reloj.Ahora()
		return s.guardarVenta(ctx, ventaID, lineas, cmd.Pagos, cmd.Origen, ahora, ahora, prods, cmd.PermitirSinStock)
	})
	if err != nil {
		return VentaDetalle{}, err
	}
	return s.repo.Detalle(ctx, ventaID)
}

// guardarVenta arma la venta, descuenta el stock (guardando el costo de lo vendido) y la guarda.
// Debe llamarse dentro de una transacción.
func (s *Servicio) guardarVenta(ctx context.Context, id string, lineas []domain.LineaVenta, pagos []domain.PagoIngresado,
	origen domain.Origen, fecha, creada time.Time, prods map[string]ProductoVenta, permitirSinStock bool) error {

	venta, _, err := domain.NuevaVenta(id, lineas, pagos, origen, fecha, s.ids.Nuevo)
	if err != nil {
		return err
	}
	venta.CreatedAt = creada

	componentes, err := s.datosComponentes(ctx, prods)
	if err != nil {
		return err
	}
	for i, it := range venta.Items {
		costo, err := s.descontarRenglon(ctx, venta.ID, it, prods[it.ProductoID], componentes, permitirSinStock)
		if err != nil {
			return err
		}
		venta.Items[i].CostoTotal = costo
	}
	return s.repo.Guardar(ctx, venta)
}

// datosComponentes trae nombre y costo de los productos que forman los combos vendidos.
func (s *Servicio) datosComponentes(ctx context.Context, prods map[string]ProductoVenta) (map[string]ProductoVenta, error) {
	var ids []string
	for _, p := range prods {
		for _, c := range p.Componentes {
			ids = append(ids, c.ProductoID)
		}
	}
	if len(ids) == 0 {
		return map[string]ProductoVenta{}, nil
	}
	return s.catalogo.Productos(ctx, ids)
}

// descontarRenglon saca el stock del renglón y devuelve su costo total (nil si no se conoce).
func (s *Servicio) descontarRenglon(ctx context.Context, ventaID string, it domain.Item, p ProductoVenta,
	componentes map[string]ProductoVenta, permitirSinStock bool) (*dinero.Dinero, error) {

	if !p.Combo {
		return s.descontar(ctx, ventaID, p, it.Cantidad, permitirSinStock)
	}
	total := dinero.Cero
	conocido := true
	for _, c := range p.Componentes {
		comp, ok := componentes[c.ProductoID]
		if !ok {
			return nil, domain.ErrProductoNoEncontrado
		}
		costo, err := s.descontar(ctx, ventaID, comp, c.Cantidad*it.Cantidad, permitirSinStock)
		if err != nil {
			return nil, err
		}
		if costo == nil {
			conocido = false
			continue
		}
		total = total.Sumar(*costo)
	}
	if !conocido {
		return nil, nil
	}
	return &total, nil
}

func (s *Servicio) descontar(ctx context.Context, ventaID string, p ProductoVenta, cantidad int, permitirSinStock bool) (*dinero.Dinero, error) {
	partes, err := s.inventario.Descontar(ctx, SalidaVenta{ProductoID: p.ID, Cantidad: cantidad, VentaID: ventaID, PermitirSinStock: permitirSinStock})
	if err != nil {
		if e, ok := errores.Como(err); ok && e.Codigo == "stock_insuficiente" {
			return nil, errores.Conflicto("stock_insuficiente", p.Nombre+": "+e.Mensaje)
		}
		return nil, err
	}
	total := dinero.Cero
	for _, parte := range partes {
		costo := parte.Costo
		if costo == nil {
			costo = p.CostoUltimo // sin lote o lote sin costo: se usa el último costo conocido
		}
		if costo == nil {
			return nil, nil
		}
		total = total.Sumar(costo.Multiplicar(parte.Cantidad))
	}
	return &total, nil
}

// AnularVenta anula una venta y devuelve el stock a los lotes de donde salió.
func (s *Servicio) AnularVenta(ctx context.Context, id, motivo string) (VentaDetalle, error) {
	err := s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		v, err := s.repo.Obtener(ctx, id)
		if err != nil {
			return err
		}
		if err := v.Anular(motivo, s.reloj.Ahora()); err != nil {
			return err
		}
		if err := s.repo.MarcarAnulada(ctx, v); err != nil {
			return err
		}
		return s.inventario.Reponer(ctx, v.ID, "Anulación de venta: "+strings.TrimSpace(motivo))
	})
	if err != nil {
		return VentaDetalle{}, err
	}
	return s.repo.Detalle(ctx, id)
}

func (s *Servicio) ObtenerVenta(ctx context.Context, id string) (VentaDetalle, error) {
	return s.repo.Detalle(ctx, id)
}

// ListarVentas devuelve las ventas, de la más reciente a la más vieja.
func (s *Servicio) ListarVentas(ctx context.Context, f Filtro, p paginacion.Solicitud) (paginacion.Pagina[VentaResumen], error) {
	lista, total, err := s.repo.Listar(ctx, f, p)
	if err != nil {
		return paginacion.Pagina[VentaResumen]{}, err
	}
	return paginacion.NuevaPagina(lista, p, total), nil
}

func idsDe(items []LineaCmd) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ProductoID
	}
	return out
}
