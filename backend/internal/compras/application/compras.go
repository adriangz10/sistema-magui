// Package application contiene los casos de uso de compras de mercadería.
package application

import (
	"context"
	"time"

	"magui/internal/compras/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/id"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/kernel/tx"
)

// --- Puertos ---

// Repositorio persiste compras.
type Repositorio interface {
	Guardar(ctx context.Context, c domain.Compra) error
	Obtener(ctx context.Context, id string) (CompraDetalle, error)
	Listar(ctx context.Context, f Filtro, p paginacion.Solicitud) ([]CompraResumen, int, error)

	// ObtenerParaEditar lee la compra con sus renglones y bloquea su fila hasta el fin de la
	// transacción (SELECT … FOR UPDATE, ADR 0002).
	ObtenerParaEditar(ctx context.Context, id string) (domain.Compra, error)
	// ActualizarCompra graba la cabecera y la versión nuevas si la fila sigue en versionAnterior;
	// si no, devuelve domain.ErrVersionDesactualizada.
	ActualizarCompra(ctx context.Context, c domain.Compra, versionAnterior int) error
	GuardarEdicion(ctx context.Context, e domain.Edicion) error
	// Ediciones devuelve el historial de la compra, de la más reciente a la más vieja.
	Ediciones(ctx context.Context, compraID string) ([]domain.Edicion, error)
	// ActualizarItems graba cantidad, costo, precio y vencimiento de renglones existentes.
	ActualizarItems(ctx context.Context, items []domain.Item) error
	// AgregarItems inserta renglones nuevos en una compra existente (US-001c).
	AgregarItems(ctx context.Context, compraID string, items []domain.Item) error
	// QuitarItems da de baja renglones (quitado_en = ahora); el renglón y su lote se conservan (ADR 0004).
	// Falla si algún id no es un renglón vigente.
	QuitarItems(ctx context.Context, ids []string, ahora time.Time) error
	// CompraMasReciente devuelve la compra más reciente que contiene al producto (por día argentino de
	// la fecha; empate: created_at e id) con el costo de su último renglón vigente de ese producto; nil si no hay ninguna.
	CompraMasReciente(ctx context.Context, productoID string) (*CompraReciente, error)
}

// CompraReciente es el resultado de CompraMasReciente.
type CompraReciente struct {
	CompraID string
	Costo    dinero.Dinero // costo del último renglón (mayor id) del producto en esa compra
}

// ProductoCompra son los datos del catálogo que necesita una compra.
type ProductoCompra struct {
	Nombre      string
	Combo       bool
	Inactivo    bool           // producto desactivado en el catálogo (el cero es activo)
	Costo      *dinero.Dinero // último costo actual del producto
	PrecioVenta dinero.Dinero  // precio de venta actual
}

// CostoProducto es lo que una compra actualiza en el catálogo.
type CostoProducto struct {
	ProductoID  string
	Costo       dinero.Dinero
	PrecioVenta *dinero.Dinero
}

// Catalogo es el puerto hacia el módulo de catálogo.
type Catalogo interface {
	Productos(ctx context.Context, ids []string) (map[string]ProductoCompra, error)
	ValidarEmpresa(ctx context.Context, id string) error
	// NombreEmpresa devuelve el nombre de la empresa o empresa_no_encontrada.
	NombreEmpresa(ctx context.Context, id string) (string, error)
	RegistrarCostos(ctx context.Context, costos []CostoProducto, motivo string) error
}

// IngresoCompra es la entrada de stock de un renglón.
type IngresoCompra struct {
	CompraID     string
	CompraItemID string
	ProductoID   string
	Cantidad     int
	Costo        dinero.Dinero
	Vencimiento  *fecha.Fecha
	Nota         string // opcional (p. ej. "Edición de compra del dd/mm/aaaa" para renglones agregados)
}

// Inventario es el puerto hacia el módulo de inventario.
type Inventario interface {
	Ingresar(ctx context.Context, in IngresoCompra) error
	// LotesDeRenglones devuelve, por id de renglón, su lote y el stock actual. Con bloquear toma los
	// lotes con FOR UPDATE y lee el stock de forma bloqueante (ADR 0002).
	LotesDeRenglones(ctx context.Context, itemIDs []string, bloquear bool) (map[string]domain.LoteRenglon, error)
	// AjustarLoteDeRenglon aplica la diferencia de cantidad (EDICION_COMPRA) y el costo y vencimiento nuevos.
	AjustarLoteDeRenglon(ctx context.Context, a AjusteRenglon) error
}

// AjusteRenglon corrige el lote de un renglón de compra editado.
type AjusteRenglon struct {
	LoteID      string
	Diferencia  int
	Costo       dinero.Dinero
	Vencimiento *fecha.Fecha
	CompraID    string
	Nota        string
	// SoloCantidad: no tocar costo ni vencimiento del lote (al quitar un renglón solo se descuenta stock).
	SoloCantidad bool
}

