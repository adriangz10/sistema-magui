// Package mysql implementa el repositorio de inventario sobre MySQL.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"slices"
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

// ObtenerLote lee el lote y su stock. Con bloquear toma la fila del lote FOR UPDATE (nadie puede
// insertar movimientos de ese lote hasta el fin de la transacción: la FK toma un bloqueo compartido
// sobre el lote) y recién entonces lee el stock. Las transacciones corren en READ COMMITTED, así
// que esa lectura común ve lo último confirmado (ADR 0002).
func (r Repositorio) ObtenerLote(ctx context.Context, id string, bloquear bool) (domain.LoteStock, error) {
	conn := db.Conn(ctx, r.DB)
	if bloquear {
		var x string
		err := conn.QueryRowContext(ctx, `SELECT id FROM lote WHERE id = ? FOR UPDATE`, id).Scan(&x)
		if errors.Is(err, sql.ErrNoRows) {
			return domain.LoteStock{}, domain.ErrLoteNoEncontrado
		}
		if err != nil {
			return domain.LoteStock{}, err
		}
	}
	l, err := scanLote(conn.QueryRowContext(ctx, consultaLotes+` WHERE l.id = ? GROUP BY l.id`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.LoteStock{}, domain.ErrLoteNoEncontrado
	}
	return l, err
}

// idsLotesConStock devuelve (sin bloquear) los lotes del producto con stock distinto de cero.
// Los agotados no hace falta bloquearlos: ninguna salida puede tomar unidades de ellos.
func (r Repositorio) idsLotesConStock(ctx context.Context, productoID string) ([]string, error) {
	rows, err := db.Conn(ctx, r.DB).QueryContext(ctx, `
		SELECT l.id FROM lote l JOIN movimiento_stock m ON m.lote_id = l.id
		WHERE l.producto_id = ? GROUP BY l.id HAVING SUM(m.cantidad) <> 0`, productoID)
	if err != nil {
		return nil, err
	}
	return leerIDs(rows)
}

func leerIDs(rows *sql.Rows) ([]string, error) {
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// bloquearIDs toma FOR UPDATE sobre los lotes indicados, siempre ordenados por id y con la clave
// primaria como único criterio (orden de bloqueo fijo: evita deadlocks entre operaciones).
func bloquearIDs(ctx context.Context, conn db.Ejecutor, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	ordenados := slices.Clone(ids)
	slices.Sort(ordenados)
	args := make([]any, len(ordenados))
	for i, id := range ordenados {
		args[i] = id
	}
	rows, err := conn.QueryContext(ctx, `SELECT id FROM lote WHERE id IN (`+marcas(len(args))+`) ORDER BY id FOR UPDATE`, args...)
	if err != nil {
		return err
	}
	return rows.Close()
}

// BloquearLotesDeProducto toma FOR UPDATE sobre los lotes con stock del producto, ordenados por id.
func (r Repositorio) BloquearLotesDeProducto(ctx context.Context, productoID string) error {
	ids, err := r.idsLotesConStock(ctx, productoID)
	if err != nil {
		return err
	}
	return bloquearIDs(ctx, db.Conn(ctx, r.DB), ids)
}

// intentosEstabilizar acota el ciclo de LotesConStock: un lote que pasó de stock 0 a distinto de
// cero entre la resolución de ids y la lectura se bloquea y se vuelve a leer.
const intentosEstabilizar = 5

func (r Repositorio) LotesConStock(ctx context.Context, productoID string, bloquear bool) ([]domain.LoteStock, error) {
	conn := db.Conn(ctx, r.DB)
	consulta := consultaLotes + `
		WHERE l.producto_id = ? GROUP BY l.id HAVING stock <> 0
		ORDER BY l.fecha_vencimiento IS NULL, l.fecha_vencimiento, l.created_at`
	if !bloquear {
		rows, err := conn.QueryContext(ctx, consulta, productoID)
		if err != nil {
			return nil, err
		}
		return scanLotes(rows)
	}
	bloqueados := map[string]bool{}
	for range intentosEstabilizar {
		ids, err := r.idsLotesConStock(ctx, productoID)
		if err != nil {
			return nil, err
		}
		if err := bloquearIDs(ctx, conn, ids); err != nil {
			return nil, err
		}
		for _, id := range ids {
			bloqueados[id] = true
		}
		rows, err := conn.QueryContext(ctx, consulta, productoID)
		if err != nil {
			return nil, err
		}
		lotes, err := scanLotes(rows)
		if err != nil {
			return nil, err
		}
		if todosBloqueados(lotes, bloqueados) {
			return lotes, nil
		}
	}
	return nil, db.ErrOperacionConcurrente
}

func todosBloqueados(lotes []domain.LoteStock, bloqueados map[string]bool) bool {
	for _, l := range lotes {
		if !bloqueados[l.ID] {
			return false
		}
	}
	return true
}

func scanLotes(rows *sql.Rows) ([]domain.LoteStock, error) {
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

func marcas(n int) string { return strings.TrimSuffix(strings.Repeat("?, ", n), ", ") }

// LotesPorCompraItem devuelve los lotes de los renglones de compra. Con bloquear, primero resuelve
// los ids de lote (la relación renglón-lote no cambia), los bloquea ordenados por id y recién
// entonces lee el stock (READ COMMITTED: ve lo último confirmado).
func (r Repositorio) LotesPorCompraItem(ctx context.Context, itemIDs []string, bloquear bool) ([]domain.LoteStock, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}
	conn := db.Conn(ctx, r.DB)
	args := make([]any, len(itemIDs))
	for i, id := range itemIDs {
		args[i] = id
	}
	if bloquear {
		ids, err := idsDeLotes(ctx, conn, itemIDs, args)
		if err != nil {
			return nil, err
		}
		if err := bloquearIDs(ctx, conn, ids); err != nil {
			return nil, err
		}
	}
	rows, err := conn.QueryContext(ctx, consultaLotes+` WHERE l.compra_item_id IN (`+marcas(len(itemIDs))+`) GROUP BY l.id ORDER BY l.id`, args...)
	if err != nil {
		return nil, err
	}
	return scanLotes(rows)
}

func idsDeLotes(ctx context.Context, conn db.Ejecutor, itemIDs []string, args []any) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `SELECT id FROM lote WHERE compra_item_id IN (`+marcas(len(itemIDs))+`)`, args...)
	if err != nil {
		return nil, err
	}
	return leerIDs(rows)
}

func (r Repositorio) ActualizarLote(ctx context.Context, id string, costo *dinero.Dinero, vencimiento *fecha.Fecha) error {
	var c any
	if costo != nil {
		c = costo.String()
	}
	var v any
	if vencimiento != nil && !vencimiento.IsZero() {
		v = *vencimiento
	}
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx, `UPDATE lote SET costo_unitario = ?, fecha_vencimiento = ? WHERE id = ?`, c, v, id)
	return err
}

func (r Repositorio) StockProducto(ctx context.Context, productoID string) (int, error) {
	var n int
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx,
		`SELECT COALESCE(SUM(cantidad), 0) FROM movimiento_stock WHERE producto_id = ?`, productoID).Scan(&n)
	return n, err
}

func (r Repositorio) StockSinLote(ctx context.Context, productoID string) (int, error) {
	var n int
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx,
		`SELECT COALESCE(SUM(cantidad), 0) FROM movimiento_stock WHERE producto_id = ? AND lote_id IS NULL`, productoID).Scan(&n)
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
