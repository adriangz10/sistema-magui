package db

import (
	"errors"

	"github.com/go-sql-driver/mysql"

	"magui/internal/kernel/errores"
)

// Códigos de MySQL para operaciones que chocaron con otra transacción (ADR 0002).
const (
	mysqlDeadlock        = 1213 // ER_LOCK_DEADLOCK
	mysqlLockWaitTimeout = 1205 // ER_LOCK_WAIT_TIMEOUT
)

// ErrOperacionConcurrente es lo que ve el cliente cuando MySQL aborta la transacción por
// deadlock o por tiempo de espera de un bloqueo y se agotaron los reintentos.
var ErrOperacionConcurrente = errores.Conflicto("operacion_concurrente",
	"Otra operación se cruzó con esta. Intentá de nuevo.")

// EsConcurrencia indica si el error (o alguno de los que envuelve) es un deadlock o una espera
// de bloqueo agotada de MySQL: la transacción se puede reintentar.
func EsConcurrencia(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && (me.Number == mysqlDeadlock || me.Number == mysqlLockWaitTimeout)
}

// EsReintentable indica si conviene reejecutar la transacción: un deadlock de MySQL (1213) o el
// ErrOperacionConcurrente que devuelve inventario cuando los lotes cambian durante el bloqueo
// (no implica esperar). La espera de bloqueo agotada (1205) no se reintenta.
func EsReintentable(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == mysqlDeadlock
	}
	return errors.Is(err, ErrOperacionConcurrente)
}

// TraducirConcurrencia convierte los errores 1213 y 1205 de MySQL en 409 operacion_concurrente;
// cualquier otro error pasa sin cambios.
func TraducirConcurrencia(err error) error {
	if EsConcurrencia(err) {
		return ErrOperacionConcurrente
	}
	return err
}
