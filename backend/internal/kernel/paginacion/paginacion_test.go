package paginacion

import (
	"encoding/json"
	"testing"
)

func TestNueva(t *testing.T) {
	casos := []struct{ p, t, wp, wt, off int }{
		{0, 0, 1, 10, 0},
		{3, 5, 3, 5, 10},
		{2, 50, 2, 10, 10}, // nunca más de 10
		{-1, -1, 1, 10, 0},
	}
	for _, c := range casos {
		s := Nueva(c.p, c.t)
		if s.Pagina != c.wp || s.Tamano != c.wt || s.Offset() != c.off {
			t.Errorf("Nueva(%d,%d) = %+v offset %d", c.p, c.t, s, s.Offset())
		}
	}
}

func TestNuevaPagina(t *testing.T) {
	p := NuevaPagina([]string{"a"}, Nueva(1, 10), 21)
	if p.TotalPages != 3 {
		t.Errorf("totalPages = %d", p.TotalPages)
	}
	vacia := NuevaPagina[string](nil, Nueva(1, 10), 0)
	b, _ := json.Marshal(vacia)
	if string(b) != `{"items":[],"page":1,"size":10,"total":0,"totalPages":0}` {
		t.Errorf("json = %s", b)
	}
}
