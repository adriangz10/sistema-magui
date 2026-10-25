// Package mysql implementa el repositorio de ventas sobre MySQL.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/paginacion"
	"magui/internal/platform/db"
	"magui/internal/ventas/application"
	"magui/internal/ventas/domain"
)

// Repositorio implementa application.Repositorio.
type Repositorio struct{ DB *sql.DB }

func opcional(d *dinero.Dinero) any {
	if d == nil {
		return nil
	}
	return d.String()
}

func (r Repositorio) Guardar(ctx context.Context, v domain.Venta) error {
	conn := db.Conn(ctx, r.DB)
	costo, _ := v.CostoTotal()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO venta (id, fecha, total, costo_total, estado, origen, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.Fecha, v.Total.String(), costo.String(), string(v.Estado), string(v.Origen), v.CreatedAt); err != nil {
		return err
	}

	args := make([]any, 0, 7*len(v.Items))
	for _, it := range v.Items {
		args = append(args, it.ID, v.ID, it.ProductoID, it.Cantidad, it.PrecioUnitario.String(),
			opcional(it.CostoUnitario()), it.Subtotal().String())
	}
	q := `INSERT INTO venta_item (id, venta_id, producto_id, cantidad, precio_unitario, costo_unitario, subtotal) VALUES ` +
		strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?, ?, ?), ", len(v.Items)), ", ")
	if _, err := conn.ExecContext(ctx, q, args...); err != nil {
		return err
	}

	args = make([]any, 0, 6*len(v.Pagos))
	for _, p := range v.Pagos {
		args = append(args, p.ID, v.ID, string(p.Metodo), p.Monto.String(), opcional(p.MontoRecibido), opcional(p.Vuelto))
	}
	q = `INSERT INTO venta_pago (id, venta_id, metodo, monto, monto_recibido, vuelto) VALUES ` +
		strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?, ?), ", len(v.Pagos)), ", ")
	_, err := conn.ExecContext(ctx, q, args...)
	return err
}

func (r Repositorio) Obtener(ctx context.Context, id string) (domain.Venta, error) {
	var v domain.Venta
	var estado, origen string
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx,
		`SELECT id, fecha, total, estado, origen, anulada_en, motivo_anulacion, created_at FROM venta WHERE id = ?`, id).
		Scan(&v.ID, &v.Fecha, &v.Total, &estado, &origen, &v.AnuladaEn, &v.MotivoAnulacion, &v.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Venta{}, domain.ErrVentaNoEncontrada
	}
	v.Estado, v.Origen = domain.Estado(estado), domain.Origen(origen)
	return v, err
}

func (r Repositorio) MarcarAnulada(ctx context.Context, v domain.Venta) error {
	res, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`UPDATE venta SET estado = 'ANULADA', anulada_en = ?, motivo_anulacion = ? WHERE id = ? AND estado = 'CONFIRMADA'`,
		v.AnuladaEn, v.MotivoAnulacion, v.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrVentaYaAnulada
	}
	return nil
}

const consultaResumen = `
	SELECT v.id, v.fecha, v.total, v.estado, v.origen,
	       (SELECT COUNT(*) FROM venta_item i WHERE i.venta_id = v.id),
	       (SELECT COALESCE(SUM(i.cantidad), 0) FROM venta_item i WHERE i.venta_id = v.id),
	       (SELECT GROUP_CONCAT(DISTINCT p.metodo ORDER BY p.metodo) FROM venta_pago p WHERE p.venta_id = v.id),
	       v.costo_total, v.anulada_en, v.motivo_anulacion,
	       (SELECT COALESCE(SUM(p.vuelto), 0) FROM venta_pago p WHERE p.venta_id = v.id),
	       NOT EXISTS (SELECT 1 FROM venta_item i WHERE i.venta_id = v.id AND i.costo_unitario IS NULL)
	FROM venta v`

