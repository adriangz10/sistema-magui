// Package enlaces conecta los puertos de ventas con catálogo e inventario (en el mismo proceso).
package enlaces

import (
	"context"

	catalogo "magui/internal/catalogo/application"
	catdomain "magui/internal/catalogo/domain"
	inventario "magui/internal/inventario/application"
	invdomain "magui/internal/inventario/domain"
	"magui/internal/ventas/application"
)

// Catalogo implementa application.Catalogo.
type Catalogo struct{ Svc *catalogo.Servicio }

func (c Catalogo) Productos(ctx context.Context, ids []string) (map[string]application.ProductoVenta, error) {
	info, err := c.Svc.InfoProductos(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]application.ProductoVenta, len(info))
	for id, p := range info {
		pv := application.ProductoVenta{ID: p.ID, Nombre: p.Nombre, Combo: p.Tipo == catdomain.TipoCombo,
			Activo: p.Activo, Precio: p.PrecioVenta, CostoUltimo: p.Costo}
		for _, it := range p.Items {
			pv.Componentes = append(pv.Componentes, application.Componente{ProductoID: it.ProductoID, Cantidad: it.Cantidad})
		}
		out[id] = pv
	}
	return out, nil
}

// Inventario implementa application.Inventario.
type Inventario struct{ Svc *inventario.Servicio }

func (i Inventario) Descontar(ctx context.Context, s application.SalidaVenta) ([]application.ParteSalida, error) {
	ref := s.VentaID
	asig, err := i.Svc.RegistrarSalida(ctx, inventario.Salida{ProductoID: s.ProductoID, Cantidad: s.Cantidad,
		Tipo: invdomain.MovVenta, ReferenciaID: &ref, PermitirNegativo: s.PermitirSinStock})
	if err != nil {
		return nil, err
	}
	out := make([]application.ParteSalida, len(asig))
	for j, a := range asig {
		out[j] = application.ParteSalida{Cantidad: a.Cantidad, Costo: a.Costo}
	}
	return out, nil
}

func (i Inventario) Reponer(ctx context.Context, ventaID, nota string) error {
	_, err := i.Svc.RevertirSalidas(ctx, ventaID, invdomain.MovVenta, nota)
	return err
}
