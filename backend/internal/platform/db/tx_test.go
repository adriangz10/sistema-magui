package db

import (
	"context"
	"database/sql"
	"errors"
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
