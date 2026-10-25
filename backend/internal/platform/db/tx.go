package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Ejecutor es lo común entre *sql.DB y *sql.Tx. Los repositorios trabajan contra esto.
type Ejecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type claveTx struct{}

// Transactor implementa kernel/tx.Transactor sobre database/sql.
type Transactor struct{ DB *sql.DB }

// EnTransaccion abre una transacción y la deja en el ctx. Si ya hay una en curso
// (llamadas anidadas), la reutiliza: todo se confirma o revierte junto.
func (t Transactor) EnTransaccion(ctx context.Context, fn func(context.Context) error) (err error) {
	if _, ok := ctx.Value(claveTx{}).(*sql.Tx); ok {
		return fn(ctx)
	}
	tx, err := t.DB.BeginTx(ctx, nil)
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

// Conn devuelve la transacción en curso si la hay; si no, la conexión normal.
func Conn(ctx context.Context, conn *sql.DB) Ejecutor {
	if tx, ok := ctx.Value(claveTx{}).(*sql.Tx); ok {
		return tx
	}
	return conn
}
