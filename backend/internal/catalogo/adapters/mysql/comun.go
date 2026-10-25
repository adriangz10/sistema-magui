// Package mysql implementa los repositorios del catálogo sobre MySQL.
package mysql

import (
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"

	"magui/internal/kernel/dinero"
)

// marcadores devuelve "?, ?, ?" para n valores.
func marcadores(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

// comoArgs convierte una lista de strings en argumentos para Exec/Query.
func comoArgs(ids []string) []any {
	out := make([]any, len(ids))
	for i, v := range ids {
		out[i] = v
	}
	return out
}

// patronLike arma "%texto%" escapando los comodines de LIKE.
func patronLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(s) + "%"
}

// dineroOpcional pasa un *Dinero a SQL (DECIMAL o NULL).
func dineroOpcional(d *dinero.Dinero) any {
	if d == nil {
		return nil
	}
	return d.String()
}

// esDuplicado indica si el error es una violación de la clave única indicada.
func esDuplicado(err error, clave string) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062 && strings.Contains(me.Message, clave)
}
