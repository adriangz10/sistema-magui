package application

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"magui/internal/caja/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/kernel/tx"
)

// sesionesMem guarda cajas en memoria; los totales se fijan a mano por test.
type sesionesMem struct {
	m         map[string]domain.Sesion
	ediciones []EdicionCaja
	totales   domain.Totales
	ventas    int
	pedidos   [][2]time.Time // rangos pedidos a Totales
}

func (r *sesionesMem) Crear(_ context.Context, s domain.Sesion) error {
	for _, x := range r.m {
		if x.Abierta() {
			return domain.ErrCajaYaAbierta
		}
	}
	r.m[s.ID] = s
	return nil
}
func (r *sesionesMem) Obtener(_ context.Context, id string) (domain.Sesion, error) {
	s, ok := r.m[id]
	if !ok {
		return s, domain.ErrCajaNoEncontrada
	}
	return s, nil
}
func (r *sesionesMem) Abierta(context.Context) (*domain.Sesion, error) {
	for _, s := range r.m {
		if s.Abierta() {
			return &s, nil
		}
	}
	return nil, nil
}
func (r *sesionesMem) Actualizar(_ context.Context, s domain.Sesion, anterior int) error {
	if r.m[s.ID].Version != anterior {
		return domain.ErrVersionDesactualizada
	}
	r.m[s.ID] = s
	return nil
}
func (r *sesionesMem) Listar(_ context.Context, _ paginacion.Solicitud) ([]domain.Sesion, int, error) {
	var out []domain.Sesion
	for _, s := range r.m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AbiertaEn.After(out[j].AbiertaEn) })
	return out, len(out), nil
}
func (r *sesionesMem) GuardarEdicion(_ context.Context, e EdicionCaja) error {
	r.ediciones = append(r.ediciones, e)
	return nil
}
func (r *sesionesMem) Ediciones(_ context.Context, id string) ([]EdicionCaja, error) {
	var out []EdicionCaja
	for _, e := range r.ediciones {
		if e.SesionID == id {
			out = append(out, e)
		}
	}
	return out, nil
}
func (r *sesionesMem) Totales(_ context.Context, desde, hasta time.Time) (domain.Totales, int, error) {
	r.pedidos = append(r.pedidos, [2]time.Time{desde, hasta})
	return r.totales, r.ventas, nil
}

type relojMovil struct{ t time.Time }

func (r *relojMovil) Ahora() time.Time { return r.t }

var _ reloj.Reloj = (*relojMovil)(nil)

func entornoSesiones() (*ServicioSesiones, *sesionesMem, *relojMovil) {
	repo := &sesionesMem{m: map[string]domain.Sesion{}}
	rl := &relojMovil{t: ahora}
	return NuevoServicioSesiones(repo, tx.SinTransaccion{}, &idsSec{}, rl), repo, rl
}

func contado(ef string) map[domain.Metodo]dinero.Dinero {
	return map[domain.Metodo]dinero.Dinero{domain.Efectivo: dinero.MustParse(ef), domain.Transferencia: 0, domain.QR: 0, domain.Tarjeta: 0}
}

