package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"magui/internal/caja/application"
	"magui/internal/caja/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/paginacion"
	"magui/internal/platform/db"
)

// Sesiones implementa application.RepositorioSesiones (apertura y cierre de caja, US-002).
type Sesiones struct{ DB *sql.DB }

const mysqlClaveDuplicada = 1062 // ER_DUP_ENTRY

const columnasSesion = `id, abierta_en, cerrada_en, monto_inicial, notas, cantidad_ventas, version, created_at, updated_at`

func scanSesion(s interface{ Scan(...any) error }) (domain.Sesion, error) {
	var x domain.Sesion
	err := s.Scan(&x.ID, &x.AbiertaEn, &x.CerradaEn, &x.MontoInicial, &x.Notas, &x.CantidadVentas, &x.Version, &x.CreatedAt, &x.UpdatedAt)
	return x, err
}

func (r Sesiones) Crear(ctx context.Context, s domain.Sesion) error {
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`INSERT INTO caja_sesion (id, abierta_en, monto_inicial, notas, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.AbiertaEn.UTC(), s.MontoInicial.String(), s.Notas, s.Version, s.CreatedAt.UTC(), s.UpdatedAt.UTC())
	var me *mysql.MySQLError
	if errors.As(err, &me) && me.Number == mysqlClaveDuplicada {
		return domain.ErrCajaYaAbierta // el índice único sobre "abierta" lo impide aunque se crucen dos pedidos
	}
	return err
}

func (r Sesiones) Obtener(ctx context.Context, id string) (domain.Sesion, error) {
	s, err := scanSesion(db.Conn(ctx, r.DB).QueryRowContext(ctx, `SELECT `+columnasSesion+` FROM caja_sesion WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Sesion{}, domain.ErrCajaNoEncontrada
	}
	if err != nil {
		return domain.Sesion{}, err
	}
	arqueos, err := r.arqueos(ctx, []string{s.ID})
	s.Arqueo = arqueos[s.ID]
	return s, err
}

