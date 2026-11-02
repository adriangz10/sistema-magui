package application

import (
	"context"
	"errors"
	"testing"

	"magui/internal/kernel/errores"
	"magui/internal/ventas/domain"
)

var errCajaCerrada = errores.Conflicto("caja_cerrada", "La caja está cerrada")

type cajaFalsa struct{ abierta bool }

func (c cajaFalsa) ExigirAbierta(context.Context) error {
	if !c.abierta {
		return errCajaCerrada
	}
	return nil
}

// US-002, D-1: sin caja abierta no se cobra; nada se guarda ni se descuenta.
func TestVentaExigeCajaAbierta(t *testing.T) {
	s, repo, inv := entorno()
	s.caja = cajaFalsa{abierta: false}
	cmd := ComandoVenta{Items: []LineaCmd{{"jabon", 1}}, Pagos: cobro(domain.PagoIngresado{Metodo: domain.Efectivo, Monto: d("5500")})}
	if _, err := s.RegistrarVenta(ctx, cmd); !errors.Is(err, errCajaCerrada) {
		t.Fatalf("err = %v", err)
	}
	if len(repo.ventas) != 0 || inv.stock["jabon"] != 10 {
		t.Errorf("no debería guardar ni descontar: ventas=%d stock=%d", len(repo.ventas), inv.stock["jabon"])
	}

	s.caja = cajaFalsa{abierta: true}
	if _, err := s.RegistrarVenta(ctx, cmd); err != nil {
		t.Fatalf("con caja abierta: %v", err)
	}
}
