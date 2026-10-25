package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/fs"

	"github.com/pressly/goose/v3"

	"magui/migrations"
)

// Migrator aplica las migraciones embebidas del dialecto indicado.
type Migrator struct {
	p *goose.Provider
}

// NewMigrator prepara las migraciones de migrations/<dialect>.
func NewMigrator(conn *sql.DB, d Dialect) (*Migrator, error) {
	var gd goose.Dialect
	switch d {
	case MySQL:
		gd = goose.DialectMySQL
	case SQLite:
		gd = goose.DialectSQLite3
	default:
		return nil, fmt.Errorf("dialecto desconocido %q", d)
	}
	sub, err := fs.Sub(migrations.FS, string(d))
	if err != nil {
		return nil, err
	}
	p, err := goose.NewProvider(gd, conn, sub)
	if err != nil {
		return nil, fmt.Errorf("preparar migraciones: %w", err)
	}
	return &Migrator{p: p}, nil
}

// Up aplica todas las migraciones pendientes.
func (m *Migrator) Up(ctx context.Context) error {
	_, err := m.p.Up(ctx)
	return err
}

// Down revierte la última migración.
func (m *Migrator) Down(ctx context.Context) error {
	_, err := m.p.Down(ctx)
	return err
}

// Reset revierte todas las migraciones (borra el esquema).
func (m *Migrator) Reset(ctx context.Context) error {
	_, err := m.p.DownTo(ctx, 0)
	return err
}

// Version devuelve la versión actual del esquema.
func (m *Migrator) Version(ctx context.Context) (int64, error) {
	return m.p.GetDBVersion(ctx)
}

// PrintStatus escribe el estado de cada migración.
func (m *Migrator) PrintStatus(ctx context.Context, w io.Writer) error {
	st, err := m.p.Status(ctx)
	if err != nil {
		return err
	}
	for _, s := range st {
		fecha := "pendiente"
		if !s.AppliedAt.IsZero() {
			fecha = s.AppliedAt.Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "%05d  %-30s %s\n", s.Source.Version, s.Source.Path, fecha)
	}
	return nil
}
