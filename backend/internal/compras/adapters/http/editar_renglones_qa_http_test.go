package http_test

import (
	"strings"
	"testing"
)

// QA US-001b: contrato HTTP de los renglones. El fake de servidor() tiene un lote con stock 1 de 4
// (consumido 3): bajar a 3 se acepta, a 2 se rechaza.

func TestQAPutRenglonConsumoBorde(t *testing.T) {
	const pre = `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":`
	const pos = `,"costoUnitario":"100"}]}`
	casos := []struct {
		nombre, cant string
		status       int
		contiene     []string
	}{
		{"justo lo consumido", "3", 200, []string{`"version":2`}},
		{"consumido menos uno", "2", 409, []string{"compra_consumida", "items[0].cantidad", "salieron 3", "hasta 3 unidades"}},
		{"cantidad cero", "0", 422, []string{"items[0].cantidad"}},
		{"cantidad negativa", "-2", 422, []string{"items[0].cantidad"}},
		{"cantidad decimal", "1.5", 422, []string{"json_invalido"}},
	}
	for _, c := range casos {
		srv := servidor()
		status, body := put(t, srv.URL, pre+c.cant+pos)
		srv.Close()
		if status != c.status {
			t.Errorf("%s: status %d (esperado %d) %s", c.nombre, status, c.status, body)
		}
		for _, s := range c.contiene {
			if !strings.Contains(body, s) {
				t.Errorf("%s: la respuesta debería contener %q: %s", c.nombre, s, body)
			}
		}
	}
}

func TestQAPutRenglonValoresInvalidos(t *testing.T) {
	casos := []struct {
		nombre, body string
		status       int
		campo        string
	}{
		{"costo -1", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"-1"}]}`, 422, "items[0].costoUnitario"},
		{"precio 0", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100","precioVentaNuevo":"0"}]}`, 422, "items[0].precioVentaNuevo"},
		{"precio negativo", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100","precioVentaNuevo":"-5"}]}`, 422, "items[0].precioVentaNuevo"},
		{"costo con 3 decimales", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100.005"}]}`, 422, "json_invalido"},
		{"vencimiento no fecha", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100","vencimiento":"mañana"}]}`, 422, "json_invalido"},
		{"campo desconocido", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100","consumido":3}]}`, 422, "json_invalido"},
		{"sin id", `{"fecha":"2026-10-20","version":1,"items":[{"productoId":"p1","cantidad":4,"costoUnitario":"100"}]}`, 422, "renglones_nuevos_no_disponibles"},
		{"producto ajeno", `{"fecha":"2026-10-20","version":1,"items":[{"id":"i1","productoId":"p9","cantidad":4,"costoUnitario":"100"}]}`, 422, "items[0].productoId"},
	}
	for _, c := range casos {
		srv := servidor()
		status, resp := put(t, srv.URL, c.body)
		srv.Close()
		if status != c.status || !strings.Contains(resp, c.campo) {
			t.Errorf("%s: %d %s", c.nombre, status, resp)
		}
	}
}

// Los renglones repetidos o de otra compra no deben aplicarse.
func TestQAPutRenglonIdRepetidoYAjeno(t *testing.T) {
	srv := servidor()
	defer srv.Close()
	rep := `{"fecha":"2026-10-20","version":1,"items":[` +
		`{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100"},` +
		`{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100"}]}`
	if status, body := put(t, srv.URL, rep); status != 422 || !strings.Contains(body, "items[1].id") {
		t.Errorf("repetido: %d %s", status, body)
	}
	ajeno := `{"fecha":"2026-10-20","version":1,"items":[{"id":"zzz","productoId":"p1","cantidad":4,"costoUnitario":"100"}]}`
	if status, body := put(t, srv.URL, ajeno); status != 422 || !strings.Contains(body, "items[0].id") {
		t.Errorf("ajeno: %d %s", status, body)
	}
}
