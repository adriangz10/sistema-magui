package mysql

import (
	"context"
	"database/sql"
	"errors"

	"magui/internal/catalogo/domain"
	"magui/internal/platform/db"
)

// Imagenes implementa application.RepositorioImagenes (fotos de productos, US-004).
type Imagenes struct{ DB *sql.DB }

func (r Imagenes) Guardar(ctx context.Context, productoID string, datos []byte, tipo string) (int, error) {
	conn := db.Conn(ctx, r.DB)
	if datos == nil {
		var existe bool
		if err := conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM producto WHERE id = ?)`, productoID).Scan(&existe); err != nil {
			return 0, err
		}
		if !existe {
			return 0, domain.ErrProductoNoEncontrado
		}
		_, err := conn.ExecContext(ctx, `DELETE FROM producto_imagen WHERE producto_id = ?`, productoID)
		return 0, err
	}
	// imagen_version solo sube (nunca se reutiliza): la URL con ?v= se guarda en caché para siempre.
	res, err := conn.ExecContext(ctx, `UPDATE producto SET imagen_version = imagen_version + 1 WHERE id = ?`, productoID)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, domain.ErrProductoNoEncontrado
	}
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO producto_imagen (producto_id, datos, tipo) VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE datos = VALUES(datos), tipo = VALUES(tipo), updated_at = CURRENT_TIMESTAMP(3)`,
		productoID, datos, tipo); err != nil {
		return 0, err
	}
	var version int
	err = conn.QueryRowContext(ctx, `SELECT imagen_version FROM producto WHERE id = ?`, productoID).Scan(&version)
	return version, err
}

func (r Imagenes) Obtener(ctx context.Context, productoID string) ([]byte, string, error) {
	var datos []byte
	var tipo string
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx, `SELECT datos, tipo FROM producto_imagen WHERE producto_id = ?`, productoID).
		Scan(&datos, &tipo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", domain.ErrSinImagen
	}
	return datos, tipo, err
}
