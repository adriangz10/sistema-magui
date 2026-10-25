package mysql

import (
	"context"
	"database/sql"

	"magui/internal/catalogo/domain"
	"magui/internal/platform/db"
)

// Categorias implementa application.RepositorioCategorias.
type Categorias struct{ DB *sql.DB }

func (r Categorias) Crear(ctx context.Context, c domain.Categoria) error {
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`INSERT INTO categoria (id, nombre, padre_id, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.Nombre, c.PadreID, c.Version, c.UpdatedAt, c.UpdatedAt)
	if esDuplicado(err, "uq_categoria_nombre") {
		return domain.ErrCategoriaDuplicada
	}
	return err
}

func (r Categorias) Actualizar(ctx context.Context, c domain.Categoria, versionAnterior int) error {
	res, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`UPDATE categoria SET nombre = ?, padre_id = ?, version = ?, updated_at = ? WHERE id = ? AND version = ?`,
		c.Nombre, c.PadreID, c.Version, c.UpdatedAt, c.ID, versionAnterior)
	if esDuplicado(err, "uq_categoria_nombre") {
		return domain.ErrCategoriaDuplicada
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var existe bool
		if err := db.Conn(ctx, r.DB).QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM categoria WHERE id = ?)`, c.ID).Scan(&existe); err != nil {
			return err
		}
		if !existe {
			return domain.ErrCategoriaNoEncontrada
		}
		return domain.ErrVersionDesactualizada
	}
	return nil
}

func (r Categorias) Eliminar(ctx context.Context, id string) error {
	res, err := db.Conn(ctx, r.DB).ExecContext(ctx, `DELETE FROM categoria WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrCategoriaNoEncontrada
	}
	return nil
}

func (r Categorias) ListarTodas(ctx context.Context) ([]domain.Categoria, error) {
	rows, err := db.Conn(ctx, r.DB).QueryContext(ctx,
		`SELECT id, nombre, padre_id, version, updated_at FROM categoria ORDER BY nombre`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Categoria
	for rows.Next() {
		var c domain.Categoria
		if err := rows.Scan(&c.ID, &c.Nombre, &c.PadreID, &c.Version, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r Categorias) ProductosPorCategoria(ctx context.Context) (map[string]int, error) {
	rows, err := db.Conn(ctx, r.DB).QueryContext(ctx,
		`SELECT categoria_id, COUNT(*) FROM producto_categoria GROUP BY categoria_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}