func scanDetalle(s interface{ Scan(...any) error }) (application.VentaDetalle, error) {
	var d application.VentaDetalle
	var estado, origen string
	var metodos sql.NullString
	err := s.Scan(&d.ID, &d.Fecha, &d.Total, &estado, &origen, &d.Productos, &d.Unidades, &metodos,
		&d.CostoTotal, &d.AnuladaEn, &d.MotivoAnulacion, &d.Vuelto, &d.CostoCompleto)
	d.Estado, d.Origen = domain.Estado(estado), domain.Origen(origen)
	d.Metodos = []domain.MetodoPago{}
	if metodos.Valid {
		for m := range strings.SplitSeq(metodos.String, ",") {
			d.Metodos = append(d.Metodos, domain.MetodoPago(m))
		}
	}
	d.Ganancia = d.Total.Restar(d.CostoTotal)
	return d, err
}

func (r Repositorio) Detalle(ctx context.Context, id string) (application.VentaDetalle, error) {
	conn := db.Conn(ctx, r.DB)
	d, err := scanDetalle(conn.QueryRowContext(ctx, consultaResumen+` WHERE v.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return application.VentaDetalle{}, domain.ErrVentaNoEncontrada
	}
	if err != nil {
		return application.VentaDetalle{}, err
	}

	d.Items = []application.ItemDTO{}
	rows, err := conn.QueryContext(ctx, `
		SELECT i.producto_id, p.sku, p.nombre, p.tipo, i.cantidad, i.precio_unitario, i.subtotal
		FROM venta_item i JOIN producto p ON p.id = i.producto_id
		WHERE i.venta_id = ? ORDER BY i.id`, id)
	if err != nil {
		return application.VentaDetalle{}, err
	}
	for rows.Next() {
		var it application.ItemDTO
		if err := rows.Scan(&it.ProductoID, &it.SKU, &it.Producto, &it.Tipo, &it.Cantidad, &it.PrecioUnitario, &it.Subtotal); err != nil {
			rows.Close()
			return application.VentaDetalle{}, err
		}
		d.Items = append(d.Items, it)
	}
	rows.Close()

	d.Pagos = []application.PagoDTO{}
	rows, err = conn.QueryContext(ctx,
		`SELECT metodo, monto, monto_recibido, vuelto FROM venta_pago WHERE venta_id = ? ORDER BY id`, id)
	if err != nil {
		return application.VentaDetalle{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var p application.PagoDTO
		var metodo string
		var recibido, vuelto sql.Null[dinero.Dinero]
		if err := rows.Scan(&metodo, &p.Monto, &recibido, &vuelto); err != nil {
			return application.VentaDetalle{}, err
		}
		p.Metodo = domain.MetodoPago(metodo)
		if recibido.Valid {
			p.MontoRecibido = &recibido.V
		}
		if vuelto.Valid {
			p.Vuelto = &vuelto.V
		}
		d.Pagos = append(d.Pagos, p)
	}
	return d, rows.Err()
}

func (r Repositorio) Listar(ctx context.Context, f application.Filtro, p paginacion.Solicitud) ([]application.VentaResumen, int, error) {
	var where []string
	var args []any
	if f.Desde != nil {
		where, args = append(where, `v.fecha >= ?`), append(args, f.Desde.UTC())
	}
	if f.Hasta != nil {
		where, args = append(where, `v.fecha < ?`), append(args, f.Hasta.UTC())
	}
	if f.Estado != "" {
		where, args = append(where, `v.estado = ?`), append(args, string(f.Estado))
	}
	if f.Metodo != "" {
		where, args = append(where, `EXISTS (SELECT 1 FROM venta_pago vp WHERE vp.venta_id = v.id AND vp.metodo = ?)`),
			append(args, string(f.Metodo))
	}
	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}
	conn := db.Conn(ctx, r.DB)
	var total int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM venta v`+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn.QueryContext(ctx, consultaResumen+cond+` ORDER BY v.fecha DESC, v.id DESC LIMIT ? OFFSET ?`,
		append(args, p.Limite(), p.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []application.VentaResumen{}
	for rows.Next() {
		d, err := scanDetalle(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, d.VentaResumen)
	}
	return out, total, rows.Err()
}
