package application

import (
	"context"
	"strings"

	"magui/internal/catalogo/domain"
	"magui/internal/kernel/paginacion"
)

// ListarProductos devuelve una página de productos filtrados.
func (s *Servicio) ListarProductos(ctx context.Context, c ConsultaProductos, p paginacion.Solicitud) (paginacion.Pagina[ProductoResumen], error) {
	f := FiltroProductos{
		Texto: strings.TrimSpace(c.Texto), EmpresaID: c.EmpresaID, Tipo: c.Tipo,
		Estado: c.Estado, StockBajo: c.StockBajo, Orden: c.Orden,
	}
	if f.Estado == "" {
		f.Estado = EstadoActivos
	}
	if f.Orden == "" {
		f.Orden = OrdenNombre
	}
	if c.CategoriaID != "" {
		arbol, err := s.arbol(ctx)
		if err != nil {
			return paginacion.Pagina[ProductoResumen]{}, err
		}
		if !arbol.Existe(c.CategoriaID) {
			return paginacion.Pagina[ProductoResumen]{}, domain.ErrCategoriaNoEncontrada
		}
		f.CategoriaIDs = arbol.ConDescendientes(c.CategoriaID)
	}
	lista, total, err := s.productos.Listar(ctx, f, p)
	if err != nil {
		return paginacion.Pagina[ProductoResumen]{}, err
	}
	return paginacion.NuevaPagina(lista, p, total), nil
}

// ObtenerProducto devuelve la ficha completa.
func (s *Servicio) ObtenerProducto(ctx context.Context, id string) (ProductoDetalle, error) {
	d, err := s.productos.ObtenerDetalle(ctx, id)
	if err != nil {
		return ProductoDetalle{}, err
	}
	arbol, err := s.arbol(ctx)
	if err != nil {
		return ProductoDetalle{}, err
	}
	for i := range d.Categorias {
		d.Categorias[i].Ruta = rutaTexto(arbol, d.Categorias[i].ID)
	}
	return d, nil
}

// BuscarPorCodigo busca por SKU o código de barras exacto (para el lector del punto de venta).
func (s *Servicio) BuscarPorCodigo(ctx context.Context, codigo string) (ProductoDetalle, error) {
	codigo = strings.TrimSpace(codigo)
	if codigo == "" {
		return ProductoDetalle{}, domain.ErrProductoNoEncontrado
	}
	id, err := s.productos.IDPorCodigo(ctx, codigo)
	if err != nil {
		return ProductoDetalle{}, err
	}
	return s.ObtenerProducto(ctx, id)
}

// CrearProducto da de alta un producto simple o un combo.
func (s *Servicio) CrearProducto(ctx context.Context, cmd ComandoProducto) (ProductoDetalle, error) {
	p, err := domain.NuevoProducto(s.ids.Nuevo(), cmd.Tipo, cmd.Datos, s.reloj.Ahora())
	if err != nil {
		return ProductoDetalle{}, err
	}
	err = s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		if err := s.validarReferencias(ctx, p); err != nil {
			return err
		}
		return s.productos.Crear(ctx, p)
	})
	if err != nil {
		return ProductoDetalle{}, err
	}
	return s.ObtenerProducto(ctx, p.ID)
}

// ActualizarProducto modifica un producto. Si cambia el precio, queda en el historial.
func (s *Servicio) ActualizarProducto(ctx context.Context, id string, datos domain.DatosProducto, version int) (ProductoDetalle, error) {
	err := s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		p, err := s.productos.ObtenerPorID(ctx, id)
		if err != nil {
			return err
		}
		if p.Version != version {
			return domain.ErrVersionDesactualizada
		}
		ahora := s.reloj.Ahora()
		anterior, err := p.Modificar(datos, ahora)
		if err != nil {
			return err
		}
		if err := s.validarReferencias(ctx, p); err != nil {
			return err
		}
		if err := s.productos.Actualizar(ctx, p, version); err != nil {
			return err
		}
		if anterior == nil {
			return nil
		}
		return s.productos.RegistrarCambiosPrecio(ctx, []domain.CambioPrecio{{
			ID: s.ids.Nuevo(), ProductoID: p.ID, PrecioAnterior: *anterior, PrecioNuevo: p.PrecioVenta,
			Origen: domain.OrigenManual, Motivo: "Edición del producto", Fecha: ahora,
		}})
	})
	if err != nil {
		return ProductoDetalle{}, err
	}
	return s.ObtenerProducto(ctx, id)
}