// --- Modelos ---

type Filtro struct {
	Desde     *time.Time
	Hasta     *time.Time // exclusivo
	EmpresaID string
}

type CompraResumen struct {
	ID        string        `json:"id"`
	Fecha     time.Time     `json:"fecha"`
	EmpresaID *string       `json:"empresaId"`
	Empresa   *string       `json:"empresa"`
	Total     dinero.Dinero `json:"total"`
	Productos int           `json:"productos"` // renglones
	Unidades  int           `json:"unidades"`
	Notas     *string       `json:"notas"`
	Version   int           `json:"version"`
	EditadaEn *time.Time    `json:"editadaEn"` // null si nunca se editó
}

type ItemDTO struct {
	ID               string         `json:"id"`
	ProductoID       string         `json:"productoId"`
	SKU              string         `json:"sku"`
	Producto         string         `json:"producto"`
	Cantidad         int            `json:"cantidad"`
	CostoUnitario    dinero.Dinero  `json:"costoUnitario"`
	Subtotal         dinero.Dinero  `json:"subtotal"`
	PrecioVentaNuevo *dinero.Dinero `json:"precioVentaNuevo"`
	Vencimiento      *fecha.Fecha   `json:"vencimiento"`
	Consumido        int            `json:"consumido"` // unidades del lote que ya salieron (cantidad - stock del lote)
	// PrecioVentaActual es el precio de venta vigente hoy del producto en el catálogo (solo lectura;
	// null si el catálogo no lo conoce). Sirve para mostrar "el precio pasará de $A a $B".
	PrecioVentaActual *dinero.Dinero `json:"precioVentaActual"`
}

type CompraDetalle struct {
	CompraResumen
	Items     []ItemDTO    `json:"items"`
	Ediciones []EdicionDTO `json:"ediciones"`
}

// EdicionDTO es una entrada del historial de ediciones (ADR 0005, sin el campo `v`).
type EdicionDTO struct {
	ID        string                 `json:"id"`
	Fecha     time.Time              `json:"fecha"`
	Origen    domain.Origen          `json:"origen"`
	Motivo    *string                `json:"motivo"`
	Version   int                    `json:"version"`
	Cabecera  []domain.CambioCampo   `json:"cabecera"`
	Renglones []domain.CambioRenglon `json:"renglones"`
}

// ItemCmd es un renglón tal como lo carga la usuaria.
type ItemCmd struct {
	ProductoID       string
	Cantidad         int
	CostoUnitario    dinero.Dinero
	PrecioVentaNuevo *dinero.Dinero
	Vencimiento      *fecha.Fecha
}

// ComandoCompra registra una compra. Fecha nil = ahora.
type ComandoCompra struct {
	Fecha     *time.Time
	EmpresaID string
	Notas     string
	Items     []ItemCmd
}

// --- Servicio ---

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

// RegistrarCompra guarda la compra, suma el stock (un lote por renglón, con su vencimiento
// y costo) y actualiza en el catálogo el último costo y el precio de reventa indicado.
// Todo en una transacción: si algo falla, no queda nada a medias.
func (s *Servicio) RegistrarCompra(ctx context.Context, cmd ComandoCompra) (CompraDetalle, error) {
	ahora := s.reloj.Ahora()
	fechaCompra := ahora
	if cmd.Fecha != nil {
		fechaCompra = *cmd.Fecha
	}
	items := make([]domain.Item, len(cmd.Items))
	for i, it := range cmd.Items {
		items[i] = domain.Item{ID: s.ids.Nuevo(), ProductoID: it.ProductoID, Cantidad: it.Cantidad,
			CostoUnitario: it.CostoUnitario, PrecioVentaNuevo: it.PrecioVentaNuevo, Vencimiento: it.Vencimiento}
	}
	compra, err := domain.NuevaCompra(s.ids.Nuevo(), fechaCompra, cmd.EmpresaID, cmd.Notas, items, ahora)
	if err != nil {
		return CompraDetalle{}, err
	}

	err = s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		if err := s.validarProductos(ctx, compra); err != nil {
			return err
		}
		if compra.EmpresaID != nil {
			if err := s.catalogo.ValidarEmpresa(ctx, *compra.EmpresaID); err != nil {
				return err
			}
		}
		if err := s.repo.Guardar(ctx, compra); err != nil {
			return err
		}
		for _, it := range compra.Items {
			if err := s.inventario.Ingresar(ctx, IngresoCompra{CompraID: compra.ID, CompraItemID: it.ID,
				ProductoID: it.ProductoID, Cantidad: it.Cantidad, Costo: it.CostoUnitario, Vencimiento: it.Vencimiento}); err != nil {
				return err
			}
		}
		// Último costo y precio: solo si esta es la compra más reciente del producto (P-A1).
		costos, err := s.costosDeAlta(ctx, compra)
		if err != nil || len(costos) == 0 {
			return err
		}
		return s.catalogo.RegistrarCostos(ctx, costos, "Compra del "+fecha.DelInstante(compra.Fecha, reloj.ZonaArgentina).String())
	})
	if err != nil {
		return CompraDetalle{}, err
	}
	return s.ObtenerCompra(ctx, compra.ID)
}

