package http_test

import (
	"strings"
	"testing"
)

// US-001c por HTTP. El fake de editar_http_test.go tiene la compra c1 con el renglón i1 (4 unidades,
// de las que ya salieron 3: stock del lote 1) y acepta cualquier producto.
func TestPutAgregarYQuitarRenglones(t *testing.T) {
	i1 := `{"id":"i1","productoId":"p1","cantidad":4,"costoUnitario":"100"}`
	nuevo := `{"productoId":"p2","cantidad":2,"costoUnitario":"15000","precioVentaNuevo":"24000","vencimiento":"2028-01-31"}`
	casos := []struct {
		nombre, items string
		status        int
		contiene      []string
	}{
		{"agregar", `[` + i1 + `,` + nuevo + `]`, 200, []string{`"version":`}},
		{"quitar con consumo", `[` + nuevo + `]`, 409, []string{"compra_consumida", "quitados[i1]", "ya salieron 3 de las 4 unidades"}},
		{"quitar todos", `[]`, 422, []string{"datos_invalidos", "Agregá al menos un producto"}},
		{"nuevo inválido", `[` + i1 + `,{"productoId":"p2","cantidad":0,"costoUnitario":"1"}]`, 422, []string{"items[1].cantidad"}},
		{"id repetido", `[` + i1 + `,` + i1 + `]`, 422, []string{"items[1].id"}},
	}
	for _, c := range casos {
		srv := servidor()
		status, body := put(t, srv.URL, `{"fecha":"2026-10-20","version":1,"items":`+c.items+`}`)
		srv.Close()
		if status != c.status {
			t.Errorf("%s: status %d (esperado %d) %s", c.nombre, status, c.status, body)
		}
		for _, s := range c.contiene {
			if !strings.Contains(body, s) {
				t.Errorf("%s: debería contener %q: %s", c.nombre, s, body)
			}
		}
	}
}

func TestPutNoQuedaElCodigoViejoDeRenglonesNoDisponibles(t *testing.T) {
	srv := servidor()
	defer srv.Close()
	_, body := put(t, srv.URL, `{"fecha":"2026-10-20","version":1,"items":[{"productoId":"p2","cantidad":1,"costoUnitario":"1"}]}`)
	for _, viejo := range []string{"renglones_nuevos_no_disponibles", "quitar_renglones_no_disponible"} {
		if strings.Contains(body, viejo) {
			t.Errorf("%s no debería existir más: %s", viejo, body)
		}
	}
}