func (r Sesiones) Abierta(ctx context.Context) (*domain.Sesion, error) {
	s, err := scanSesion(db.Conn(ctx, r.DB).QueryRowContext(ctx,
		`SELECT `+columnasSesion+` FROM caja_sesion WHERE cerrada_en IS NULL ORDER BY abierta_en DESC LIMIT 1`))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r Sesiones) Actualizar(ctx context.Context, s domain.Sesion, versionAnterior int) error {
	conn := db.Conn(ctx, r.DB)
	var cerrada any
	if s.CerradaEn != nil {
		cerrada = s.CerradaEn.UTC()
	}
	res, err := conn.ExecContext(ctx,
		`UPDATE caja_sesion SET cerrada_en = ?, monto_inicial = ?, notas = ?, cantidad_ventas = ?, version = ?, updated_at = ?
		 WHERE id = ? AND version = ?`,
		cerrada, s.MontoInicial.String(), s.Notas, s.CantidadVentas, s.Version, s.UpdatedAt.UTC(), s.ID, versionAnterior)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrVersionDesactualizada
	}
	if s.Abierta() {
		return nil
	}
	if _, err := conn.ExecContext(ctx, `DELETE FROM caja_arqueo WHERE sesion_id = ?`, s.ID); err != nil {
		return err
	}
	for _, a := range s.Arqueo {
		contado := dinero.Cero
		if a.Contado != nil {
			contado = *a.Contado
		}
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO caja_arqueo (sesion_id, metodo, ventas, gastos, esperado, contado) VALUES (?, ?, ?, ?, ?, ?)`,
			s.ID, string(a.Metodo), a.Ventas.String(), a.Gastos.String(), a.Esperado.String(), contado.String()); err != nil {
			return err
		}
	}
	return nil
}

// arqueos carga el arqueo de varias cajas, en el orden de domain.Medios.
func (r Sesiones) arqueos(ctx context.Context, ids []string) (map[string][]domain.Arqueo, error) {
	out := map[string][]domain.Arqueo{}
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := db.Conn(ctx, r.DB).QueryContext(ctx,
		`SELECT sesion_id, metodo, ventas, gastos, esperado, contado FROM caja_arqueo
		 WHERE sesion_id IN (?`+strings.Repeat(", ?", len(ids)-1)+`)
		 ORDER BY sesion_id, FIELD(metodo, 'EFECTIVO', 'TRANSFERENCIA', 'QR', 'TARJETA')`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, metodo string
		var a domain.Arqueo
		var contado dinero.Dinero
		if err := rows.Scan(&id, &metodo, &a.Ventas, &a.Gastos, &a.Esperado, &contado); err != nil {
			return nil, err
		}
		a.Metodo, a.Contado = domain.Metodo(metodo), &contado
		out[id] = append(out[id], a)
	}
	return out, rows.Err()
}

func (r Sesiones) Listar(ctx context.Context, p paginacion.Solicitud) ([]domain.Sesion, int, error) {
	conn := db.Conn(ctx, r.DB)
	var total int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM caja_sesion`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn.QueryContext(ctx, `SELECT `+columnasSesion+` FROM caja_sesion ORDER BY abierta_en DESC, id DESC LIMIT ? OFFSET ?`,
		p.Limite(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	var out []domain.Sesion
	var ids []string
	for rows.Next() {
		s, err := scanSesion(rows)
		if err != nil {
			rows.Close()
			return nil, 0, err
		}
		out, ids = append(out, s), append(ids, s.ID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	arqueos, err := r.arqueos(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		out[i].Arqueo = arqueos[out[i].ID]
	}
	return out, total, nil
}

func (r Sesiones) GuardarEdicion(ctx context.Context, e application.EdicionCaja) error {
	cambios, err := json.Marshal(e.Cambios)
	if err != nil {
		return err
	}
	_, err = db.Conn(ctx, r.DB).ExecContext(ctx,
		`INSERT INTO caja_edicion (id, sesion_id, fecha, motivo, version, cambios) VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.SesionID, e.Fecha.UTC(), e.Motivo, e.Version, string(cambios))
	return err
}

func (r Sesiones) Ediciones(ctx context.Context, sesionID string) ([]application.EdicionCaja, error) {
	rows, err := db.Conn(ctx, r.DB).QueryContext(ctx,
		`SELECT id, fecha, motivo, version, cambios FROM caja_edicion WHERE sesion_id = ? ORDER BY version DESC`, sesionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []application.EdicionCaja
	for rows.Next() {
		e := application.EdicionCaja{SesionID: sesionID}
		var cambios []byte
		if err := rows.Scan(&e.ID, &e.Fecha, &e.Motivo, &e.Version, &cambios); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(cambios, &e.Cambios); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Totales suma ventas confirmadas (por medio de pago) y gastos no anulados en [desde, hasta).
func (r Sesiones) Totales(ctx context.Context, desde, hasta time.Time) (domain.Totales, int, error) {
	conn := db.Conn(ctx, r.DB)
	t := domain.Totales{}
	sumar := func(consulta string, aplicar func(m *domain.Movimientos, v dinero.Dinero)) error {
		rows, err := conn.QueryContext(ctx, consulta, desde.UTC(), hasta.UTC())
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var metodo string
			var v dinero.Dinero
			if err := rows.Scan(&metodo, &v); err != nil {
				return err
			}
			m := t[domain.Metodo(metodo)]
			aplicar(&m, v)
			t[domain.Metodo(metodo)] = m
		}
		return rows.Err()
	}
	if err := sumar(`SELECT vp.metodo, SUM(vp.monto) FROM venta v JOIN venta_pago vp ON vp.venta_id = v.id
		WHERE v.estado = 'CONFIRMADA' AND v.fecha >= ? AND v.fecha < ? GROUP BY vp.metodo`,
		func(m *domain.Movimientos, v dinero.Dinero) { m.Ventas = v }); err != nil {
		return nil, 0, err
	}
	if err := sumar(`SELECT metodo, SUM(monto) FROM gasto_caja
		WHERE anulado_en IS NULL AND fecha >= ? AND fecha < ? GROUP BY metodo`,
		func(m *domain.Movimientos, v dinero.Dinero) { m.Gastos = v }); err != nil {
		return nil, 0, err
	}
	var n int
	err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM venta WHERE estado = 'CONFIRMADA' AND fecha >= ? AND fecha < ?`,
		desde.UTC(), hasta.UTC()).Scan(&n)
	return t, n, err
}
