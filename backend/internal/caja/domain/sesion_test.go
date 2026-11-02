package domain

import (
	"errors"
	"testing"
	"time"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
)

var (
	t0 = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	m  = dinero.MustParse
)

func contadoTodo(ef, tr, qr, ta string) map[Metodo]dinero.Dinero {
	return map[Metodo]dinero.Dinero{Efectivo: m(ef), Transferencia: m(tr), QR: m(qr), Tarjeta: m(ta)}
}

func campoInvalido(t *testing.T, err error, campo string) {
	t.Helper()
	e, ok := errores.Como(err)
	if !ok || e.Campos[campo] == "" {
		t.Fatalf("se esperaba error en %q, vino %v", campo, err)
	}
}

func TestAbrirSesion(t *testing.T) {
	s, err := AbrirSesion("s1", m("10000"), "  turno mañana ", t0)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Abierta() || s.Version != 1 || *s.Notas != "turno mañana" || s.MontoInicial != m("10000") {
		t.Errorf("sesión = %+v", s)
	}
	_, err = AbrirSesion("s2", m("-1"), "", t0)
	campoInvalido(t, err, "montoInicial")
}

func TestCerrarConciliaCadaMedio(t *testing.T) {
	s, _ := AbrirSesion("s1", m("10000"), "", t0)
	tot := Totales{
		Efectivo:      {Ventas: m("5000"), Gastos: m("1000")},
		Transferencia: {Ventas: m("3000")},
		QR:            {Ventas: m("2000"), Gastos: m("500")},
	}
	if err := s.Cerrar(tot, contadoTodo("13500", "3000", "1600", "0"), "", t0.Add(8*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if s.Abierta() || s.Version != 2 {
		t.Fatalf("debería quedar cerrada en versión 2: %+v", s)
	}
	esperado := map[Metodo][2]string{ // esperado, diferencia
		Efectivo:      {"14000.00", "-500.00"}, // 10000 + 5000 − 1000; faltan 500
		Transferencia: {"3000.00", "0.00"},
		QR:            {"1500.00", "100.00"}, // sobran 100
		Tarjeta:       {"0.00", "0.00"},
	}
	for i, a := range s.Arqueo {
		if a.Metodo != Medios[i] {
			t.Errorf("orden del arqueo: %v en %d", a.Metodo, i)
		}
		if a.Esperado.String() != esperado[a.Metodo][0] || a.Diferencia().String() != esperado[a.Metodo][1] {
			t.Errorf("%s: esperado %s diferencia %s", a.Metodo, a.Esperado, a.Diferencia())
		}
	}
	if err := s.Cerrar(tot, contadoTodo("0", "0", "0", "0"), "", t0); !errors.Is(err, ErrCajaYaCerrada) {
		t.Errorf("cerrar dos veces: %v", err)
	}
}

func TestCerrarExigeContadoDeTodosLosMedios(t *testing.T) {
	s, _ := AbrirSesion("s1", 0, "", t0)
	c := contadoTodo("0", "0", "0", "0")
	delete(c, QR)
	campoInvalido(t, s.Cerrar(nil, c, "", t0), "contado.QR")
	c = contadoTodo("-1", "0", "0", "0")
	campoInvalido(t, s.Cerrar(nil, c, "", t0), "contado.EFECTIVO")
	if !s.Abierta() {
		t.Error("con error no debe cerrarse")
	}
}

func TestEditarCajaCerradaRecalculaYRegistraCambios(t *testing.T) {
	s, _ := AbrirSesion("s1", m("10000"), "", t0)
	tot := Totales{Efectivo: {Ventas: m("5000")}}
	_ = s.Cerrar(tot, contadoTodo("15000", "0", "0", "0"), "", t0.Add(time.Hour))

	// Después del cierre se anuló una venta de 1000 en efectivo: el esperado baja.
	tot2 := Totales{Efectivo: {Ventas: m("4000")}}
	cambios, err := s.Editar(Edicion{MontoInicial: m("10000"), Contado: contadoTodo("14000", "0", "0", "0"),
		Notas: "recontado", Motivo: " me  equivoqué al contar "}, tot2, t0.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if s.Version != 3 || s.Arqueo[0].Esperado != m("14000") || !s.Arqueo[0].Diferencia().EsCero() {
		t.Errorf("tras editar: %+v", s.Arqueo[0])
	}
	campos := map[string]Cambio{}
	for _, c := range cambios {
		campos[c.Campo] = c
	}
	if c := campos["contado.EFECTIVO"]; *c.Antes != "15000.00" || *c.Despues != "14000.00" {
		t.Errorf("contado: %+v", c)
	}
	if c := campos["esperado.EFECTIVO"]; *c.Antes != "15000.00" || *c.Despues != "14000.00" {
		t.Errorf("esperado: %+v", c)
	}
	if c := campos["notas"]; c.Antes != nil || *c.Despues != "recontado" {
		t.Errorf("notas: %+v", c)
	}
	if _, ok := campos["montoInicial"]; ok || len(cambios) != 3 {
		t.Errorf("solo deben figurar los campos que cambiaron: %+v", cambios)
	}
}

func TestEditarValidaMotivoYCambios(t *testing.T) {
	s, _ := AbrirSesion("s1", m("100"), "", t0)
	_, err := s.Editar(Edicion{MontoInicial: m("200")}, nil, t0)
	campoInvalido(t, err, "motivo")
	if _, err := s.Editar(Edicion{MontoInicial: m("100"), Motivo: "nada"}, nil, t0); !errors.Is(err, ErrSinCambios) {
		t.Errorf("sin cambios: %v", err)
	}
	// Caja abierta: se edita el monto inicial y se ignora el contado.
	cambios, err := s.Editar(Edicion{MontoInicial: m("200"), Motivo: "faltaba un billete"}, nil, t0)
	if err != nil || len(cambios) != 1 || s.MontoInicial != m("200") || s.Version != 2 || !s.Abierta() {
		t.Errorf("cambios=%+v err=%v sesión=%+v", cambios, err, s)
	}
}

func TestEditarSoloRecalculoNoAlcanza(t *testing.T) {
	s, _ := AbrirSesion("s1", 0, "", t0)
	_ = s.Cerrar(Totales{}, contadoTodo("0", "0", "0", "0"), "", t0)
	// Mismos datos pedidos, pero el esperado cambió por una venta nueva: no es una edición de la dueña.
	_, err := s.Editar(Edicion{Contado: contadoTodo("0", "0", "0", "0"), Motivo: "x"},
		Totales{Efectivo: {Ventas: m("10")}}, t0)
	if !errors.Is(err, ErrSinCambios) {
		t.Errorf("err = %v", err)
	}
}
