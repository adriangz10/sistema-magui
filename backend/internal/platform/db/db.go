// Package db abre conexiones a MySQL (nube) y SQLite (escritorio) y aplica migraciones.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/go-sql-driver/mysql" // driver "mysql"
	_ "modernc.org/sqlite"             // driver "sqlite" (Go puro, sin CGO)
)

// Dialect identifica el motor de base de datos.
type Dialect string

const (
	MySQL  Dialect = "mysql"
	SQLite Dialect = "sqlite"
)

// OpenMySQL abre el pool de conexiones y verifica que la base responda.
func OpenMySQL(ctx context.Context, dsn string) (*sql.DB, error) {
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrir mysql: %w", err)
	}
	conn.SetMaxOpenConns(20)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)
	if err := ping(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("conectar a mysql: %w", err)
	}
	return conn, nil
}

// OpenSQLite abre (o crea) el archivo local con claves foráneas activas y modo WAL.
// Usa una sola conexión: evita bloqueos "database is locked" en una app de un solo usuario.
func OpenSQLite(ctx context.Context, path string) (*sql.DB, error) {
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "synchronous(NORMAL)")
	conn, err := sql.Open("sqlite", "file:"+path+"?"+q.Encode())
	if err != nil {
		return nil, fmt.Errorf("abrir sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)
	if err := ping(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("conectar a sqlite: %w", err)
	}
	return conn, nil
}

func ping(ctx context.Context, conn *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return conn.PingContext(ctx)
}
