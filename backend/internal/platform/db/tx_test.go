package db

import (
	"fmt"

	"github.com/go-sql-driver/mysql"

	"context"
	"database/sql"
	"errors"
	"magui/internal/kernel/errores"
	"path/filepath"
	"testing"
)

func baseSQLite(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "tx.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	mustExec(t, conn, `CREATE TABLE t (v INTEGER)`)
	return conn
}

func contar(t *testing.T, conn *sql.DB) int {
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestTransaccionConfirma(t *testing.T) {
	conn := baseSQLite(t)
	tr := Transactor{DB: conn}
	err := tr.EnTransaccion(context.Background(), func(ctx context.Context) error {
		_, err := Conn(ctx, conn).ExecContext(ctx, `INSERT INTO t VALUES (1)`)
		return err
	})
	if err != nil || contar(t, conn) != 1 {
		t.Fatalf("err=%v filas=%d", err, contar(t, conn))
	}
}

func TestTransaccionRevierteAnidada(t *testing.T) {
	conn := baseSQLite(t)
	tr := Transactor{DB: conn}
	boom := errors.New("boom")
	err := tr.EnTransaccion(context.Background(), func(ctx context.Context) error {
		if _, err := Conn(ctx, conn).ExecContext(ctx, `INSERT INTO t VALUES (1)`); err != nil {
			return err
		}
		// La anidada reutiliza la misma transacción
		return tr.EnTransaccion(ctx, func(ctx context.Context) error {
			if _, err := Conn(ctx, conn).ExecContext(ctx, `INSERT INTO t VALUES (2)`); err != nil {
				return err
			}
			return boom
		})
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if n := contar(t, conn); n != 0 {
		t.Fatalf("debería revertir todo, quedaron %d filas", n)
	}
}

func TestReintentaEnConcurrencia(t *testing.T) {
	original := esperarReintento
	esperarReintento = func(context.Context, int) {}
	t.Cleanup(func() { esperarReintento = original })
	conn := baseSQLite(t)
	tr := Transactor{DB: conn}

	// Falla dos veces con deadlock y la tercera funciona: se confirma una sola vez.
	llamadas := 0
	err := tr.EnTransaccion(context.Background(), func(ctx context.Context) error {
		llamadas++
		if _, err := Conn(ctx, conn).ExecContext(ctx, `INSERT INTO t VALUES (1)`); err != nil {
			return err
		}
		if llamadas < 3 {
			return fmt.Errorf("venta: %w", &mysql.MySQLError{Number: 1213})
		}
		return nil
	})
	if err != nil || llamadas != 3 || contar(t, conn) != 1 {
		t.Fatalf("err=%v llamadas=%d filas=%d (los intentos fallidos deben revertirse)", err, llamadas, contar(t, conn))
	}

	// Deadlock permanente: MaxIntentos intentos y 409 operacion_concurrente.
	llamadas = 0
	err = tr.EnTransaccion(context.Background(), func(ctx context.Context) error {
		llamadas++
		return &mysql.MySQLError{Number: 1213}
	})
	if e, ok := errores.Como(err); !ok || e.Codigo != "operacion_concurrente" || llamadas != MaxIntentos {
		t.Fatalf("err=%v llamadas=%d", err, llamadas)
	}

	// Espera de bloqueo agotada (1205): NO se reintenta (cada intento puede tardar todo el lock wait),
	// se traduce a 409 de una vez.
	llamadas = 0
	err = tr.EnTransaccion(context.Background(), func(ctx context.Context) error {
		llamadas++
		return &mysql.MySQLError{Number: 1205}
	})
	if e, ok := errores.Como(err); !ok || e.Codigo != "operacion_concurrente" || llamadas != 1 {
		t.Fatalf("1205: err=%v llamadas=%d", err, llamadas)
	}

	// El ErrOperacionConcurrente de inventario (lotes que cambiaron durante el bloqueo) sí se reintenta.
	llamadas = 0
	err = tr.EnTransaccion(context.Background(), func(ctx context.Context) error {
		llamadas++
		if llamadas < 2 {
			return ErrOperacionConcurrente
		}
		return nil
	})
	if err != nil || llamadas != 2 {
		t.Fatalf("lotes cambiados: err=%v llamadas=%d", err, llamadas)
	}

	// Otros errores no se reintentan.
	llamadas = 0
	boom := errors.New("boom")
	if err := tr.EnTransaccion(context.Background(), func(context.Context) error { llamadas++; return boom }); !errors.Is(err, boom) || llamadas != 1 {
		t.Fatalf("err=%v llamadas=%d", err, llamadas)
	}

	// Una transacción anidada no reintenta por su cuenta: el reintento es del nivel externo.
	externas, internas := 0, 0
	err = tr.EnTransaccion(context.Background(), func(ctx context.Context) error {
		externas++
		return tr.EnTransaccion(ctx, func(context.Context) error {
			internas++
			if externas < 2 {
				return &mysql.MySQLError{Number: 1213}
			}
			return nil
		})
	})
	if err != nil || externas != 2 || internas != 2 {
		t.Fatalf("err=%v externas=%d internas=%d", err, externas, internas)
	}
}

func TestTraduceConcurrenciaTambienSiFallaElRollback(t *testing.T) {
	// El error de rollback va unido al original: igual debe reconocerse el 1213.
	err := errors.Join(&mysql.MySQLError{Number: 1213}, fmt.Errorf("revertir transacción: %w", errors.New("conexión cerrada")))
	if !EsConcurrencia(err) || TraducirConcurrencia(err) != ErrOperacionConcurrente {
		t.Error("debería traducir aunque falle el rollback")
	}
}

func TestOpcionesDeAislamiento(t *testing.T) {
	if o := (Transactor{}).opciones(); o != nil {
		t.Error("sin Isolation se usa el del driver")
	}
	if o := NuevoTransactorMySQL(nil).opciones(); o == nil || o.Isolation != sql.LevelReadCommitted {
		t.Errorf("MySQL debe correr en READ COMMITTED: %+v", o)
	}
}