func TestAbrirCerrarCaja(t *testing.T) {
	s, repo, rl := entornoSesiones()
	if err := s.ExigirAbierta(ctx); !errors.Is(err, domain.ErrCajaCerrada) {
		t.Fatalf("sin caja: %v", err)
	}
	if act, _ := s.Actual(ctx); act != nil {
		t.Fatal("no debería haber caja actual")
	}

	ab, err := s.Abrir(ctx, dinero.MustParse("10000"), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Abrir(ctx, 0, ""); !errors.Is(err, domain.ErrCajaYaAbierta) {
		t.Errorf("segunda apertura: %v", err)
	}
	if err := s.ExigirAbierta(ctx); err != nil {
		t.Errorf("con caja abierta: %v", err)
	}

	// En vivo: el arqueo muestra el esperado sin contado.
	repo.totales = domain.Totales{domain.Efectivo: {Ventas: dinero.MustParse("5000"), Gastos: dinero.MustParse("1000")}}
	repo.ventas = 3
	rl.t = ahora.Add(4 * time.Hour)
	act, err := s.Actual(ctx)
	if err != nil || act == nil {
		t.Fatal(err)
	}
	if act.Arqueo[0].Esperado.String() != "14000.00" || act.Arqueo[0].Contado != nil || act.Diferencia != nil || act.CantidadVentas != 3 {
		t.Errorf("actual = %+v", act)
	}

	rl.t = ahora.Add(8 * time.Hour)
	if _, err := s.Cerrar(ctx, ab.ID, ComandoCierre{Version: 99, Contado: contado("13500")}); !errors.Is(err, domain.ErrVersionDesactualizada) {
		t.Errorf("versión vieja: %v", err)
	}
	ce, err := s.Cerrar(ctx, ab.ID, ComandoCierre{Version: ab.Version, Contado: contado("13500"), Notas: "faltó cambio"})
	if err != nil {
		t.Fatal(err)
	}
	if ce.Abierta || ce.Diferencia.String() != "-500.00" || ce.TotalContado.String() != "13500.00" || ce.Editada {
		t.Errorf("cierre = %+v", ce)
	}
	ult := repo.pedidos[len(repo.pedidos)-1]
	if !ult[0].Equal(ahora) || !ult[1].Equal(rl.t) {
		t.Errorf("período del cierre = %v", ult)
	}
	if err := s.ExigirAbierta(ctx); !errors.Is(err, domain.ErrCajaCerrada) {
		t.Errorf("tras cerrar: %v", err)
	}
	if _, err := s.Cerrar(ctx, ab.ID, ComandoCierre{Version: ce.Version, Contado: contado("1")}); !errors.Is(err, domain.ErrCajaYaCerrada) {
		t.Errorf("cerrar dos veces: %v", err)
	}
}

func TestEditarCajaGuardaHistorial(t *testing.T) {
	s, repo, rl := entornoSesiones()
	ab, _ := s.Abrir(ctx, dinero.MustParse("1000"), "")
	repo.totales = domain.Totales{domain.Efectivo: {Ventas: dinero.MustParse("500")}}
	rl.t = ahora.Add(time.Hour)
	ce, _ := s.Cerrar(ctx, ab.ID, ComandoCierre{Version: ab.Version, Contado: contado("1400")})

	rl.t = ahora.Add(48 * time.Hour)
	det, err := s.Editar(ctx, ab.ID, ComandoEdicion{Version: ce.Version, MontoInicial: dinero.MustParse("900"),
		Contado: contado("1400"), Motivo: "  el inicial era 900 "})
	if err != nil {
		t.Fatal(err)
	}
	if !det.Editada || det.Diferencia.String() != "0.00" || len(det.Ediciones) != 1 {
		t.Fatalf("detalle = %+v", det)
	}
	e := det.Ediciones[0]
	if e.Motivo != "el inicial era 900" || e.Version != 3 || !e.Fecha.Equal(rl.t) || len(e.Cambios) != 2 {
		t.Errorf("edición = %+v", e)
	}
	// Se recalcula con el período original de la caja, no hasta hoy.
	ult := repo.pedidos[len(repo.pedidos)-1]
	if !ult[1].Equal(ahora.Add(time.Hour)) {
		t.Errorf("recalculó hasta %v", ult[1])
	}
	if _, err := s.Editar(ctx, ab.ID, ComandoEdicion{Version: ce.Version, MontoInicial: 0, Contado: contado("0"), Motivo: "x"}); !errors.Is(err, domain.ErrVersionDesactualizada) {
		t.Errorf("edición con versión vieja: %v", err)
	}

	lista, err := s.Listar(ctx, paginacion.Solicitud{Pagina: 1, Tamano: 10})
	if err != nil || lista.Total != 1 || lista.Items[0].ID != ab.ID {
		t.Errorf("listado = %+v %v", lista, err)
	}
}