// CambiarEstadoProducto activa o desactiva un producto.
func (s *Servicio) CambiarEstadoProducto(ctx context.Context, id string, activo bool) (ProductoDetalle, error) {
	err := s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		p, err := s.productos.ObtenerPorID(ctx, id)
		if err != nil {
			return err
		}
		version := p.Version
		p.CambiarEstado(activo, s.reloj.Ahora())
		if p.Version == version {
			return nil
		}
		return s.productos.Actualizar(ctx, p, version)
	})
	if err != nil {
		return ProductoDetalle{}, err
	}
	return s.ObtenerProducto(ctx, id)
}

// HistorialPrecios devuelve los cambios de precio de un producto, del más reciente al más viejo.
func (s *Servicio) HistorialPrecios(ctx context.Context, productoID string, p paginacion.Solicitud) (paginacion.Pagina[CambioPrecioDTO], error) {
	if _, err := s.productos.ObtenerPorID(ctx, productoID); err != nil {
		return paginacion.Pagina[CambioPrecioDTO]{}, err
	}
	lista, total, err := s.productos.HistorialPrecios(ctx, productoID, p)
	if err != nil {
		return paginacion.Pagina[CambioPrecioDTO]{}, err
	}
	return paginacion.NuevaPagina(lista, p, total), nil
}

// validarReferencias revisa códigos únicos, empresa, categorías y componentes del combo.
func (s *Servicio) validarReferencias(ctx context.Context, p domain.Producto) error {
	if enUso, err := s.productos.CodigoEnUso(ctx, p.SKU, p.ID); err != nil {
		return err
	} else if enUso {
		return domain.ErrSKUDuplicado
	}
	if p.CodigoBarras != nil {
		if enUso, err := s.productos.CodigoEnUso(ctx, *p.CodigoBarras, p.ID); err != nil {
			return err
		} else if enUso {
			return domain.ErrCodigoBarrasDuplicado
		}
	}

	if p.EmpresaID != nil {
		if _, err := s.empresas.ObtenerPorID(ctx, *p.EmpresaID); err != nil {
			return err
		}
	}

	if len(p.CategoriaIDs) > 0 {
		arbol, err := s.arbol(ctx)
		if err != nil {
			return err
		}
		for _, c := range p.CategoriaIDs {
			if !arbol.Existe(c) {
				return domain.ErrCategoriaNoEncontrada
			}
		}
	}

	if p.Tipo == domain.TipoCombo {
		return s.validarComponentes(ctx, p)
	}
	return nil
}

func (s *Servicio) validarComponentes(ctx context.Context, combo domain.Producto) error {
	ids := make([]string, len(combo.Items))
	for i, it := range combo.Items {
		ids[i] = it.ProductoID
	}
	encontrados, err := s.productos.ObtenerVarios(ctx, ids)
	if err != nil {
		return err
	}
	porID := make(map[string]domain.Producto, len(encontrados))
	for _, p := range encontrados {
		porID[p.ID] = p
	}
	for _, it := range combo.Items {
		comp, ok := porID[it.ProductoID]
		switch {
		case !ok:
			return domain.ErrComboInvalido("Uno de los productos del combo no existe")
		case comp.Tipo != domain.TipoSimple:
			return domain.ErrComboInvalido("Un combo no puede incluir otro combo: " + comp.Nombre)
		case !comp.Activo:
			return domain.ErrComboInvalido("El producto está desactivado: " + comp.Nombre)
		}
	}
	return nil
}
