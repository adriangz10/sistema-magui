// Package tx define el puerto de transacciones: un caso de uso puede agrupar varias
// operaciones de repositorio en una unidad atómica sin saber nada de SQL.
package tx

import "context"

// Transactor ejecuta fn dentro de una transacción. Si fn devuelve error, se revierte todo.
// Los repositorios toman la transacción desde el ctx que recibe fn.
type Transactor interface {
	EnTransaccion(ctx context.Context, fn func(ctx context.Context) error) error
}

// SinTransaccion ejecuta fn directamente (útil en tests con repositorios en memoria).
type SinTransaccion struct{}

func (SinTransaccion) EnTransaccion(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
