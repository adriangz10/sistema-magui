// Package enlaces conecta los puertos de compras con los módulos de catálogo e inventario
// (llamadas en el mismo proceso; comparten la transacción a través del contexto).
package enlaces

import (
	"context"

	catalogo "magui/internal/catalogo/application"
	catdomain "magui/internal/catalogo/domain"
	"magui/internal/compras/application"
	inventario "magui/internal/inventario/application"
	invdomain "magui/internal/inventario/domain"
)

// Catalogo implementa application.Catalogo.
type Catalogo struct{ Svc *catalogo.Servicio }

func (c Catalogo) Productos(ctx context.Context, ids []string) (map[string]application.ProductoCompra, error) {
	info, err := c.Svc.InfoProductos(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]application.ProductoCompra, len(info))
	for id, p := range info {
		out[id] = application.ProductoCompra{Nombre: p.Nombre, Combo: p.Tipo == catdomain.TipoCombo}
	}
	return out, nil
}

func (c Catalogo) ValidarEmpresa(ctx context.Context, id string) error {
	_, err := c.Svc.ObtenerEmpresa(ctx, id)
	return err
}

func (c Catalogo) NombreEmpresa(ctx context.Context, id string) (string, error) {
	e, err := c.Svc.ObtenerEmpresa(ctx, id)
	if err != nil {
		return "", err
	}
	return e.Nombre, nil
}

func (c Catalogo) RegistrarCostos(ctx context.Context, costos []application.CostoProducto, motivo string) error {
	cambios := make([]catalogo.CostoCompra, len(costos))
	for i, co := range costos {
		cambios[i] = catalogo.CostoCompra{ProductoID: co.ProductoID, Costo: co.Costo, PrecioVenta: co.PrecioVenta}
	}
	return c.Svc.RegistrarCostosCompra(ctx, cambios, motivo)
}

// Inventario implementa application.Inventario.
type Inventario struct{ Svc *inventario.Servicio }

func (i Inventario) Ingresar(ctx context.Context, in application.IngresoCompra) error {
	costo := in.Costo
	_, err := i.Svc.RegistrarIngreso(ctx, inventario.Ingreso{
		ProductoID: in.ProductoID, Cantidad: in.Cantidad, Costo: &costo, Vencimiento: in.Vencimiento,
		CompraItemID: &in.CompraItemID, Tipo: invdomain.MovCompra, ReferenciaID: &in.CompraID,
	})
	return err
}
