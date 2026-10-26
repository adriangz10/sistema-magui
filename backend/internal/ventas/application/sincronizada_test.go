package application

import (
	"testing"
	"time"

	"magui/internal/kernel/errores"
	"magui/internal/ventas/domain"
)

const idOffline = "0192f3a0-1111-7000-8000-000000000001"

func ventaOffline() VentaSincronizada {
	return VentaSincronizada{
		ID:    idOffline,
		Fecha: time.Date(2026, 10, 24, 18, 30, 0, 0, time.UTC),
		// precio viejo: se respeta lo que se cobró, no el precio actual (59200)
		Items: []LineaSincronizada{{ProductoID: "kaiak", Cantidad: 1, Precio: d("55000")}},
		Pagos: cobro(domain.PagoIngresado{Metodo: domain.Efectivo, Monto: d("55000")}),
	}
}

func TestVentaSincronizadaRespetaIDFechaYPrecio(t *testing.T) {
	s, repo, inv := entorno()
	r, err := s.RegistrarVentaSincronizada(ctx, ventaOffline())
	if err != nil {
		t.Fatal(err)
	}
	if r.YaExistia || r.Venta.ID != idOffline {
		t.Fatalf("resultado = %+v", r)
	}
	v := repo.ventas[idOffline]
	if !v.Fecha.Equal(ventaOffline().Fecha) || v.Origen != domain.OrigenDesktop || v.Total != d("55000") {
		t.Errorf("venta = fecha %v origen %s total %s", v.Fecha, v.Origen, v.Total)
	}
	if !v.CreatedAt.Equal(time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)) {
		t.Errorf("created_at = %v, want hora del servidor", v.CreatedAt)
	}
	if inv.stock["kaiak"] != 4 {
		t.Errorf("stock = %d, want 4", inv.stock["kaiak"])
	}
}

func TestVentaSincronizadaEsIdempotente(t *testing.T) {
	s, repo, inv := entorno()
	if _, err := s.RegistrarVentaSincronizada(ctx, ventaOffline()); err != nil {
		t.Fatal(err)
	}
	r, err := s.RegistrarVentaSincronizada(ctx, ventaOffline())
	if err != nil {
		t.Fatal(err)
	}
	if !r.YaExistia || len(repo.ventas) != 1 || inv.stock["kaiak"] != 4 {
		t.Errorf("reenvío: yaExistia=%v ventas=%d stock=%d", r.YaExistia, len(repo.ventas), inv.stock["kaiak"])
	}
}

func TestVentaSincronizadaAceptaBajaYSinStock(t *testing.T) {
	s, _, inv := entorno()
	cmd := ventaOffline()
	cmd.Items = []LineaSincronizada{{ProductoID: "baja", Cantidad: 2, Precio: d("100")}, {ProductoID: "kaiak", Cantidad: 9, Precio: d("100")}}
	cmd.Pagos = cobro(domain.PagoIngresado{Metodo: domain.Transferencia, Monto: d("1100")})
	if _, err := s.RegistrarVentaSincronizada(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if inv.stock["kaiak"] != -4 {
		t.Errorf("stock = %d, want -4 (se vendió sin stock registrado)", inv.stock["kaiak"])
	}
}

func TestVentaSincronizadaErrores(t *testing.T) {
	s, repo, _ := entorno()
	casos := map[string]func(*VentaSincronizada){
		"id invalido":     func(c *VentaSincronizada) { c.ID = "abc" },
		"no existe":       func(c *VentaSincronizada) { c.Items[0].ProductoID = "nada" },
		"pago incompleto": func(c *VentaSincronizada) { c.Pagos[0].Monto = d("100") },
	}
	for nombre, cambiar := range casos {
		cmd := ventaOffline()
		cambiar(&cmd)
		if _, err := s.RegistrarVentaSincronizada(ctx, cmd); err == nil {
			t.Errorf("%s: sin error", nombre)
		} else if _, ok := errores.Como(err); !ok {
			t.Errorf("%s: error no tipado %v", nombre, err)
		}
	}
	if len(repo.ventas) != 0 {
		t.Errorf("se guardaron %d ventas", len(repo.ventas))
	}
}

func TestVentaSincronizadaConRelojAdelantado(t *testing.T) {
	s, repo, _ := entorno()
	cmd := ventaOffline()
	cmd.Fecha = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := s.RegistrarVentaSincronizada(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if got := repo.ventas[idOffline].Fecha; !got.Equal(time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)) {
		t.Errorf("fecha = %v, want la del servidor", got)
	}
}
