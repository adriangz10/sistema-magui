package domain

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"magui/internal/kernel/errores"
)

func ticketValido() Ticket {
	return Ticket{Habilitado: true, ImprimirAlCobrar: true, AnchoPapel: 58, MostrarLogo: true}
}

func TestAplicarNormalizaTextos(t *testing.T) {
	n := Negocio{Nombre: "Magui"}
	err := n.Aplicar(Cambios{Nombre: "  Perfumería   Magui ", Ticket: ticketValido(),
		Datos: "Av. Siempreviva 742\r\n\r\n  Tel   1234  \n", Pie: "   "})
	if err != nil {
		t.Fatal(err)
	}
	if n.Nombre != "Perfumería Magui" || *n.Ticket.Datos != "Av. Siempreviva 742\nTel 1234" || n.Ticket.Pie != nil {
		t.Errorf("negocio = %+v datos=%q", n, *n.Ticket.Datos)
	}
}

func TestAplicarValida(t *testing.T) {
	casos := map[string]Cambios{
		"nombre":            {Nombre: "  ", Ticket: ticketValido()},
		"ticket.anchoPapel": {Nombre: "X", Ticket: Ticket{AnchoPapel: 70}},
		"ticket.datos":      {Nombre: "X", Ticket: ticketValido(), Datos: "1\n2\n3\n4\n5\n6\n7"},
		"ticket.pie":        {Nombre: "X", Ticket: ticketValido(), Pie: strings.Repeat("a", 201)},
	}
	for campo, c := range casos {
		n := Negocio{Nombre: "Magui"}
		e, ok := errores.Como(n.Aplicar(c))
		if !ok || e.Campos[campo] == "" {
			t.Errorf("%s: se esperaba error, vino %v", campo, e)
		}
		if n.Nombre != "Magui" {
			t.Errorf("%s: con error no debe cambiar nada", campo)
		}
	}
}

func TestSinTicketNoImprimeAlCobrar(t *testing.T) {
	n := Negocio{}
	_ = n.Aplicar(Cambios{Nombre: "X", Ticket: Ticket{Habilitado: false, ImprimirAlCobrar: true, AnchoPapel: 80}})
	if n.Ticket.ImprimirAlCobrar {
		t.Error("si el ticket está deshabilitado no se imprime al cobrar")
	}
}

// PNG mínimo válido (1x1).
var png1x1 = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func TestValidarLogo(t *testing.T) {
	if tipo, err := ValidarLogo(png1x1); err != nil || tipo != "image/png" {
		t.Errorf("png: %q %v", tipo, err)
	}
	if _, err := ValidarLogo([]byte(`<svg onload="alert(1)"></svg>`)); !errors.Is(err, ErrLogoTipo) {
		t.Errorf("svg: %v", err)
	}
	if _, err := ValidarLogo(nil); !errors.Is(err, ErrLogoVacio) {
		t.Errorf("vacío: %v", err)
	}
	grande := append(bytes.Clone(png1x1), make([]byte, TamanoMaximoLogo)...)
	if _, err := ValidarLogo(grande); !errors.Is(err, ErrLogoGrande) {
		t.Errorf("grande: %v", err)
	}
}
