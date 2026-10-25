// Package mysql guarda la credencial de acceso en MySQL.
package mysql

import (
	"context"
	"database/sql"
	"errors"

	"magui/internal/auth/domain"
	"magui/internal/platform/db"
)

// Repositorio implementa application.RepositorioCredencial.
type Repositorio struct{ DB *sql.DB }

func (r Repositorio) Obtener(ctx context.Context) (domain.Credencial, bool, error) {
	var c domain.Credencial
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx,
		`SELECT hash, version, actualizada_en FROM credencial_acceso WHERE id = 1`).
		Scan(&c.Hash, &c.Version, &c.ActualizadaEn)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Credencial{}, false, nil
	}
	if err != nil {
		return domain.Credencial{}, false, err
	}
	return c, true, nil
}

func (r Repositorio) Guardar(ctx context.Context, c domain.Credencial) error {
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx, `
		INSERT INTO credencial_acceso (id, hash, version, actualizada_en) VALUES (1, ?, ?, ?)
		ON DUPLICATE KEY UPDATE hash = VALUES(hash), version = VALUES(version), actualizada_en = VALUES(actualizada_en)`,
		c.Hash, c.Version, c.ActualizadaEn.UTC())
	return err
}
