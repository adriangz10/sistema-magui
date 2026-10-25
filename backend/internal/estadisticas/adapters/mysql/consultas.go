// Package mysql resuelve las consultas de estadísticas sobre MySQL (solo lectura).
package mysql

import (
	"context"
	"database/sql"
	"time"

	"magui/internal/estadisticas/application"
	"magui/internal/kernel/dinero"
)

// Consultas implementa application.Consultas. Solo cuentan las ventas CONFIRMADAS y los gastos no anulados.
type Consultas struct{ DB *sql.DB }

func utc(t time.Time) time.Time { return t.UTC() }

func (c Consultas) Ventas(ctx context.Context, desde, hasta time.Time) (application.TotalesVentas, error) {
	var t application.TotalesVentas
	err := c.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(estado = 'CONFIRMADA'), 0),
		       COALESCE(SUM(CASE WHEN estado = 'CONFIRMADA' THEN total END), 0),
		       COALESCE(SUM(CASE WHEN estado = 'CONFIRMADA' THEN costo_total END), 0),
		       COALESCE(SUM(estado = 'ANULADA'), 0),
		       (SELECT COUNT(*) FROM venta_item i JOIN venta v2 ON v2.id = i.venta_id
		         WHERE v2.estado = 'CONFIRMADA' AND v2.fecha >= ? AND v2.fecha < ? AND i.costo_unitario IS NULL)
		FROM venta WHERE fecha >= ? AND fecha < ?`,
		utc(desde), utc(hasta), utc(desde), utc(hasta)).
		Scan(&t.Cantidad, &t.Total, &t.Costo, &t.Anuladas, &t.RenglonesSinCosto)
	return t, err
}

func (c Consultas) VentasPorMetodo(ctx context.Context, desde, hasta time.Time) ([]application.MontoPorMetodo, error) {
	rows, err := c.DB.QueryContext(ctx, `
		SELECT p.metodo, SUM(p.monto), COUNT(DISTINCT p.venta_id)
		FROM venta_pago p JOIN venta v ON v.id = p.venta_id
		WHERE v.estado = 'CONFIRMADA' AND v.fecha >= ? AND v.fecha < ?
		GROUP BY p.metodo`, utc(desde), utc(hasta))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []application.MontoPorMetodo
	for rows.Next() {
		var m application.MontoPorMetodo
		if err := rows.Scan(&m.Metodo, &m.Monto, &m.Cantidad); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (c Consultas) Compras(ctx context.Context, desde, hasta time.Time) (int, dinero.Dinero, error) {
	var n int
	var total dinero.Dinero
	err := c.DB.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(total), 0) FROM compra WHERE fecha >= ? AND fecha < ?`,
		utc(desde), utc(hasta)).Scan(&n, &total)
	return n, total, err
}

func (c Consultas) Gastos(ctx context.Context, desde, hasta time.Time) (application.TotalesGastos, error) {
	var g application.TotalesGastos
	err := c.DB.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(monto), 0), COALESCE(SUM(CASE WHEN metodo = 'EFECTIVO' THEN monto END), 0)
		FROM gasto_caja WHERE anulado_en IS NULL AND fecha >= ? AND fecha < ?`,
		utc(desde), utc(hasta)).Scan(&g.Cantidad, &g.Total, &g.Efectivo)
	return g, err
}

func (c Consultas) VentasPorDia(ctx context.Context, desde, hasta time.Time, offset string) ([]application.FilaDia, error) {
	rows, err := c.DB.QueryContext(ctx, `
		SELECT DATE(CONVERT_TZ(fecha, '+00:00', ?)) AS dia, COUNT(*), SUM(total), SUM(total - costo_total)
		FROM venta WHERE estado = 'CONFIRMADA' AND fecha >= ? AND fecha < ?
		GROUP BY dia ORDER BY dia`, offset, utc(desde), utc(hasta))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []application.FilaDia
	for rows.Next() {
		var f application.FilaDia
		if err := rows.Scan(&f.Fecha, &f.Cantidad, &f.Total, &f.Ganancia); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (c Consultas) VentasPorHora(ctx context.Context, desde, hasta time.Time, offset string) ([]application.FilaHora, error) {
	rows, err := c.DB.QueryContext(ctx, `
		SELECT HOUR(CONVERT_TZ(fecha, '+00:00', ?)) AS hora, COUNT(*), SUM(total)
		FROM venta WHERE estado = 'CONFIRMADA' AND fecha >= ? AND fecha < ?
		GROUP BY hora ORDER BY hora`, offset, utc(desde), utc(hasta))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []application.FilaHora
	for rows.Next() {
		var f application.FilaHora
		if err := rows.Scan(&f.Hora, &f.Cantidad, &f.Total); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (c Consultas) MasVendidos(ctx context.Context, desde, hasta time.Time, limite int, orden application.OrdenRanking) ([]application.ProductoVendido, error) {
	ordenSQL := `unidades DESC, total DESC`
	if orden == application.PorImporte {
		ordenSQL = `total DESC, unidades DESC`
	}
	rows, err := c.DB.QueryContext(ctx, `
		SELECT i.producto_id, p.sku, p.nombre, p.tipo, SUM(i.cantidad) AS unidades, SUM(i.subtotal) AS total,
		       SUM(CASE WHEN i.costo_unitario IS NOT NULL THEN i.subtotal - i.costo_unitario * i.cantidad END)
		FROM venta_item i
		JOIN venta v ON v.id = i.venta_id
		JOIN producto p ON p.id = i.producto_id
		WHERE v.estado = 'CONFIRMADA' AND v.fecha >= ? AND v.fecha < ?
		GROUP BY i.producto_id, p.sku, p.nombre, p.tipo
		ORDER BY `+ordenSQL+`, p.nombre LIMIT ?`, utc(desde), utc(hasta), limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []application.ProductoVendido{}
	for rows.Next() {
		var p application.ProductoVendido
		var ganancia sql.Null[dinero.Dinero]
		if err := rows.Scan(&p.ProductoID, &p.SKU, &p.Nombre, &p.Tipo, &p.Unidades, &p.Total, &ganancia); err != nil {
			return nil, err
		}
		if ganancia.Valid {
			p.Ganancia = &ganancia.V
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
