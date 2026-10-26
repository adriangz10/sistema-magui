// Package enlaces conecta los puertos de la sincronización con catálogo y ventas (en el mismo proceso).
package enlaces

import (
	"context"
	"fmt"

	catalogo "magui/internal/catalogo/application"
	"magui/internal/kernel/errores"
	"magui/internal/sync/application"
	ventas "magui/internal/ventas/application"
	vdomain "magui/internal/ventas/domain"
)

// Catalogo implementa application.Catalogo.
type Catalogo struct{ Svc *catalogo.Servicio }

func (c Catalogo) Copia(ctx context.Context) ([]application.ProductoCopia, []application.ComponenteCopia, error) {
	cat, err := c.Svc.ExportarCatalogo(ctx)
	if err != nil {
		return nil, nil, err
	}
	ps := make([]application.ProductoCopia, len(cat.Productos))
	for i, p := range cat.Productos {
		ps[i] = application.ProductoCopia{ID: p.ID, SKU: p.SKU, CodigoBarras: p.CodigoBarras, Nombre: p.Nombre,
			Tipo: string(p.Tipo), Empresa: p.Empresa, PrecioVenta: p.PrecioVenta, Stock: p.Stock, Activo: p.Activo, Version: p.Version}
	}
	cs := make([]application.ComponenteCopia, len(cat.Componentes))
	for i, x := range cat.Componentes {
		cs[i] = application.ComponenteCopia{ComboID: x.ComboID, ProductoID: x.ProductoID, Cantidad: x.Cantidad}
	}
	return ps, cs, nil
}

// Ventas implementa application.Ventas.
type Ventas struct{ Svc *ventas.Servicio }

func (v Ventas) RegistrarOffline(ctx context.Context, in application.VentaOffline) (bool, error) {
	cmd := ventas.VentaSincronizada{ID: in.ID, Fecha: in.Fecha, Origen: vdomain.OrigenDesktop}
	var val errores.Validacion
	for _, it := range in.Items {
		cmd.Items = append(cmd.Items, ventas.LineaSincronizada{ProductoID: it.ProductoID, Cantidad: it.Cantidad, Precio: it.Precio})
	}
	for i, p := range in.Pagos {
		m, ok := vdomain.ParseMetodo(p.Metodo)
		val.Si(!ok, fmt.Sprintf("pagos[%d].metodo", i), "Use EFECTIVO, TRANSFERENCIA, QR o TARJETA")
		cmd.Pagos = append(cmd.Pagos, vdomain.PagoIngresado{Metodo: m, Monto: p.Monto})
	}
	if err := val.Err(); err != nil {
		return false, err
	}
	r, err := v.Svc.RegistrarVentaSincronizada(ctx, cmd)
	return r.YaExistia, err
}
