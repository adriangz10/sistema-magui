// Package mysql implementa el repositorio de inventario sobre MySQL.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"magui/internal/inventario/application"
	"magui/internal/inventario/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/paginacion"
	"magui/internal/platform/db"
)

const claveDiasAlerta = "dias_alerta_vencimiento"

// DiasAlertaPorDefecto si no está configurado.
const DiasAlertaPorDefecto = 90

// Repositorio implementa application.Repositorio.
type Repositorio struct{ DB *sql.DB }

func (r Repositorio) CrearLote(ctx context.Context, l domain.Lote) error {
	var costo any
	if l.Costo != nil {
		costo = l.Costo.String()
	}
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`INSERT INTO lote (id, producto_id, fecha_vencimiento, costo_unitario, compra_item_id, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		l.ID, l.ProductoID, l.Vencimiento, costo, l.CompraItemID, l.CreatedAt)
	return err
}

func (r Repositorio) RegistrarMovimientos(ctx context.Context, movs []domain.Movimiento) error {
	if len(movs) == 0 {
		return nil
	}
	args := make([]any, 0, 8*len(movs))
	for _, m := range movs {
		var nota any
		if m.Nota != "" {
			nota = m.Nota
		}
		args = append(args, m.ID, m.ProductoID, m.LoteID, m.Cantidad, string(m.Tipo), m.ReferenciaID, nota, m.Fecha)
	}
	q := `INSERT INTO movimiento_stock (id, producto_id, lote_id, cantidad, tipo, referencia_id, nota, fecha) VALUES ` +
		strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?, ?, ?, ?), ", len(movs)), ", ")
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx, q, args...)
	return err
}

func (r Repositorio) MovimientosPorReferencia(ctx context.Context, referenciaID string, tipo domain.TipoMovimiento) ([]domain.Movimiento, error) {
	rows, err := db.Conn(ctx, r.DB).QueryContext(ctx, `
		SELECT id, producto_id, lote_id, cantidad, tipo, referencia_id, COALESCE(nota, ''), fecha
		FROM movimiento_stock WHERE referencia_id = ? AND tipo = ? ORDER BY id`, referenciaID, string(tipo))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Movimiento
	for rows.Next() {
		var m domain.Movimiento
		var t string
		if err := rows.Scan(&m.ID, &m.ProductoID, &m.LoteID, &m.Cantidad, &t, &m.ReferenciaID, &m.Nota, &m.Fecha); err != nil {
			return nil, err
		}
		m.Tipo = domain.TipoMovimiento(t)
		out = append(out, m)
	}
	return out, rows.Err()
}

const consultaLotes = `
	SELECT l.id, l.producto_id, l.fecha_vencimiento, l.costo_unitario, l.compra_item_id, l.created_at,
	       COALESCE(SUM(m.cantidad), 0) AS stock
	FROM lote l LEFT JOIN movimiento_stock m ON m.lote_id = l.id`

func scanLote(s interface{ Scan(...any) error }) (domain.LoteStock, error) {
	var l domain.LoteStock
	var venc sql.Null[fecha.Fecha]
	var costo sql.Null[dinero.Dinero]
	if err := s.Scan(&l.ID, &l.ProductoID, &venc, &costo, &l.CompraItemID, &l.CreatedAt, &l.Stock); err != nil {
		return l, err
	}
	if venc.Valid {
		l.Vencimiento = &venc.V
	}
	if costo.Valid {
		l.Costo = &costo.V
	}
	return l, nil
}

func (r Repositorio) ObtenerLote(ctx context.Context, id string) (domain.LoteStock, error) {
	l, err := scanLote(db.Conn(ctx, r.DB).QueryRowContext(ctx, consultaLotes+` WHERE l.id = ? GROUP BY l.id`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.LoteStock{}, domain.ErrLoteNoEncontrado
	}
	return l, err
}

func (r Repositorio) LotesConStock(ctx context.Context, productoID string) ([]domain.LoteStock, error) {
	rows, err := db.Conn(ctx, r.DB).QueryContext(ctx, consultaLotes+`
		WHERE l.producto_id = ? GROUP BY l.id HAVING stock <> 0
		ORDER BY l.fecha_vencimiento IS NULL, l.fecha_vencimiento, l.created_at`, productoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LoteStock
	for rows.Next() {
		l, err := scanLote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r Repositorio) StockProducto(ctx context.Context, productoID string) (int, error) {
	var n int
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx,
		`SELECT COALESCE(SUM(cantidad), 0) FROM movimiento_stock WHERE producto_id = ?`, productoID).Scan(&n)
	return n, err
}

func (r Repositorio) ListarMovimientos(ctx context.Context, f application.FiltroMovimientos, p paginacion.Solicitud) ([]application.MovimientoDTO, int, error) {
	var where []string
	var args []any
	if f.ProductoID != "" {
		where, args = append(where, `m.producto_id = ?`), append(args, f.ProductoID)
	}
	if f.Tipo != "" {
		where, args = append(where, `m.tipo = ?`), append(args, string(f.Tipo))
	}
	if f.Desde != nil {
		where, args = append(where, `m.fecha >= ?`), append(args, f.Desde.UTC())
	}
	if f.Hasta != nil {
		where, args = append(where, `m.fecha < ?`), append(args, f.Hasta.UTC())
	}
	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}
	conn := db.Conn(ctx, r.DB)
	var total int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM movimiento_stock m`+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn.QueryContext(ctx, `
		SELECT m.id, m.fecha, m.producto_id, p.sku, p.nombre, m.cantidad, m.tipo, m.referencia_id,
		       COALESCE(m.nota, ''), l.fecha_vencimiento
		FROM movimiento_stock m
		JOIN producto p ON p.id = m.producto_id
		LEFT JOIN lote l ON l.id = m.lote_id`+cond+`
		ORDER BY m.fecha DESC, m.id DESC LIMIT ? OFFSET ?`, append(args, p.Limite(), p.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []application.MovimientoDTO{}
	for rows.Next() {
		var m application.MovimientoDTO
		var tipo string
		var venc sql.Null[fecha.Fecha]
		if err := rows.Scan(&m.ID, &m.Fecha, &m.ProductoID, &m.SKU, &m.Producto, &m.Cantidad, &tipo,
			&m.ReferenciaID, &m.Nota, &venc); err != nil {
			return nil, 0, err
		}
		m.Tipo = domain.TipoMovimiento(tipo)
		if venc.Valid {
			m.LoteVencimiento = &venc.V
		}
		out = append(out, m)
	}
	return out, total, rows.Err()
}

func (r Repositorio) ListarVencimientos(ctx context.Context, f application.FiltroVencimientos, p paginacion.Solicitud) ([]application.VencimientoDTO, int, error) {
	where := []string{`l.fecha_vencimiento IS NOT NULL`}
	var args []any
	if f.Desde != nil {
		where, args = append(where, `l.fecha_vencimiento >= ?`), append(args, *f.Desde)
	}
	if f.Hasta != nil {
		where, args = append(where, `l.fecha_vencimiento < ?`), append(args, *f.Hasta)
	}
	base := `
		SELECT l.id, l.producto_id, p.sku, p.nombre, l.fecha_vencimiento, SUM(m.cantidad) AS stock
		FROM lote l
		JOIN movimiento_stock m ON m.lote_id = l.id
		JOIN producto p ON p.id = l.producto_id
		WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY l.id, l.producto_id, p.sku, p.nombre, l.fecha_vencimiento
		HAVING stock > 0`
	conn := db.Conn(ctx, r.DB)
	var total int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM (`+base+`) t`, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn.QueryContext(ctx, base+` ORDER BY l.fecha_vencimiento, p.nombre LIMIT ? OFFSET ?`,
		append(args, p.Limite(), p.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []application.VencimientoDTO{}
	for rows.Next() {
		var v application.VencimientoDTO
		if err := rows.Scan(&v.LoteID, &v.ProductoID, &v.SKU, &v.Producto, &v.Vencimiento, &v.Stock); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

func (r Repositorio) ContarAlertas(ctx context.Context, hoy, limite fecha.Fecha) (application.AlertasDTO, error) {
	var a application.AlertasDTO
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx, `
		SELECT
		  (SELECT COUNT(*) FROM v_stock_lote WHERE stock > 0 AND fecha_vencimiento < ?),
		  (SELECT COUNT(*) FROM v_stock_lote WHERE stock > 0 AND fecha_vencimiento >= ? AND fecha_vencimiento < ?),
		  (SELECT COUNT(*) FROM producto p JOIN v_stock_producto s ON s.producto_id = p.id
		    WHERE p.activo AND s.stock <= 0),
		  (SELECT COUNT(*) FROM producto p JOIN v_stock_producto s ON s.producto_id = p.id
		    WHERE p.activo AND s.stock > 0 AND s.stock <= p.stock_minimo)`,
		hoy, hoy, limite).Scan(&a.Vencidos, &a.PorVencer, &a.SinStock, &a.StockBajo)
	return a, err
}

func (r Repositorio) DiasAlerta(ctx context.Context) (int, error) {
	var valor string
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx, `SELECT valor FROM configuracion WHERE clave = ?`, claveDiasAlerta).Scan(&valor)
	if errors.Is(err, sql.ErrNoRows) {
		return DiasAlertaPorDefecto, nil
	}
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(valor)
	if err != nil || n <= 0 {
		return DiasAlertaPorDefecto, nil
	}
	return n, nil
}

func (r Repositorio) GuardarDiasAlerta(ctx context.Context, dias int) error {
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`INSERT INTO configuracion (clave, valor) VALUES (?, ?) ON DUPLICATE KEY UPDATE valor = VALUES(valor)`,
		claveDiasAlerta, strconv.Itoa(dias))
	return err
}
