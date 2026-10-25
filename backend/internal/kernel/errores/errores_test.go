package errores

import (
	"errors"
	"fmt"
	"testing"
)

var errSinStock = Conflicto("stock_insuficiente", "No hay stock")

func TestIsPorCodigo(t *testing.T) {
	err := fmt.Errorf("vender: %w", Conflicto("stock_insuficiente", "Quedan 2 unidades"))
	if !errors.Is(err, errSinStock) {
		t.Fatal("errors.Is debería comparar por código")
	}
	if errors.Is(err, NoEncontrado("producto_no_encontrado", "")) {
		t.Fatal("códigos distintos no deberían coincidir")
	}
	e, ok := Como(err)
	if !ok || e.Tipo != TipoConflicto || e.Mensaje != "Quedan 2 unidades" {
		t.Fatalf("Como = %+v, %v", e, ok)
	}
}

func TestValidacion(t *testing.T) {
	var v Validacion
	if v.Err() != nil {
		t.Fatal("sin problemas debería ser nil")
	}
	v.Si(true, "nombre", "Obligatorio")
	v.Si(true, "nombre", "Otro mensaje")
	v.Si(false, "precio", "No debería aparecer")
	e, ok := Como(v.Err())
	if !ok || e.Codigo != "datos_invalidos" || len(e.Campos) != 1 || e.Campos["nombre"] != "Obligatorio" {
		t.Fatalf("validación = %+v", e)
	}
}
