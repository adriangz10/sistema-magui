package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

// Ejecutor es lo común entre *sql.DB y *sql.Tx. Los repositorios trabajan contra esto.
type Ejecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type claveTx struct{}

// MaxIntentos es la cantidad de veces que se ejecuta una transacción que MySQL aborta por
// deadlock (1213) antes de rendirse (ADR 0002).
const MaxIntentos = 3

// esperarReintento hace una pausa corta con jitter antes de reintentar (reemplazable en tests).
var esperarReintento = func(ctx context.Context, intento int) {
	pausa := time.Duration(intento)*10*time.Millisecond + time.Duration(rand.IntN(20))*time.Millisecond
	select {
	case <-time.After(pausa):
	case <-ctx.Done():
	}
}

// Transactor implementa kernel/tx.Transactor sobre database/sql.
//
// Isolation fija el nivel de aislamiento de las transacciones (cero = el del driver). Para MySQL se
// construye con NuevoTransactorMySQL: READ COMMITTED, sin bloqueos de brecha (gap/next-key) y con
// lecturas que ven lo último confirmado; junto con los FOR UPDATE de los lotes (ADR 0002) alcanza
// para serializar ventas, ajustes y ediciones de compra. SQLite (escritorio) usa el del driver.
type Transactor struct {
	DB        *sql.DB
	Isolation sql.IsolationLevel
}

// NuevoTransactorMySQL arma el Transactor de las transacciones del servidor (READ COMMITTED).
func NuevoTransactorMySQL(conn *sql.DB) Transactor {
	return Transactor{DB: conn, Isolation: sql.LevelReadCommitted}
}

// EnTransaccion abre una transacción y la deja en el ctx. Si ya hay una en curso
// (llamadas anidadas), la reutiliza: todo se confirma o revierte junto.
//
// Si MySQL aborta la transacción por deadlock (1213), la reejecuta hasta MaxIntentos veces con una
// pausa corta; recién entonces devuelve 409 operacion_concurrente. La espera de bloqueo agotada
// (1205) NO se reintenta (cada intento puede tardar todo el lock wait): se traduce a 409 directo.
// Tampoco se reintenta nada que no sea un choque de bloqueos. Por
// eso `fn` debe poder ejecutarse más de una vez: sin efectos fuera de la base y sin acumular
// estado entre intentos. El reintento solo ocurre en la transacción más externa.
func (t Transactor) EnTransaccion(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := ctx.Value(claveTx{}).(*sql.Tx); ok {
		return fn(ctx)
	}
	var err error
	for intento := 1; ; intento++ {
		if err = t.unaVez(ctx, fn); err == nil {
			return nil
		}
		if !EsReintentable(err) || intento >= MaxIntentos || ctx.Err() != nil {
			return TraducirConcurrencia(err)
		}
		esperarReintento(ctx, intento)
	}
}

func (t Transactor) unaVez(ctx context.Context, fn func(context.Context) error) (err error) {
	tx, err := t.DB.BeginTx(ctx, t.opciones())
	if err != nil {
		return fmt.Errorf("iniciar transacción: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(context.WithValue(ctx, claveTx{}, tx)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("revertir transacción: %w", rbErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar transacción: %w", err)
	}
	return nil
}

func (t Transactor) opciones() *sql.TxOptions {
	if t.Isolation == sql.LevelDefault {
		return nil
	}
	return &sql.TxOptions{Isolation: t.Isolation}
}

// Conn devuelve la transacción en curso si la hay; si no, la conexión normal.
func Conn(ctx context.Context, conn *sql.DB) Ejecutor {
	if tx, ok := ctx.Value(claveTx{}).(*sql.Tx); ok {
		return tx
	}
	return conn
}
