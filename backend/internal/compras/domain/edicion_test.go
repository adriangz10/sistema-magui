package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
)

var ahora = time.Date(2026, 10, 27, 16, 0, 0, 0, time.UTC) // 13:00 en Argentina

func f(s string) *fecha.Fecha { x, _ := fecha.Parse(s); return &x }
func sp(s string) *string     { return &s }

func base() Compra {
	return Compra{ID: "c1", Fecha: time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC), EmpresaID: sp("n"), Empresa: "Natura",
		Items: []Item{{ID: "i", ProductoID: "p", Cantidad: 1, CostoUnitario: dinero.MustParse("10")}}, Version: 1}
}

func cab() CambiosCabecera {
	return CambiosCabecera{Version: 1, Dia: f("2026-10-20"), EmpresaID: "n", EmpresaNombre: "Natura"}
}

func campos(err error) map[string]string {
	e, _ := errores.Como(err)
	if e == nil {
		return nil
	}
	return e.Campos
}

func TestPlanMismoDiaConservaInstante(t *testing.T) {
	// 22:30 en Argentina del 20 = 01:30 UTC del 21: el día se compara en Argentina.
	c := base()
	c.Fecha = time.Date(2026, 10, 21, 1, 30, 0, 0, time.UTC)
	p, err := PlanificarEdicion(c, cab(), ahora)
	if err != nil || !p.SinCambios {
		t.Fatalf("plan = %+v, %v", p, err)
	}
}

func TestPlanFechas(t *testing.T) {
	c := cab()
	c.Dia = f("2026-10-27") // hoy
	p, _ := PlanificarEdicion(base(), c, ahora)
	if !p.Compra.Fecha.Equal(ahora) {
		t.Errorf("hoy debería usar ahora: %v", p.Compra.Fecha)
	}
	c.Dia = f("2026-10-19")
	p, _ = PlanificarEdicion(base(), c, ahora)
	if !p.Compra.Fecha.Equal(time.Date(2026, 10, 19, 15, 0, 0, 0, time.UTC)) {
		t.Errorf("otro día = mediodía AR: %v", p.Compra.Fecha)
	}
	c.Dia = f("2026-10-28")
	if _, err := PlanificarEdicion(base(), c, ahora); campos(err)["fecha"] != "La fecha no puede ser futura" {
		t.Errorf("futura: %v", err)
	}
	c.Dia, c.Instante = nil, nil
	if _, err := PlanificarEdicion(base(), c, ahora); campos(err)["fecha"] == "" {
		t.Errorf("sin fecha: %v", err)
	}
	fut := ahora.Add(2 * time.Hour)
	c.Instante = &fut
	if _, err := PlanificarEdicion(base(), c, ahora); campos(err)["fecha"] == "" {
		t.Errorf("instante futuro: %v", err)
	}
}

func TestPlanValidaciones(t *testing.T) {
	c := cab()
	c.Notas = strings.Repeat("a", 501)
	c.Motivo = strings.Repeat("b", 256)
	c.Origen = "FAX"
	c.Version = 0
	cs := campos(ValidarEdicion(c, ahora))
	for _, k := range []string{"notas", "motivo", "origen", "version"} {
		if cs[k] == "" {
			t.Errorf("falta error en %s: %v", k, cs)
		}
	}
	// Lista vacía: "datos_invalidos" (al menos un renglón).
	c = cab()
	c.Renglones = &[]RenglonEdicion{}
	if campos(ValidarEdicion(c, ahora))["items"] == "" {
		t.Error("lista vacía debería fallar")
	}
}

func TestPlanSinCambiosIgnoraMotivoYOrigen(t *testing.T) {
	c := cab()
	c.Motivo, c.Origen = "algo", "DESKTOP"
	p, err := PlanificarEdicion(base(), c, ahora)
	if err != nil || !p.SinCambios {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestPlanQuitarEmpresa(t *testing.T) {
	c := cab()
	c.EmpresaID, c.EmpresaNombre = "", ""
	p, _ := PlanificarEdicion(base(), c, ahora)
	if p.Compra.EmpresaID != nil || len(p.Cambios.Cabecera) != 1 || p.Cambios.Cabecera[0].Antes != "Natura" || p.Cambios.Cabecera[0].Despues != nil {
		t.Errorf("%+v", p.Cambios)
	}
}

func TestDocumentoDeCambiosGolden(t *testing.T) {
	c := CambiosCabecera{Version: 1, Dia: f("2026-10-19"), EmpresaID: "avon", EmpresaNombre: "Avon", Notas: "Pedido ciclo 15", Motivo: "Error de carga"}
	p, err := PlanificarEdicion(base(), c, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if p.Compra.Version != 2 || !p.Compra.UpdatedAt.Equal(ahora) {
		t.Errorf("compra = %+v", p.Compra)
	}
	b, _ := json.Marshal(p.Cambios)
	want := `{"v":1,"cabecera":[` +
		`{"campo":"fecha","antes":"2026-10-20T15:00:00.000Z","despues":"2026-10-19T15:00:00.000Z"},` +
		`{"campo":"empresa","antes":"Natura","despues":"Avon","antesId":"n","despuesId":"avon"},` +
		`{"campo":"notas","antes":null,"despues":"Pedido ciclo 15"}],"renglones":[]}`
	if string(b) != want {
		t.Errorf("json =\n%s\nquería\n%s", b, want)
	}
	if _, err := LeerCambios(b); err != nil {
		t.Error(err)
	}
	if _, err := LeerCambios([]byte(`{"v":2}`)); err == nil {
		t.Error("v desconocida debería fallar")
	}
}
