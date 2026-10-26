// Package sqlite implementa la base local del escritorio: copia del catálogo, cola de ventas,
// sesión y los puertos de ventas (para cobrar con las mismas reglas que el servidor).
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"magui/internal/escritorio/application"
	"magui/internal/kernel/dinero"
	"magui/internal/platform/db"
	sincro "magui/internal/sync/application"
)

// Base agrupa los adaptadores sobre el mismo archivo SQLite.
type Base struct{ DB *sql.DB }

func fechaTexto(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func marcadores(n int) string { return strings.TrimSuffix(strings.Repeat("?, ", n), ", ") }

func comoArgs(ids []string) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

// --- Cola (implementa application.Cola) ---

func (b Base) Pendientes(ctx context.Context, limite int) ([]sincro.VentaOffline, error) {
	rows, err := db.Conn(ctx, b.DB).QueryContext(ctx,
		`SELECT envio FROM venta_local WHERE estado = 'PENDIENTE' ORDER BY fecha, id LIMIT ?`, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []sincro.VentaOffline
	for rows.Next() {
		var crudo string
		if err := rows.Scan(&crudo); err != nil {
			return nil, err
		}
		var v sincro.VentaOffline
		if err := json.Unmarshal([]byte(crudo), &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (b Base) MarcarEnviadas(ctx context.Context, ids []string, en time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	args := append([]any{fechaTexto(en)}, comoArgs(ids)...)
	_, err := db.Conn(ctx, b.DB).ExecContext(ctx,
		`UPDATE venta_local SET estado = 'ENVIADA', enviada_en = ?, en_copia = 0, error = NULL
		 WHERE id IN (`+marcadores(len(ids))+`)`, args...)
	return err
}

func (b Base) MarcarRechazada(ctx context.Context, id, motivo string) error {
	_, err := db.Conn(ctx, b.DB).ExecContext(ctx,
		`UPDATE venta_local SET estado = 'RECHAZADA', error = ? WHERE id = ?`, motivo, id)
	return err
}

func (b Base) Contar(ctx context.Context) (pendientes, rechazadas int, err error) {
	err = db.Conn(ctx, b.DB).QueryRowContext(ctx, `
		SELECT COALESCE(SUM(estado = 'PENDIENTE'), 0), COALESCE(SUM(estado = 'RECHAZADA'), 0) FROM venta_local`).
		Scan(&pendientes, &rechazadas)
	return
}

func (b Base) ReemplazarCopia(ctx context.Context, c sincro.CopiaCatalogo) error {
	return db.Transactor{DB: b.DB}.EnTransaccion(ctx, func(ctx context.Context) error {
		conn := db.Conn(ctx, b.DB)
		for _, q := range []string{`DELETE FROM copia_componente`, `DELETE FROM copia_producto`} {
			if _, err := conn.ExecContext(ctx, q); err != nil {
				return err
			}
		}
		for _, p := range c.Productos {
			if _, err := conn.ExecContext(ctx, `INSERT INTO copia_producto
				(id, sku, codigo_barras, nombre, tipo, empresa, precio_venta, stock, activo, version)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				p.ID, p.SKU, p.CodigoBarras, p.Nombre, p.Tipo, p.Empresa, p.PrecioVenta.Centavos(), p.Stock, p.Activo, p.Version); err != nil {
				return err
			}
		}
		for _, x := range c.Componentes {
			if _, err := conn.ExecContext(ctx, `INSERT INTO copia_componente (combo_id, producto_id, cantidad) VALUES (?, ?, ?)`,
				x.ComboID, x.ProductoID, x.Cantidad); err != nil {
				return err
			}
		}
		// La copia se pidió después de subir estas ventas: su stock ya las descuenta.
		if _, err := conn.ExecContext(ctx, `UPDATE venta_local SET en_copia = 1 WHERE estado = 'ENVIADA'`); err != nil {
			return err
		}
		return guardarAjuste(ctx, conn, "copia_generada_en", fechaTexto(c.GeneradaEn))
	})
}

func (b Base) HayCopia(ctx context.Context) (bool, time.Time, error) {
	v, ok, err := leerAjuste(ctx, db.Conn(ctx, b.DB), "copia_generada_en")
	if err != nil || !ok {
		return false, time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	return err == nil, t, err
}

// consultaStock: stock de la copia menos lo vendido en esta PC que la copia todavía no refleja
// (ventas pendientes, enviadas antes de bajar una copia nueva y la venta que se está cobrando).
const consultaStock = `
	SELECT p.id, p.sku, p.codigo_barras, p.nombre, p.tipo, p.empresa, p.precio_venta,
	       p.stock - COALESCE(s.vendido, 0), p.activo, p.version
	FROM copia_producto p
	LEFT JOIN (
		SELECT sl.producto_id, SUM(sl.cantidad) AS vendido
		FROM salida_local sl LEFT JOIN venta_local v ON v.id = sl.venta_id
		WHERE v.id IS NULL OR v.estado = 'PENDIENTE' OR (v.estado = 'ENVIADA' AND v.en_copia = 0)
		GROUP BY sl.producto_id
	) s ON s.producto_id = p.id`

func (b Base) Productos(ctx context.Context) ([]sincro.ProductoCopia, error) {
	conn := db.Conn(ctx, b.DB)
	rows, err := conn.QueryContext(ctx, consultaStock+` ORDER BY p.nombre, p.id`)
	if err != nil {
		return nil, err
	}
	var out []sincro.ProductoCopia
	idx := map[string]int{}
	for rows.Next() {
		p, err := scanProducto(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		idx[p.ID] = len(out)
		out = append(out, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Combos: cuántos se pueden armar con el stock local de sus componentes.
	comps, err := componentes(ctx, conn, nil)
	if err != nil {
		return nil, err
	}
	stock := func(id string) int {
		if j, ok := idx[id]; ok {
			return out[j].Stock
		}
		return 0
	}
	for i := range out {
		if out[i].Tipo == "COMBO" {
			out[i].Stock = stockCombo(comps[out[i].ID], stock)
		}
	}
	return out, nil
}

func scanProducto(s interface{ Scan(...any) error }) (sincro.ProductoCopia, error) {
	var p sincro.ProductoCopia
	var precio int64
	err := s.Scan(&p.ID, &p.SKU, &p.CodigoBarras, &p.Nombre, &p.Tipo, &p.Empresa, &precio, &p.Stock, &p.Activo, &p.Version)
	p.PrecioVenta = dinero.DesdeCentavos(precio)
	return p, err
}

type componente struct {
	productoID string
	cantidad   int
}

// componentes devuelve los componentes por combo (de todos, o solo de los combos indicados).
func componentes(ctx context.Context, conn db.Ejecutor, combos []string) (map[string][]componente, error) {
	q := `SELECT combo_id, producto_id, cantidad FROM copia_componente`
	if combos != nil {
		if len(combos) == 0 {
			return map[string][]componente{}, nil
		}
		q += ` WHERE combo_id IN (` + marcadores(len(combos)) + `)`
	}
	rows, err := conn.QueryContext(ctx, q+` ORDER BY combo_id, producto_id`, comoArgs(combos)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]componente{}
	for rows.Next() {
		var combo string
		var c componente
		if err := rows.Scan(&combo, &c.productoID, &c.cantidad); err != nil {
			return nil, err
		}
		out[combo] = append(out[combo], c)
	}
	return out, rows.Err()
}

func stockCombo(comps []componente, stock func(string) int) int {
	if len(comps) == 0 {
		return 0
	}
	minimo := -1
	for _, c := range comps {
		n := max(0, stock(c.productoID)) / max(1, c.cantidad)
		if minimo < 0 || n < minimo {
			minimo = n
		}
	}
	return minimo
}

// --- Ajustes y sesión ---

func leerAjuste(ctx context.Context, conn db.Ejecutor, clave string) (string, bool, error) {
	var v string
	err := conn.QueryRowContext(ctx, `SELECT valor FROM ajuste WHERE clave = ?`, clave).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return v, err == nil, err
}

func guardarAjuste(ctx context.Context, conn db.Ejecutor, clave, valor string) error {
	_, err := conn.ExecContext(ctx,
		`INSERT INTO ajuste (clave, valor) VALUES (?, ?) ON CONFLICT (clave) DO UPDATE SET valor = excluded.valor`, clave, valor)
	return err
}

// Sesiones implementa application.Sesiones.
type Sesiones struct{ DB *sql.DB }

func (s Sesiones) Leer(ctx context.Context) (application.Sesion, error) {
	v, ok, err := leerAjuste(ctx, db.Conn(ctx, s.DB), "sesion")
	if err != nil || !ok {
		return application.Sesion{}, err
	}
	var ses application.Sesion
	return ses, json.Unmarshal([]byte(v), &ses)
}

func (s Sesiones) Guardar(ctx context.Context, ses application.Sesion) error {
	b, err := json.Marshal(ses)
	if err != nil {
		return err
	}
	return guardarAjuste(ctx, db.Conn(ctx, s.DB), "sesion", string(b))
}
