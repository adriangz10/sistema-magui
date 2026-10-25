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
}

// ProductoCompra son los datos del catálogo que necesita una compra.
type ProductoCompra struct {
	Nombre string
	Combo  bool
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
}

// Inventario es el puerto hacia el módulo de inventario.
type Inventario interface {
	Ingresar(ctx context.Context, in IngresoCompra) error
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
}

type CompraDetalle struct {
	CompraResumen
	Items []ItemDTO `json:"items"`
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
		costos := make([]CostoProducto, 0, len(compra.Items))
		for _, it := range compra.Items {
			if err := s.inventario.Ingresar(ctx, IngresoCompra{CompraID: compra.ID, CompraItemID: it.ID,
				ProductoID: it.ProductoID, Cantidad: it.Cantidad, Costo: it.CostoUnitario, Vencimiento: it.Vencimiento}); err != nil {
				return err
			}
			costos = append(costos, CostoProducto{ProductoID: it.ProductoID, Costo: it.CostoUnitario, PrecioVenta: it.PrecioVentaNuevo})
		}
		return s.catalogo.RegistrarCostos(ctx, costos, "Compra del "+fecha.DelInstante(compra.Fecha, reloj.ZonaArgentina).String())
	})
	if err != nil {
		return CompraDetalle{}, err
	}
	return s.repo.Obtener(ctx, compra.ID)
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
	for _, it := range c.Items {
		p, ok := prods[it.ProductoID]
		if !ok {
			return domain.ErrProductoNoEncontrado
		}
		if p.Combo {
			return domain.ErrProductoNoComprable(p.Nombre)
		}
	}
	return nil
}

// ObtenerCompra devuelve el detalle de una compra.
func (s *Servicio) ObtenerCompra(ctx context.Context, id string) (CompraDetalle, error) {
	return s.repo.Obtener(ctx, id)
}

// ListarCompras devuelve las compras, de la más reciente a la más vieja.
func (s *Servicio) ListarCompras(ctx context.Context, f Filtro, p paginacion.Solicitud) (paginacion.Pagina[CompraResumen], error) {
	lista, total, err := s.repo.Listar(ctx, f, p)
	if err != nil {
		return paginacion.Pagina[CompraResumen]{}, err
	}
	return paginacion.NuevaPagina(lista, p, total), nil
}
