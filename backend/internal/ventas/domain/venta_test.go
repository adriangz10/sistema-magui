package domain

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
)

func d(s string) dinero.Dinero { return dinero.MustParse(s) }

func ids() func() string {
	n := 0
	return func() string { n++; return fmt.Sprintf("id-%d", n) }
}

func codigo(err error) string {
	if e, ok := errores.Como(err); ok {
		return e.Codigo
	}
	return ""
}

func TestCalcularPagos(t *testing.T) {
	total := d("10000")
	casos := []struct {
		nombre string
		pagos  []PagoIngresado
		codigo string
		vuelto string
		falta  string
	}{
		{"exacto en efectivo", []PagoIngresado{{Efectivo, d("10000")}}, "", "0.00", "0.00"},
		{"efectivo con vuelto", []PagoIngresado{{Efectivo, d("20000")}}, "", "10000.00", "0.00"},
		{"combinado con vuelto", []PagoIngresado{{Transferencia, d("6000")}, {Efectivo, d("5000")}}, "", "1000.00", "0.00"},
		{"dos tarjetas", []PagoIngresado{{Tarjeta, d("4000")}, {Tarjeta, d("6000")}}, "", "0.00", "0.00"},
		{"falta", []PagoIngresado{{QR, d("3000")}, {Efectivo, d("2000")}}, "pagos_insuficientes", "0.00", "5000.00"},
		{"sin pagos", nil, "pagos_insuficientes", "0.00", "10000.00"},
		{"tarjeta de más", []PagoIngresado{{Tarjeta, d("10001")}}, "pagos_excedidos", "", ""},
		{"efectivo sobrante", []PagoIngresado{{QR, d("10000")}, {Efectivo, d("500")}}, "pagos_excedidos", "", ""},
		{"dos efectivos", []PagoIngresado{{Efectivo, d("5000")}, {Efectivo, d("5000")}}, "datos_invalidos", "", ""},
		{"monto cero", []PagoIngresado{{QR, d("0")}}, "datos_invalidos", "", ""},
		{"método raro", []PagoIngresado{{"CHEQUE", d("10000")}}, "datos_invalidos", "", ""},
	}
	for _, c := range casos {
		pagos, res, err := CalcularPagos(total, c.pagos, ids())
		if codigo(err) != c.codigo {
			t.Errorf("%s: código %q, want %q (%v)", c.nombre, codigo(err), c.codigo, err)
			continue
		}
		if c.vuelto != "" && (res.Vuelto.String() != c.vuelto || res.Falta.String() != c.falta) {
			t.Errorf("%s: vuelto %s falta %s", c.nombre, res.Vuelto, res.Falta)
		}
		if err == nil {
			aplicado := dinero.Cero
			for _, p := range pagos {
				aplicado = aplicado.Sumar(p.Monto)
			}
			if aplicado != total {
				t.Errorf("%s: aplicado %s != total", c.nombre, aplicado)
			}
		}
	}
}

func TestPagoEfectivoGuardaRecibidoYVuelto(t *testing.T) {
	pagos, _, err := CalcularPagos(d("10000"), []PagoIngresado{{Efectivo, d("12000")}, {QR, d("3000")}}, ids())
	if err != nil {
		t.Fatal(err)
	}
	ef := pagos[len(pagos)-1]
	if ef.Metodo != Efectivo || ef.Monto.String() != "7000.00" || ef.MontoRecibido.String() != "12000.00" || ef.Vuelto.String() != "5000.00" {
		t.Errorf("efectivo = %+v", ef)
	}
	if pagos[0].MontoRecibido != nil || pagos[0].Vuelto != nil {
		t.Error("los pagos que no son efectivo no tienen recibido ni vuelto")
	}
}

func TestFaltaInformaMonto(t *testing.T) {
	_, _, err := CalcularPagos(d("10000"), []PagoIngresado{{QR, d("2500.50")}}, ids())
	if err == nil || err.Error() != "pagos_insuficientes: Faltan $7499.50 para completar el cobro" {
		t.Errorf("err = %v", err)
	}
}

func TestNuevaVenta(t *testing.T) {
	ahora := time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)
	lineas := []LineaVenta{{"kaiak", 1, d("59200")}, {"jabon", 2, d("5500")}, {"kaiak", 1, d("59200")}}
	v, res, err := NuevaVenta("v1", lineas, []PagoIngresado{{Efectivo, d("150000")}}, OrigenWeb, ahora, ids())
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Items) != 2 || v.Items[0].Cantidad != 2 || v.Total.String() != "129400.00" || res.Vuelto.String() != "20600.00" {
		t.Errorf("venta = %+v, resumen = %+v", v, res)
	}
	if v.Estado != Confirmada || !v.Fecha.Equal(ahora) {
		t.Errorf("estado/fecha = %s %v", v.Estado, v.Fecha)
	}
	if _, _, err := NuevaVenta("v2", nil, nil, OrigenWeb, ahora, ids()); codigo(err) != "datos_invalidos" {
		t.Errorf("sin items: %v", err)
	}
	if _, _, err := NuevaVenta("v3", []LineaVenta{{"x", 0, d("1")}}, nil, OrigenWeb, ahora, ids()); codigo(err) != "datos_invalidos" {
		t.Errorf("cantidad cero: %v", err)
	}
}

func TestCostos(t *testing.T) {
	c := d("100")
	v := Venta{Items: []Item{{Cantidad: 3, CostoTotal: &c}, {Cantidad: 1}}}
	total, completo := v.CostoTotal()
	if total.String() != "100.00" || completo {
		t.Errorf("costo = %s completo %v", total, completo)
	}
	if u := v.Items[0].CostoUnitario(); u.String() != "33.33" {
		t.Errorf("costo unitario = %s", u)
	}
	if v.Items[1].CostoUnitario() != nil {
		t.Error("sin costo debería ser nil")
	}
}

func TestAnular(t *testing.T) {
	v := Venta{Estado: Confirmada}
	ahora := time.Now()
	if err := v.Anular(" ", ahora); codigo(err) != "datos_invalidos" {
		t.Errorf("sin motivo: %v", err)
	}
	if err := v.Anular("Devolución", ahora); err != nil || v.Estado != Anulada || *v.MotivoAnulacion != "Devolución" {
		t.Fatalf("anular: %v %+v", err, v)
	}
	if err := v.Anular("otra vez", ahora); !errors.Is(err, ErrVentaYaAnulada) {
		t.Errorf("doble anulación: %v", err)
	}
}

func TestParse(t *testing.T) {
	if m, ok := ParseMetodo(" qr "); !ok || m != QR {
		t.Error("ParseMetodo")
	}
	if o, ok := ParseOrigen(""); !ok || o != OrigenWeb {
		t.Error("ParseOrigen vacío")
	}
	if _, ok := ParseOrigen("fax"); ok {
		t.Error("ParseOrigen inválido")
	}
}
