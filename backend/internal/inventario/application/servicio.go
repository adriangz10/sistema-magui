package application

import (
	"context"
	"strings"

	"magui/internal/inventario/domain"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/id"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/kernel/tx"
)

// ErrProductoNoEncontrado se devuelve cuando el catálogo no tiene el producto.
var ErrProductoNoEncontrado = errores.NoEncontrado("producto_no_encontrado", "El producto no existe")

// Servicio reúne los casos de uso del inventario.
type Servicio struct {
	repo      Repositorio
	productos Productos
	tx        tx.Transactor
	ids       id.Generador
	reloj     reloj.Reloj
}

type Dependencias struct {
	Repo      Repositorio
	Productos Productos
	Tx        tx.Transactor
	IDs       id.Generador
	Reloj     reloj.Reloj
}

func NuevoServicio(d Dependencias) *Servicio {
	return &Servicio{repo: d.Repo, productos: d.Productos, tx: d.Tx, ids: d.IDs, reloj: d.Reloj}
}

// hoy es la fecha actual en Argentina.
func (s *Servicio) hoy() fecha.Fecha { return fecha.DelInstante(s.reloj.Ahora(), reloj.ZonaArgentina) }

// productoConStock verifica que el producto exista y no sea un combo.
func (s *Servicio) productoConStock(ctx context.Context, productoID string) (InfoProducto, error) {
	info, err := s.productos.Info(ctx, []string{productoID})
	if err != nil {
		return InfoProducto{}, err
	}
	p, ok := info[productoID]
	if !ok {
		return InfoProducto{}, ErrProductoNoEncontrado
	}
	if p.Combo {
		return InfoProducto{}, domain.ErrComboSinStock
	}
	return p, nil
}

// RegistrarIngreso crea un lote con su movimiento de entrada. Lo usan compras y ajustes.
func (s *Servicio) RegistrarIngreso(ctx context.Context, in Ingreso) (loteID string, err error) {
	var v errores.Validacion
	v.Si(in.Cantidad <= 0, "cantidad", "La cantidad debe ser mayor a cero")
	v.Si(in.Costo != nil && in.Costo.EsNegativo(), "costo", "El costo no puede ser negativo")
	if err := v.Err(); err != nil {
		return "", err
	}
	if in.Vencimiento != nil && in.Vencimiento.IsZero() {
		in.Vencimiento = nil
	}
	err = s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		if _, err := s.productoConStock(ctx, in.ProductoID); err != nil {
			return err
		}
		ahora := s.reloj.Ahora()
		lote := domain.Lote{ID: s.ids.Nuevo(), ProductoID: in.ProductoID, Vencimiento: in.Vencimiento,
			Costo: in.Costo, CompraItemID: in.CompraItemID, CreatedAt: ahora}
		if err := s.repo.CrearLote(ctx, lote); err != nil {
			return err
		}
		loteID = lote.ID
		return s.repo.RegistrarMovimientos(ctx, []domain.Movimiento{{
			ID: s.ids.Nuevo(), ProductoID: in.ProductoID, LoteID: &lote.ID, Cantidad: in.Cantidad,
			Tipo: in.Tipo, ReferenciaID: in.ReferenciaID, Nota: strings.TrimSpace(in.Nota), Fecha: ahora,
		}})
	})
	return loteID, err
}

