package db

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"

	"magui/internal/kernel/errores"
)

func TestTraducirConcurrencia(t *testing.T) {
	for _, n := range []uint16{1213, 1205} {
		err := TraducirConcurrencia(fmt.Errorf("insertar: %w", &mysql.MySQLError{Number: n}))
		if e, ok := errores.Como(err); !ok || e.Codigo != "operacion_concurrente" || e.Tipo != errores.TipoConflicto {
			t.Errorf("%d: %v", n, err)
		}
	}
	otro := &mysql.MySQLError{Number: 1062}
	if err := TraducirConcurrencia(otro); !errors.Is(err, otro) {
		t.Errorf("otro error debería pasar igual: %v", err)
	}
	if TraducirConcurrencia(nil) != nil {
		t.Error("nil debería seguir siendo nil")
	}
}