// costosDeAlta arma lo que la compra recién guardada cambia en el catálogo: por cada producto en
// el que es la compra más reciente, el costo de su último renglón y el último precio informado.
func (s *Servicio) costosDeAlta(ctx context.Context, c domain.Compra) ([]CostoProducto, error) {
	var out []CostoProducto
	vistos := map[string]bool{}
	for _, it := range c.Items {
		if vistos[it.ProductoID] {
			continue
		}
		vistos[it.ProductoID] = true
		rec, err := s.repo.CompraMasReciente(ctx, it.ProductoID)
		if err != nil {
			return nil, err
		}
		if rec == nil || rec.CompraID != c.ID {
			continue
		}
		cp := CostoProducto{ProductoID: it.ProductoID, Costo: rec.Costo}
		for _, o := range c.Items {
			if o.ProductoID == it.ProductoID && o.PrecioVentaNuevo != nil {
				cp.PrecioVenta = o.PrecioVentaNuevo
			}
		}
		out = append(out, cp)
	}
	return out, nil
}

func (s *Servicio) validarProductos(ctx context.Context, c domain.Compra) error {
	ids := make([]string, len(c.Items))
	for i, it := range c.Items {
		ids[i] = it.ProductoID
	}
	prods, err := s.catalogo.Productos(ctx, ids)
	if err != nil {
		return err
	}
	for i, it := range c.Items {
		p, ok := prods[it.ProductoID]
		if !ok {
			return domain.ErrProductoNoEncontrado
		}
		if p.Combo {
			return domain.ErrProductoNoComprable(p.Nombre)
		}
		if p.Inactivo {
			return domain.ErrProductoInactivo(p.Nombre, i)
		}
	}
	return nil
}

// ObtenerCompra devuelve el detalle de una compra.
func (s *Servicio) ObtenerCompra(ctx context.Context, id string) (CompraDetalle, error) {
	det, err := s.repo.Obtener(ctx, id)
	if err != nil {
		return CompraDetalle{}, err
	}
	eds, err := s.repo.Ediciones(ctx, id)
	if err != nil {
		return CompraDetalle{}, err
	}
	if err := s.completarConsumo(ctx, &det); err != nil {
		return CompraDetalle{}, err
	}
	det.Ediciones = make([]EdicionDTO, len(eds))
	for i, e := range eds {
		det.Ediciones[i] = EdicionDTO{ID: e.ID, Fecha: e.Fecha, Origen: e.Origen, Motivo: e.Motivo, Version: e.Version,
			Cabecera: e.Cambios.Cabecera, Renglones: e.Cambios.Renglones}
	}
	return det, nil
}

// completarConsumo calcula items[].consumido (unidades del lote del renglón que ya salieron) y
// items[].precioVentaActual (una sola consulta al catálogo para todos los productos de la compra).
func (s *Servicio) completarConsumo(ctx context.Context, det *CompraDetalle) error {
	ids := make([]string, len(det.Items))
	for i, it := range det.Items {
		ids[i] = it.ID
	}
	lotes, err := s.inventario.LotesDeRenglones(ctx, ids, false)
	if err != nil {
		return err
	}
	pids := make([]string, 0, len(det.Items))
	vistos := map[string]bool{}
	for _, it := range det.Items {
		if !vistos[it.ProductoID] {
			vistos[it.ProductoID] = true
			pids = append(pids, it.ProductoID)
		}
	}
	prods, err := s.catalogo.Productos(ctx, pids)
	if err != nil {
		return err
	}
	for i, it := range det.Items {
		if p, ok := prods[it.ProductoID]; ok {
			precio := p.PrecioVenta
			det.Items[i].PrecioVentaActual = &precio
		}
		if l, ok := lotes[it.ID]; ok {
			det.Items[i].Consumido = max(0, it.Cantidad-l.Stock)
		}
	}
	return nil
}

// ListarCompras devuelve las compras, de la más reciente a la más vieja.
func (s *Servicio) ListarCompras(ctx context.Context, f Filtro, p paginacion.Solicitud) (paginacion.Pagina[CompraResumen], error) {
	lista, total, err := s.repo.Listar(ctx, f, p)
	if err != nil {
		return paginacion.Pagina[CompraResumen]{}, err
	}
	return paginacion.NuevaPagina(lista, p, total), nil
}
