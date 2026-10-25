package application

import (
	"context"
	"strings"

	"magui/internal/catalogo/domain"
)

// VistaPreviaAjuste calcula cómo quedarían los precios sin guardar nada.
func (s *Servicio) VistaPreviaAjuste(ctx context.Context, cmd ComandoAjuste) (ResultadoAjuste, error) {
	res, _, err := s.calcularAjuste(ctx, cmd)
	return res, err
}

// AplicarAjuste cambia en bloque los precios de los productos activos de una categoría
// (y sus subcategorías si se pide). Todo o nada: si un precio queda inválido no cambia ninguno.
func (s *Servicio) AplicarAjuste(ctx context.Context, cmd ComandoAjuste) (ResultadoAjuste, error) {
	var res ResultadoAjuste
	err := s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		r, productos, err := s.calcularAjuste(ctx, cmd)
		if err != nil {
			return err
		}
		res = r
		if len(productos) == 0 {
			return nil
		}
		ahora := s.reloj.Ahora()
		motivo := strings.TrimSpace(cmd.Motivo)
		if motivo == "" {
			motivo = "Ajuste por categoría"
		}
		cambios := make([]domain.CambioPrecio, 0, len(productos))
		for i := range productos {
			anterior, err := productos[i].CambiarPrecio(res.Productos[i].PrecioNuevo, ahora)
			if err != nil {
				return err
			}
			cambios = append(cambios, domain.CambioPrecio{
				ID: s.ids.Nuevo(), ProductoID: productos[i].ID, PrecioAnterior: anterior,
				PrecioNuevo: productos[i].PrecioVenta, Origen: domain.OrigenCascada, Motivo: motivo, Fecha: ahora,
			})
		}
		if err := s.productos.ActualizarPrecios(ctx, productos); err != nil {
			return err
		}
		return s.productos.RegistrarCambiosPrecio(ctx, cambios)
	})
	if err != nil {
		return ResultadoAjuste{}, err
	}
	res.Aplicado = true
	return res, nil
}

// calcularAjuste devuelve la vista previa y los productos alcanzados (en el mismo orden),
// omitiendo los que no cambian de precio.
func (s *Servicio) calcularAjuste(ctx context.Context, cmd ComandoAjuste) (ResultadoAjuste, []domain.Producto, error) {
	if err := cmd.Ajuste.Validar(); err != nil {
		return ResultadoAjuste{}, nil, err
	}
	arbol, err := s.arbol(ctx)
	if err != nil {
		return ResultadoAjuste{}, nil, err
	}
	if !arbol.Existe(cmd.CategoriaID) {
		return ResultadoAjuste{}, nil, domain.ErrCategoriaNoEncontrada
	}
	ids := []string{cmd.CategoriaID}
	if cmd.IncluirSubcategorias {
		ids = arbol.ConDescendientes(cmd.CategoriaID)
	}
	productos, err := s.productos.ActivosEnCategorias(ctx, ids)
	if err != nil {
		return ResultadoAjuste{}, nil, err
	}

	res := ResultadoAjuste{Productos: []PrecioPrevisto{}}
	alcanzados := make([]domain.Producto, 0, len(productos))
	for _, p := range productos {
		nuevo := cmd.Ajuste.Aplicar(p.PrecioVenta)
		if nuevo == p.PrecioVenta {
			continue
		}
		if !nuevo.EsPositivo() {
			// Reutiliza el mensaje del dominio, que nombra al producto.
			_, err := p.CambiarPrecio(nuevo, s.reloj.Ahora())
			return ResultadoAjuste{}, nil, err
		}
		res.Productos = append(res.Productos, PrecioPrevisto{
			ProductoID: p.ID, SKU: p.SKU, Nombre: p.Nombre, PrecioAnterior: p.PrecioVenta, PrecioNuevo: nuevo,
		})
		alcanzados = append(alcanzados, p)
	}
	res.Cantidad = len(res.Productos)
	return res, alcanzados, nil
}
