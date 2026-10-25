// Package mysql implementa el repositorio de compras sobre MySQL.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"magui/internal/compras/application"
	"magui/internal/compras/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/paginacion"
	"magui/internal/platform/db"
)

// Repositorio implementa application.Repositorio.
type Repositorio struct{ DB *sql.DB }

func (r Repositorio) Guardar(ctx context.Context, c domain.Compra) error {
	conn := db.Conn(ctx, r.DB)
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO compra (id, fecha, empresa_id, total, notas, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.Fecha, c.EmpresaID, c.Total.String(), c.Notas, c.CreatedAt); err != nil {
		return err
	}
	args := make([]any, 0, 7*len(c.Items))
	for _, it := range c.Items {
		var precio any
		if it.PrecioVentaNuevo != nil {
			precio = it.PrecioVentaNuevo.String()
		}
		args = append(args, it.ID, c.ID, it.ProductoID, it.Cantidad, it.CostoUnitario.String(), precio, it.Vencimiento)
	}
	q := `INSERT INTO compra_item (id, compra_id, producto_id, cantidad, costo_unitario, precio_venta_nuevo, fecha_vencimiento) VALUES ` +
		strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?, ?, ?), ", len(c.Items)), ", ")
	_, err := conn.ExecContext(ctx, q, args...)
	return err
}

const consultaResumen = `
	SELECT c.id, c.fecha, c.empresa_id, e.nombre, c.total, c.notas,
	       (SELECT COUNT(*) FROM compra_item ci WHERE ci.compra_id = c.id),
	       (SELECT COALESCE(SUM(ci.cantidad), 0) FROM compra_item ci WHERE ci.compra_id = c.id)
	FROM compra c LEFT JOIN empresa e ON e.id = c.empresa_id`

func scanResumen(s interface{ Scan(...any) error }) (application.CompraResumen, error) {
	var c application.CompraResumen
	err := s.Scan(&c.ID, &c.Fecha, &c.EmpresaID, &c.Empresa, &c.Total, &c.Notas, &c.Productos, &c.Unidades)
	return c, err
}

func (r Repositorio) Obtener(ctx context.Context, id string) (application.CompraDetalle, error) {
	conn := db.Conn(ctx, r.DB)
	res, err := scanResumen(conn.QueryRowContext(ctx, consultaResumen+` WHERE c.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return application.CompraDetalle{}, domain.ErrCompraNoEncontrada
	}
	if err != nil {
		return application.CompraDetalle{}, err
	}
	det := application.CompraDetalle{CompraResumen: res, Items: []application.ItemDTO{}}
	rows, err := conn.QueryContext(ctx, `
		SELECT ci.id, ci.producto_id, p.sku, p.nombre, ci.cantidad, ci.costo_unitario, ci.precio_venta_nuevo, ci.fecha_vencimiento
		FROM compra_item ci JOIN producto p ON p.id = ci.producto_id
		WHERE ci.compra_id = ? ORDER BY ci.id`, id)
	if err != nil {
		return application.CompraDetalle{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var it application.ItemDTO
		var precio sql.Null[dinero.Dinero]
		var venc sql.Null[fecha.Fecha]
		if err := rows.Scan(&it.ID, &it.ProductoID, &it.SKU, &it.Producto, &it.Cantidad, &it.CostoUnitario, &precio, &venc); err != nil {
			return application.CompraDetalle{}, err
		}
		it.Subtotal = it.CostoUnitario.Multiplicar(it.Cantidad)
		if precio.Valid {
			it.PrecioVentaNuevo = &precio.V
		}
		if venc.Valid {
			it.Vencimiento = &venc.V
		}
		det.Items = append(det.Items, it)
	}
	return det, rows.Err()
}

func (r Repositorio) Listar(ctx context.Context, f application.Filtro, p paginacion.Solicitud) ([]application.CompraResumen, int, error) {
	var where []string
	var args []any
	if f.Desde != nil {
		where, args = append(where, `c.fecha >= ?`), append(args, f.Desde.UTC())
	}
	if f.Hasta != nil {
		where, args = append(where, `c.fecha < ?`), append(args, f.Hasta.UTC())
	}
	if f.EmpresaID != "" {
		where, args = append(where, `c.empresa_id = ?`), append(args, f.EmpresaID)
	}
	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}
	conn := db.Conn(ctx, r.DB)
	var total int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM compra c`+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn.QueryContext(ctx, consultaResumen+cond+` ORDER BY c.fecha DESC, c.id DESC LIMIT ? OFFSET ?`,
		append(args, p.Limite(), p.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []application.CompraResumen{}
	for rows.Next() {
		c, err := scanResumen(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}
