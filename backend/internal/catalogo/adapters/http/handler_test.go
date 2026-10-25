package http

import (
	"net/http/httptest"
	"testing"

	"magui/internal/catalogo/application"
	"magui/internal/catalogo/domain"
	"magui/internal/kernel/errores"
)

func TestConsultaDesdeURL(t *testing.T) {
	c, err := consultaDesdeURL(httptest.NewRequest("GET", "/productos?q=kaiak&tipo=combo&estado=TODOS&orden=stock&stockBajo=true", nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.Texto != "kaiak" || c.Tipo != domain.TipoCombo || c.Estado != application.EstadoTodos ||
		c.Orden != application.OrdenStock || !c.StockBajo {
		t.Errorf("consulta = %+v", c)
	}
	_, err = consultaDesdeURL(httptest.NewRequest("GET", "/productos?tipo=x&estado=y&orden=z", nil))
	e, _ := errores.Como(err)
	if e == nil || len(e.Campos) != 3 {
		t.Errorf("valores inválidos: %v", err)
	}
}

func TestAjusteReqComando(t *testing.T) {
	cmd, err := ajusteReq{Tipo: "porcentaje", Valor: "10,5"}.comando("cat")
	if err != nil || cmd.Ajuste.Tipo != domain.AjustePorcentaje || cmd.Ajuste.Porcentaje != 1050 || !cmd.IncluirSubcategorias {
		t.Errorf("porcentaje: %+v %v", cmd, err)
	}
	no := false
	cmd, _ = ajusteReq{Tipo: "MONTO", Valor: "-200", IncluirSubcategorias: &no}.comando("cat")
	if cmd.Ajuste.Monto.String() != "-200.00" || cmd.IncluirSubcategorias {
		t.Errorf("monto: %+v", cmd)
	}
	if _, err := (ajusteReq{Tipo: "MONTO", Valor: "diez"}).comando("cat"); err == nil {
		t.Error("valor inválido debería fallar")
	}
}
