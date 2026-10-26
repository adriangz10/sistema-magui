package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/ventas/domain"
)

// LineaSincronizada es un renglón cobrado sin conexión, con el precio que se cobró en ese momento.
type LineaSincronizada struct {
	ProductoID string
	Cantidad   int
	Precio     dinero.Dinero
}

// VentaSincronizada es una venta hecha en el escritorio sin conexión que llega al servidor.
type VentaSincronizada struct {
	// ID generado por el escritorio (UUID). Permite reenviar la venta sin duplicarla.
	ID     string
	Fecha  time.Time
	Origen domain.Origen
	Items  []LineaSincronizada
	Pagos  []domain.PagoIngresado
}

// ResultadoSincronizada indica si la venta se registró ahora o ya estaba (reenvío).
type ResultadoSincronizada struct {
	Venta     VentaDetalle
	YaExistia bool
}

// toleranciaReloj: si el reloj de la PC está adelantado, la venta no puede quedar en el futuro.
const toleranciaReloj = 5 * time.Minute

// RegistrarVentaSincronizada guarda una venta cobrada sin conexión. A diferencia de RegistrarVenta:
//   - respeta el ID, la fecha y los precios que se cobraron (ya pasó, no se puede rechazar por precio);
//   - acepta productos desactivados después de la venta y descuenta aunque no haya stock registrado;
//   - es idempotente: si la venta ya existe, devuelve la existente sin tocar el stock.
func (s *Servicio) RegistrarVentaSincronizada(ctx context.Context, cmd VentaSincronizada) (ResultadoSincronizada, error) {
	var v errores.Validacion
	_, errID := uuid.Parse(cmd.ID)
	v.Si(errID != nil, "id", "Debe ser un UUID")
	if err := v.Err(); err != nil {
		return ResultadoSincronizada{}, err
	}
	if cmd.Origen == "" {
		cmd.Origen = domain.OrigenDesktop
	}
	ahora := s.reloj.Ahora()
	if cmd.Fecha.IsZero() || cmd.Fecha.After(ahora.Add(toleranciaReloj)) {
		cmd.Fecha = ahora
	}

	if existe, err := s.repo.Existe(ctx, cmd.ID); err != nil {
		return ResultadoSincronizada{}, err
	} else if existe {
		return s.existente(ctx, cmd.ID)
	}

	err := s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		ids := make([]string, len(cmd.Items))
		for i, it := range cmd.Items {
			ids[i] = it.ProductoID
		}
		prods, err := s.catalogo.Productos(ctx, ids)
		if err != nil {
			return err
		}
		lineas := make([]domain.LineaVenta, len(cmd.Items))
		for i, it := range cmd.Items {
			if _, ok := prods[it.ProductoID]; !ok {
				return domain.ErrProductoNoEncontrado
			}
			lineas[i] = domain.LineaVenta{ProductoID: it.ProductoID, Cantidad: it.Cantidad, Precio: it.Precio}
		}
		return s.guardarVenta(ctx, cmd.ID, lineas, cmd.Pagos, cmd.Origen, cmd.Fecha, ahora, prods, true)
	})
	if err != nil {
		// Dos envíos simultáneos de la misma venta: el segundo choca con el primero.
		if existe, e := s.repo.Existe(ctx, cmd.ID); e == nil && existe {
			return s.existente(ctx, cmd.ID)
		}
		return ResultadoSincronizada{}, err
	}
	det, err := s.repo.Detalle(ctx, cmd.ID)
	return ResultadoSincronizada{Venta: det}, err
}

func (s *Servicio) existente(ctx context.Context, id string) (ResultadoSincronizada, error) {
	det, err := s.repo.Detalle(ctx, id)
	return ResultadoSincronizada{Venta: det, YaExistia: true}, err
}
