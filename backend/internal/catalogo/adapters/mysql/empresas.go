package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"magui/internal/catalogo/application"
	"magui/internal/catalogo/domain"
	"magui/internal/kernel/paginacion"
	"magui/internal/platform/db"
)

// Empresas implementa application.RepositorioEmpresas.
type Empresas struct{ DB *sql.DB }

func (r Empresas) Crear(ctx context.Context, e domain.Empresa) error {
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`INSERT INTO empresa (id, nombre, activo, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.Nombre, e.Activo, e.Version, e.UpdatedAt, e.UpdatedAt)
	if esDuplicado(err, "uq_empresa_nombre") {
		return domain.ErrEmpresaDuplicada
	}
	return err
}

func (r Empresas) Actualizar(ctx context.Context, e domain.Empresa, versionAnterior int) error {
	res, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`UPDATE empresa SET nombre = ?, activo = ?, version = ?, updated_at = ? WHERE id = ? AND version = ?`,
		e.Nombre, e.Activo, e.Version, e.UpdatedAt, e.ID, versionAnterior)
	if esDuplicado(err, "uq_empresa_nombre") {
		return domain.ErrEmpresaDuplicada
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.ObtenerPorID(ctx, e.ID); err != nil {
			return err
		}
		return domain.ErrVersionDesactualizada
	}
	return nil
}

const columnasEmpresa = `id, nombre, activo, version, updated_at`

func scanEmpresa(s interface{ Scan(...any) error }) (domain.Empresa, error) {
	var e domain.Empresa
	err := s.Scan(&e.ID, &e.Nombre, &e.Activo, &e.Version, &e.UpdatedAt)
	return e, err
}

func (r Empresas) ObtenerPorID(ctx context.Context, id string) (domain.Empresa, error) {
	e, err := scanEmpresa(db.Conn(ctx, r.DB).QueryRowContext(ctx,
		`SELECT `+columnasEmpresa+` FROM empresa WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Empresa{}, domain.ErrEmpresaNoEncontrada
	}
	return e, err
}

func (r Empresas) ExisteNombre(ctx context.Context, nombre, excluirID string) (bool, error) {
	var existe bool
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM empresa WHERE nombre = ? AND id <> ?)`, nombre, excluirID).Scan(&existe)
	return existe, err
}

func (r Empresas) Listar(ctx context.Context, f application.FiltroEmpresas, p paginacion.Solicitud) ([]domain.Empresa, int, error) {
	var where []string
	var args []any
	if t := strings.TrimSpace(f.Texto); t != "" {
		where = append(where, `nombre LIKE ?`)
		args = append(args, patronLike(t))
	}
	if f.SoloActivas {
		where = append(where, `activo`)
	}
	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}
	conn := db.Conn(ctx, r.DB)
	var total int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM empresa`+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn.QueryContext(ctx, `SELECT `+columnasEmpresa+` FROM empresa`+cond+
		` ORDER BY nombre, id LIMIT ? OFFSET ?`, append(args, p.Limite(), p.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Empresa
	for rows.Next() {
		e, err := scanEmpresa(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}