// RegistrarSalida descuenta stock repartiéndolo entre lotes (primero lo que vence antes).
// Devuelve de qué lotes salió (con su costo, que ventas usa para calcular el margen).
func (s *Servicio) RegistrarSalida(ctx context.Context, sal Salida) ([]domain.Asignacion, error) {
	if sal.Cantidad <= 0 {
		var v errores.Validacion
		v.Agregar("cantidad", "La cantidad debe ser mayor a cero")
		return nil, v.Err()
	}
	var asignaciones []domain.Asignacion
	err := s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		if _, err := s.productoConStock(ctx, sal.ProductoID); err != nil {
			return err
		}
		lotes, err := s.repo.LotesConStock(ctx, sal.ProductoID)
		if err != nil {
			return err
		}
		asignaciones = domain.AsignarFEFO(lotes, sal.Cantidad)
		if !sal.PermitirNegativo && asignaciones[len(asignaciones)-1].LoteID == nil {
			stock, err := s.repo.StockProducto(ctx, sal.ProductoID)
			if err != nil {
				return err
			}
			if stock < sal.Cantidad {
				return domain.ErrStockInsuficiente(max(stock, 0))
			}
		}
		ahora := s.reloj.Ahora()
		movs := make([]domain.Movimiento, len(asignaciones))
		for i, a := range asignaciones {
			movs[i] = domain.Movimiento{ID: s.ids.Nuevo(), ProductoID: sal.ProductoID, LoteID: a.LoteID,
				Cantidad: -a.Cantidad, Tipo: sal.Tipo, ReferenciaID: sal.ReferenciaID,
				Nota: strings.TrimSpace(sal.Nota), Fecha: ahora}
		}
		return s.repo.RegistrarMovimientos(ctx, movs)
	})
	if err != nil {
		return nil, err
	}
	return asignaciones, nil
}

// AjustarStock corrige el stock a mano (conteo, rotura, vencido, regalo, ...).
// Positivo: entra a un lote existente o a uno nuevo. Negativo: sale de un lote o por FEFO,
// sin dejar el stock por debajo de cero.
func (s *Servicio) AjustarStock(ctx context.Context, cmd ComandoAjuste) (StockProductoDTO, error) {
	if err := domain.ValidarAjuste(cmd.Cantidad, cmd.Motivo); err != nil {
		return StockProductoDTO{}, err
	}
	nota := strings.TrimSpace(cmd.Motivo)
	err := s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		if _, err := s.productoConStock(ctx, cmd.ProductoID); err != nil {
			return err
		}
		if cmd.LoteID != "" {
			return s.ajustarLote(ctx, cmd, nota)
		}
		if cmd.Cantidad > 0 {
			_, err := s.RegistrarIngreso(ctx, Ingreso{ProductoID: cmd.ProductoID, Cantidad: cmd.Cantidad,
				Costo: cmd.Costo, Vencimiento: cmd.Vencimiento, Tipo: domain.MovAjuste, Nota: nota})
			return err
		}
		_, err := s.RegistrarSalida(ctx, Salida{ProductoID: cmd.ProductoID, Cantidad: -cmd.Cantidad,
			Tipo: domain.MovAjuste, Nota: nota})
		return err
	})
	if err != nil {
		return StockProductoDTO{}, err
	}
	return s.StockDeProducto(ctx, cmd.ProductoID)
}

func (s *Servicio) ajustarLote(ctx context.Context, cmd ComandoAjuste, nota string) error {
	lote, err := s.repo.ObtenerLote(ctx, cmd.LoteID)
	if err != nil {
		return err
	}
	if lote.ProductoID != cmd.ProductoID {
		return domain.ErrLoteNoEncontrado
	}
	if lote.Stock+cmd.Cantidad < 0 {
		return domain.ErrStockInsuficiente(max(lote.Stock, 0))
	}
	return s.repo.RegistrarMovimientos(ctx, []domain.Movimiento{{
		ID: s.ids.Nuevo(), ProductoID: cmd.ProductoID, LoteID: &lote.ID, Cantidad: cmd.Cantidad,
		Tipo: domain.MovAjuste, Nota: nota, Fecha: s.reloj.Ahora(),
	}})
}

