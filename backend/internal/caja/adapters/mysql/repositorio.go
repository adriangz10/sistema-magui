// Package mysql implementa el repositorio de gastos sobre MySQL.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"magui/internal/caja/application"
	"magui/internal/caja/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/paginacion"
	"magui/internal/platform/db"
)

type Repositorio struct{ DB *sql.DB }

func (r Repositorio) Guardar(ctx context.Context, g domain.Gasto) error {
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`INSERT INTO gasto_caja (id, fecha, monto, motivo, categoria, metodo, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		g.ID, g.Fecha, g.Monto.String(), g.Motivo, g.Categoria, string(g.Metodo), g.CreatedAt)
	return err
}

const columnas = `id, fecha, monto, motivo, categoria, metodo, anulado_en, motivo_anulacion, created_at`

func scan(s interface{ Scan(...any) error }) (domain.Gasto, error) {
	var g domain.Gasto
	var metodo string
	err := s.Scan(&g.ID, &g.Fecha, &g.Monto, &g.Motivo, &g.Categoria, &metodo, &g.AnuladoEn, &g.MotivoAnulacion, &g.CreatedAt)
	g.Metodo = domain.Metodo(metodo)
	return g, err
}

func (r Repositorio) Obtener(ctx context.Context, id string) (domain.Gasto, error) {
	g, err := scan(db.Conn(ctx, r.DB).QueryRowContext(ctx, `SELECT `+columnas+` FROM gasto_caja WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Gasto{}, domain.ErrGastoNoEncontrado
	}
	return g, err
}

func (r Repositorio) MarcarAnulado(ctx context.Context, g domain.Gasto) error {
	res, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`UPDATE gasto_caja SET anulado_en = ?, motivo_anulacion = ? WHERE id = ? AND anulado_en IS NULL`,
		g.AnuladoEn, g.MotivoAnulacion, g.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrGastoYaAnulado
	}
	return nil
}

func (r Repositorio) Listar(ctx context.Context, f application.Filtro, p paginacion.Solicitud) ([]domain.Gasto, int, dinero.Dinero, error) {
	var where []string
	var args []any
	if f.Desde != nil {
		where, args = append(where, `fecha >= ?`), append(args, f.Desde.UTC())
	}
	if f.Hasta != nil {
		where, args = append(where, `fecha < ?`), append(args, f.Hasta.UTC())
	}
	if f.Categoria != "" {
		where, args = append(where, `categoria = ?`), append(args, f.Categoria)
	}
	if !f.IncluirAnulados {
		where = append(where, `anulado_en IS NULL`)
	}
	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}
	conn := db.Conn(ctx, r.DB)
	var total int
	var suma dinero.Dinero
	// La suma excluye anulados aunque se muestren.
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN anulado_en IS NULL THEN monto END), 0) FROM gasto_caja`+cond, args...).
		Scan(&total, &suma); err != nil {
		return nil, 0, 0, err
	}
	rows, err := conn.QueryContext(ctx, `SELECT `+columnas+` FROM gasto_caja`+cond+` ORDER BY fecha DESC, id DESC LIMIT ? OFFSET ?`,
		append(args, p.Limite(), p.Offset())...)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()
	var out []domain.Gasto
	for rows.Next() {
		g, err := scan(rows)
		if err != nil {
			return nil, 0, 0, err
		}
		out = append(out, g)
	}
	return out, total, suma, rows.Err()
}

func (r Repositorio) CategoriasUsadas(ctx context.Context) ([]string, error) {
	rows, err := db.Conn(ctx, r.DB).QueryContext(ctx,
		`SELECT categoria FROM gasto_caja WHERE categoria IS NOT NULL GROUP BY categoria ORDER BY COUNT(*) DESC, categoria LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
