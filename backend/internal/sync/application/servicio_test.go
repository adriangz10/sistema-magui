package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"magui/internal/kernel/errores"
	"magui/internal/kernel/reloj"
)

type ventasFalsas map[string]error // id -> error a devolver ("ya" = ya existía)

func (v ventasFalsas) RegistrarOffline(_ context.Context, in VentaOffline) (bool, error) {
	err := v[in.ID]
	if err != nil && err.Error() == "ya" {
		return true, nil
	}
	return false, err
}

type catalogoVacio struct{}

func (catalogoVacio) Copia(context.Context) ([]ProductoCopia, []ComponenteCopia, error) {
	return nil, nil, nil
}

func TestRecibirVentas(t *testing.T) {
	falsas := ventasFalsas{
		"b": errors.New("ya"),
		"c": errores.Conflicto("producto_no_encontrado", "Uno de los productos no existe"),
	}
	s := NuevoServicio(catalogoVacio{}, falsas, reloj.Fijo{T: time.Now()})
	res, err := s.RecibirVentas(context.Background(), []VentaOffline{{ID: "a"}, {ID: "b"}, {ID: "c"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []EstadoEnvio{Registrada, YaExistia, Rechazada}
	for i, w := range want {
		if res.Resultados[i].Estado != w {
			t.Errorf("%d: %s, want %s", i, res.Resultados[i].Estado, w)
		}
	}
	if res.Resultados[2].Error.Code != "producto_no_encontrado" {
		t.Errorf("error = %+v", res.Resultados[2].Error)
	}

	// Una falla del servidor (no es un rechazo) corta el envío: el escritorio reintenta todo.
	falsas["a"] = errors.New("se cayó la base")
	if _, err := s.RecibirVentas(context.Background(), []VentaOffline{{ID: "a"}}); err == nil {
		t.Error("una falla del servidor debe devolver error")
	}
}

// Un choque de bloqueos (operacion_concurrente) es transitorio: no puede marcar la venta como
// rechazada para siempre; el envío falla y el escritorio reintenta toda la tanda.
func TestRecibirVentasOperacionConcurrenteNoRechaza(t *testing.T) {
	falsas := ventasFalsas{"b": errores.Conflicto("operacion_concurrente", "Otra operación se cruzó con esta, intentá de nuevo.")}
	s := NuevoServicio(catalogoVacio{}, falsas, reloj.Fijo{T: time.Now()})
	res, err := s.RecibirVentas(context.Background(), []VentaOffline{{ID: "a"}, {ID: "b"}, {ID: "c"}})
	if err == nil {
		t.Fatalf("debería devolver error para que se reintente la tanda, devolvió %+v", res)
	}
	if e, ok := errores.Como(err); !ok || e.Codigo != "operacion_concurrente" {
		t.Errorf("err = %v", err)
	}
	// Reintento: sin el choque, todas se registran (y ninguna quedó rechazada).
	delete(falsas, "b")
	res, err = s.RecibirVentas(context.Background(), []VentaOffline{{ID: "a"}, {ID: "b"}, {ID: "c"}})
	if err != nil || len(res.Resultados) != 3 {
		t.Fatalf("reintento: %+v %v", res, err)
	}
	for _, r := range res.Resultados {
		if r.Estado == Rechazada {
			t.Errorf("%s quedó rechazada", r.ID)
		}
	}
}

func TestCopiaVaciaDevuelveListas(t *testing.T) {
	s := NuevoServicio(catalogoVacio{}, ventasFalsas{}, reloj.Fijo{T: time.Now()})
	c, _ := s.CopiaCatalogo(context.Background())
	if c.Productos == nil || c.Componentes == nil {
		t.Error("las listas vacías deben ser [] y no null")
	}
}