// StockDeProducto devuelve el stock total y el detalle por lote con su estado de vencimiento.
func (s *Servicio) StockDeProducto(ctx context.Context, productoID string) (StockProductoDTO, error) {
	p, err := s.productoConStock(ctx, productoID)
	if err != nil {
		return StockProductoDTO{}, err
	}
	stock, err := s.repo.StockProducto(ctx, productoID)
	if err != nil {
		return StockProductoDTO{}, err
	}
	lotes, err := s.repo.LotesConStock(ctx, productoID)
	if err != nil {
		return StockProductoDTO{}, err
	}
	dias, err := s.repo.DiasAlerta(ctx)
	if err != nil {
		return StockProductoDTO{}, err
	}
	hoy := s.hoy()
	out := StockProductoDTO{ProductoID: p.ID, SKU: p.SKU, Nombre: p.Nombre, Stock: stock, Lotes: []LoteDTO{}}
	for _, l := range lotes {
		estado, restantes := domain.Clasificar(l.Vencimiento, hoy, dias)
		out.Lotes = append(out.Lotes, LoteDTO{ID: l.ID, Vencimiento: l.Vencimiento, Costo: l.Costo,
			Stock: l.Stock, Estado: estado, DiasRestantes: restantes})
	}
	return out, nil
}

// ListarMovimientos devuelve el historial de movimientos, del más reciente al más viejo.
func (s *Servicio) ListarMovimientos(ctx context.Context, f FiltroMovimientos, p paginacion.Solicitud) (paginacion.Pagina[MovimientoDTO], error) {
	if f.Tipo != "" && !f.Tipo.Valido() {
		var v errores.Validacion
		v.Agregar("tipo", "Tipo de movimiento desconocido")
		return paginacion.Pagina[MovimientoDTO]{}, v.Err()
	}
	lista, total, err := s.repo.ListarMovimientos(ctx, f, p)
	if err != nil {
		return paginacion.Pagina[MovimientoDTO]{}, err
	}
	return paginacion.NuevaPagina(lista, p, total), nil
}

// ListarVencimientos devuelve lotes con stock ordenados por fecha de vencimiento.
func (s *Servicio) ListarVencimientos(ctx context.Context, estado EstadoVencimientos, p paginacion.Solicitud) (paginacion.Pagina[VencimientoDTO], error) {
	dias, err := s.repo.DiasAlerta(ctx)
	if err != nil {
		return paginacion.Pagina[VencimientoDTO]{}, err
	}
	hoy := s.hoy()
	limite := hoy.SumarDias(dias)
	var f FiltroVencimientos
	switch estado {
	case FiltroVencidos:
		f.Hasta = &hoy
	case FiltroPorVencer:
		f.Desde, f.Hasta = &hoy, &limite
	case FiltroTodos, "":
	default:
		var v errores.Validacion
		v.Agregar("estado", "Use vencidos, por_vencer o todos")
		return paginacion.Pagina[VencimientoDTO]{}, v.Err()
	}
	lista, total, err := s.repo.ListarVencimientos(ctx, f, p)
	if err != nil {
		return paginacion.Pagina[VencimientoDTO]{}, err
	}
	for i := range lista {
		e, d := domain.Clasificar(&lista[i].Vencimiento, hoy, dias)
		lista[i].Estado, lista[i].DiasRestantes = e, *d
	}
	return paginacion.NuevaPagina(lista, p, total), nil
}

// Alertas resume vencimientos y faltantes para la pantalla de inicio.
func (s *Servicio) Alertas(ctx context.Context) (AlertasDTO, error) {
	dias, err := s.repo.DiasAlerta(ctx)
	if err != nil {
		return AlertasDTO{}, err
	}
	hoy := s.hoy()
	a, err := s.repo.ContarAlertas(ctx, hoy, hoy.SumarDias(dias))
	if err != nil {
		return AlertasDTO{}, err
	}
	a.DiasAlerta = dias
	return a, nil
}

// DiasAlerta devuelve con cuántos días de anticipación se avisa un vencimiento.
func (s *Servicio) DiasAlerta(ctx context.Context) (int, error) { return s.repo.DiasAlerta(ctx) }

// CambiarDiasAlerta configura el plazo de aviso de vencimiento.
func (s *Servicio) CambiarDiasAlerta(ctx context.Context, dias int) error {
	if err := domain.ValidarDiasAlerta(dias); err != nil {
		return err
	}
	return s.repo.GuardarDiasAlerta(ctx, dias)
}
