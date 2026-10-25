// Package catalogo conecta el puerto Productos del inventario con el módulo de catálogo.
package catalogo

import (
	"context"

	catalogo "magui/internal/catalogo/application"
	catdomain "magui/internal/catalogo/domain"
	"magui/internal/inventario/application"
)

// Productos implementa application.Productos usando el servicio del catálogo.
type Productos struct{ Catalogo *catalogo.Servicio }

func (p Productos) Info(ctx context.Context, ids []string) (map[string]application.InfoProducto, error) {
	info, err := p.Catalogo.InfoProductos(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]application.InfoProducto, len(info))
	for id, i := range info {
		out[id] = application.InfoProducto{ID: i.ID, SKU: i.SKU, Nombre: i.Nombre,
			Combo: i.Tipo == catdomain.TipoCombo, Activo: i.Activo}
	}
	return out, nil
}
