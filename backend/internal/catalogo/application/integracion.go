package application

import (
	"context"

	"magui/internal/catalogo/domain"
	"magui/internal/kernel/dinero"
)

// Casos de uso que otros módulos (inventario, compras, ventas) consumen a través de sus puertos.

// InfoProducto es la vista mínima de un producto para otros módulos.
type InfoProducto struct {
	ID          string
	SKU         string
	Nombre      string
	Tipo        domain.Tipo
	Activo      bool
	PrecioVenta dinero.Dinero
	Costo       *dinero.Dinero
	Items       []domain.ComboItem
}

// InfoProductos devuelve los productos encontrados indexados por ID.
func (s *Servicio) InfoProductos(ctx context.Context, ids []string) (map[string]InfoProducto, error) {
	lista, err := s.productos.ObtenerVarios(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]InfoProducto, len(lista))
	for _, p := range lista {
		info := InfoProducto{ID: p.ID, SKU: p.SKU, Nombre: p.Nombre, Tipo: p.Tipo, Activo: p.Activo,
			PrecioVenta: p.PrecioVenta, Costo: p.Costo}
		if p.Tipo == domain.TipoCombo {
			completo, err := s.productos.ObtenerPorID(ctx, p.ID)
			if err != nil {
				return nil, err
			}
			info.Items = completo.Items
		}
		out[p.ID] = info
	}
	return out, nil
}

// ComponenteCombo indica que un combo lleva cierta cantidad de un producto.
type ComponenteCombo struct {
	ComboID    string `json:"comboId"`
	ProductoID string `json:"productoId"`
	Cantidad   int    `json:"cantidad"`
}

// CatalogoCompleto es la copia del catálogo que descarga el escritorio para vender sin conexión.
type CatalogoCompleto struct {
	Productos   []ProductoResumen `json:"productos"`
	Componentes []ComponenteCombo `json:"componentes"`
}

// ExportarCatalogo devuelve todos los productos (activos e inactivos) con su stock y los
// componentes de los combos. Es chico (cientos de productos): se descarga entero.
func (s *Servicio) ExportarCatalogo(ctx context.Context) (CatalogoCompleto, error) {
	prods, comps, err := s.productos.Exportar(ctx)
	if err != nil {
		return CatalogoCompleto{}, err
	}
	if prods == nil {
		prods = []ProductoResumen{}
	}
	if comps == nil {
		comps = []ComponenteCombo{}
	}
	return CatalogoCompleto{Productos: prods, Componentes: comps}, nil
}

// ObtenerEmpresa devuelve una empresa (para validar referencias desde otros módulos).
func (s *Servicio) ObtenerEmpresa(ctx context.Context, id string) (EmpresaDTO, error) {
	e, err := s.empresas.ObtenerPorID(ctx, id)
	if err != nil {
		return EmpresaDTO{}, err
	}
	return empresaDTO(e), nil
}

// CostoCompra es lo que una compra cambia en un producto.
type CostoCompra struct {
	ProductoID  string
	Costo       dinero.Dinero
	PrecioVenta *dinero.Dinero // nuevo precio de reventa (opcional, se carga a mano)
}

// RegistrarCostosCompra guarda el último costo de cada producto y, si se indicó, su nuevo
// precio de reventa (queda en el historial con origen COMPRA).
func (s *Servicio) RegistrarCostosCompra(ctx context.Context, cambios []CostoCompra, motivo string) error {
	return s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		ahora := s.reloj.Ahora()
		var historial []domain.CambioPrecio
		for _, c := range cambios {
			p, err := s.productos.ObtenerPorID(ctx, c.ProductoID)
			if err != nil {
				return err
			}
			version := p.Version
			anterior, err := p.RegistrarCompra(c.Costo, c.PrecioVenta, ahora)
			if err != nil {
				return err
			}
			if err := s.productos.Actualizar(ctx, p, version); err != nil {
				return err
			}
			if anterior != nil {
				historial = append(historial, domain.CambioPrecio{ID: s.ids.Nuevo(), ProductoID: p.ID,
					PrecioAnterior: *anterior, PrecioNuevo: p.PrecioVenta, Origen: domain.OrigenCompra,
					Motivo: motivo, Fecha: ahora})
			}
		}
		return s.productos.RegistrarCambiosPrecio(ctx, historial)
	})
}
