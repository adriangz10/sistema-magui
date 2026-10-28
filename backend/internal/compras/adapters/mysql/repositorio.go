// Package mysql implementa el repositorio de compras sobre MySQL.
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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
		`INSERT INTO compra (id, fecha, empresa_id, total, notas, created_at, version, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Fecha, c.EmpresaID, c.Total.String(), c.Notas, c.CreatedAt, 1, c.CreatedAt); err != nil {
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
	SELECT c.id, c.fecha, c.empresa_id, e.nombre, c.total, c.notas, c.version, c.updated_at,
	       (SELECT COUNT(*) FROM compra_item ci WHERE ci.compra_id = c.id),
	       (SELECT COALESCE(SUM(ci.cantidad), 0) FROM compra_item ci WHERE ci.compra_id = c.id)
	FROM compra c LEFT JOIN empresa e ON e.id = c.empresa_id`

func scanResumen(s interface{ Scan(...any) error }) (application.CompraResumen, error) {
	var c application.CompraResumen
	var actualizada time.Time
	err := s.Scan(&c.ID, &c.Fecha, &c.EmpresaID, &c.Empresa, &c.Total, &c.Notas, &c.Version, &actualizada, &c.Productos, &c.Unidades)
	if c.Version > 1 {
		c.EditadaEn = &actualizada
	}
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

// ObtenerParaEditar lee la compra con FOR UPDATE: otra edición de la misma compra espera
// hasta que esta transacción termine (ADR 0002).
func (r Repositorio) ObtenerParaEditar(ctx context.Context, id string) (domain.Compra, error) {
	conn := db.Conn(ctx, r.DB)
	var c domain.Compra
	var total dinero.Dinero
	err := conn.QueryRowContext(ctx, `
		SELECT c.id, c.fecha, c.empresa_id, e.nombre, c.total, c.notas, c.created_at, c.version, c.updated_at
		FROM compra c LEFT JOIN empresa e ON e.id = c.empresa_id
		WHERE c.id = ? FOR UPDATE OF c`, id).
		Scan(&c.ID, &c.Fecha, &c.EmpresaID, &nombreEmpresa{&c.Empresa}, &total, &c.Notas, &c.CreatedAt, &c.Version, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Compra{}, domain.ErrCompraNoEncontrada
	}
	if err != nil {
		return domain.Compra{}, err
	}
	c.Total = total
	rows, err := conn.QueryContext(ctx, `
		SELECT id, producto_id, cantidad, costo_unitario, precio_venta_nuevo, fecha_vencimiento
		FROM compra_item WHERE compra_id = ? ORDER BY id`, id)
	if err != nil {
		return domain.Compra{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var it domain.Item
		var precio sql.Null[dinero.Dinero]
		var venc sql.Null[fecha.Fecha]
		if err := rows.Scan(&it.ID, &it.ProductoID, &it.Cantidad, &it.CostoUnitario, &precio, &venc); err != nil {
			return domain.Compra{}, err
		}
		if precio.Valid {
			it.PrecioVentaNuevo = &precio.V
		}
		if venc.Valid {
			it.Vencimiento = &venc.V
		}
		c.Items = append(c.Items, it)
	}
	return c, rows.Err()
}

// nombreEmpresa lee el nombre (NULL si la compra no tiene empresa) como string.
type nombreEmpresa struct{ dst *string }

func (n *nombreEmpresa) Scan(src any) error {
	var s sql.NullString
	if err := s.Scan(src); err != nil {
		return err
	}
	*n.dst = s.String
	return nil
}

func (r Repositorio) ActualizarCompra(ctx context.Context, c domain.Compra, versionAnterior int) error {
	res, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`UPDATE compra SET fecha = ?, empresa_id = ?, notas = ?, version = ?, updated_at = ? WHERE id = ? AND version = ?`,
		c.Fecha, c.EmpresaID, c.Notas, c.Version, c.UpdatedAt, c.ID, versionAnterior)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return domain.ErrVersionDesactualizada
	}
	return nil
}

func (r Repositorio) GuardarEdicion(ctx context.Context, e domain.Edicion) error {
	doc, err := json.Marshal(e.Cambios)
	if err != nil {
		return err
	}
	_, err = db.Conn(ctx, r.DB).ExecContext(ctx,
		`INSERT INTO compra_edicion (id, compra_id, fecha, origen, motivo, version, cambios) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.CompraID, e.Fecha, string(e.Origen), e.Motivo, e.Version, string(doc))
	return err
}

func (r Repositorio) Ediciones(ctx context.Context, compraID string) ([]domain.Edicion, error) {
	rows, err := db.Conn(ctx, r.DB).QueryContext(ctx, `
		SELECT id, compra_id, fecha, origen, motivo, version, cambios
		FROM compra_edicion WHERE compra_id = ? ORDER BY version DESC`, compraID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Edicion{}
	for rows.Next() {
		var e domain.Edicion
		var origen string
		var doc []byte
		if err := rows.Scan(&e.ID, &e.CompraID, &e.Fecha, &origen, &e.Motivo, &e.Version, &doc); err != nil {
			return nil, err
		}
		e.Origen = domain.Origen(origen)
		if e.Cambios, err = domain.LeerCambios(doc); err != nil {
			return nil, fmt.Errorf("edición %s: %w", e.ID, err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
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
