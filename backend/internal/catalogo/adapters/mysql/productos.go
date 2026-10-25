package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"magui/internal/catalogo/application"
	"magui/internal/catalogo/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/paginacion"
	"magui/internal/platform/db"
)

// Productos implementa application.RepositorioProductos.
type Productos struct{ DB *sql.DB }

// --- Escritura ---

func (r Productos) Crear(ctx context.Context, p domain.Producto) error {
	conn := db.Conn(ctx, r.DB)
	_, err := conn.ExecContext(ctx, `
		INSERT INTO producto (id, sku, codigo_barras, nombre, descripcion, empresa_id, tipo, precio_venta, costo,
		                      stock_minimo, mostrar_en_tienda, activo, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.SKU, p.CodigoBarras, p.Nombre, p.Descripcion, p.EmpresaID, string(p.Tipo), p.PrecioVenta.String(),
		dineroOpcional(p.Costo), p.StockMinimo, p.MostrarEnTienda, p.Activo, p.Version, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return traducirDuplicado(err)
	}
	return r.guardarRelaciones(ctx, conn, p)
}

func (r Productos) Actualizar(ctx context.Context, p domain.Producto, versionAnterior int) error {
	conn := db.Conn(ctx, r.DB)
	res, err := conn.ExecContext(ctx, `
		UPDATE producto SET sku = ?, codigo_barras = ?, nombre = ?, descripcion = ?, empresa_id = ?, precio_venta = ?,
		       costo = ?, stock_minimo = ?, mostrar_en_tienda = ?, activo = ?, version = ?, updated_at = ?
		WHERE id = ? AND version = ?`,
		p.SKU, p.CodigoBarras, p.Nombre, p.Descripcion, p.EmpresaID, p.PrecioVenta.String(),
		dineroOpcional(p.Costo), p.StockMinimo, p.MostrarEnTienda, p.Activo, p.Version, p.UpdatedAt,
		p.ID, versionAnterior)
	if err != nil {
		return traducirDuplicado(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := r.ObtenerPorID(ctx, p.ID); err != nil {
			return err
		}
		return domain.ErrVersionDesactualizada
	}
	if _, err := conn.ExecContext(ctx, `DELETE FROM producto_categoria WHERE producto_id = ?`, p.ID); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `DELETE FROM combo_item WHERE combo_id = ?`, p.ID); err != nil {
		return err
	}
	return r.guardarRelaciones(ctx, conn, p)
}

func (r Productos) guardarRelaciones(ctx context.Context, conn db.Ejecutor, p domain.Producto) error {
	if len(p.CategoriaIDs) > 0 {
		args := make([]any, 0, 2*len(p.CategoriaIDs))
		for _, c := range p.CategoriaIDs {
			args = append(args, p.ID, c)
		}
		q := `INSERT INTO producto_categoria (producto_id, categoria_id) VALUES ` +
			strings.TrimSuffix(strings.Repeat("(?, ?), ", len(p.CategoriaIDs)), ", ")
		if _, err := conn.ExecContext(ctx, q, args...); err != nil {
			return err
		}
	}
	if len(p.Items) > 0 {
		args := make([]any, 0, 3*len(p.Items))
		for _, it := range p.Items {
			args = append(args, p.ID, it.ProductoID, it.Cantidad)
		}
		q := `INSERT INTO combo_item (combo_id, producto_id, cantidad) VALUES ` +
			strings.TrimSuffix(strings.Repeat("(?, ?, ?), ", len(p.Items)), ", ")
		if _, err := conn.ExecContext(ctx, q, args...); err != nil {
			return err
		}
	}
	return nil
}

func traducirDuplicado(err error) error {
	switch {
	case esDuplicado(err, "uq_producto_sku"):
		return domain.ErrSKUDuplicado
	case esDuplicado(err, "uq_producto_codigo_barras"):
		return domain.ErrCodigoBarrasDuplicado
	}
	return err
}

func (r Productos) ActualizarPrecios(ctx context.Context, productos []domain.Producto) error {
	conn := db.Conn(ctx, r.DB)
	for _, p := range productos {
		if _, err := conn.ExecContext(ctx,
			`UPDATE producto SET precio_venta = ?, version = ?, updated_at = ? WHERE id = ?`,
			p.PrecioVenta.String(), p.Version, p.UpdatedAt, p.ID); err != nil {
			return err
		}
	}
	return nil
}

func (r Productos) RegistrarCambiosPrecio(ctx context.Context, cambios []domain.CambioPrecio) error {
	if len(cambios) == 0 {
		return nil
	}
	args := make([]any, 0, 7*len(cambios))
	for _, c := range cambios {
		args = append(args, c.ID, c.ProductoID, c.PrecioAnterior.String(), c.PrecioNuevo.String(),
			string(c.Origen), c.Motivo, c.Fecha)
	}
	q := `INSERT INTO precio_historial (id, producto_id, precio_anterior, precio_nuevo, origen, motivo, fecha) VALUES ` +
		strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?, ?, ?), ", len(cambios)), ", ")
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx, q, args...)
	return err
}

// --- Lectura de entidades ---

const columnasProducto = `p.id, p.sku, p.codigo_barras, p.nombre, p.descripcion, p.empresa_id, p.tipo, p.precio_venta,
	p.costo, p.stock_minimo, p.mostrar_en_tienda, p.activo, p.version, p.created_at, p.updated_at`

func scanProducto(s interface{ Scan(...any) error }) (domain.Producto, error) {
	var p domain.Producto
	var tipo string
	var costo sql.Null[dinero.Dinero]
	err := s.Scan(&p.ID, &p.SKU, &p.CodigoBarras, &p.Nombre, &p.Descripcion, &p.EmpresaID, &tipo, &p.PrecioVenta,
		&costo, &p.StockMinimo, &p.MostrarEnTienda, &p.Activo, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	p.Tipo = domain.Tipo(tipo)
	if costo.Valid {
		p.Costo = &costo.V
	}
	return p, err
}

func (r Productos) ObtenerPorID(ctx context.Context, id string) (domain.Producto, error) {
	conn := db.Conn(ctx, r.DB)
	p, err := scanProducto(conn.QueryRowContext(ctx, `SELECT `+columnasProducto+` FROM producto p WHERE p.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Producto{}, domain.ErrProductoNoEncontrado
	}
	if err != nil {
		return domain.Producto{}, err
	}
	if p.CategoriaIDs, err = r.categoriasDe(ctx, conn, id); err != nil {
		return domain.Producto{}, err
	}
	rows, err := conn.QueryContext(ctx, `SELECT producto_id, cantidad FROM combo_item WHERE combo_id = ? ORDER BY producto_id`, id)
	if err != nil {
		return domain.Producto{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var it domain.ComboItem
		if err := rows.Scan(&it.ProductoID, &it.Cantidad); err != nil {
			return domain.Producto{}, err
		}
		p.Items = append(p.Items, it)
	}
	return p, rows.Err()
}

func (r Productos) categoriasDe(ctx context.Context, conn db.Ejecutor, id string) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `SELECT categoria_id FROM producto_categoria WHERE producto_id = ?`, id)
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

// ObtenerVarios trae los datos básicos (sin categorías ni componentes) de los productos indicados.
func (r Productos) ObtenerVarios(ctx context.Context, ids []string) ([]domain.Producto, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return r.consultarProductos(ctx, `SELECT `+columnasProducto+` FROM producto p WHERE p.id IN (`+marcadores(len(ids))+`)`,
		comoArgs(ids)...)
}

// ActivosEnCategorias trae los datos básicos (sin categorías ni componentes).
func (r Productos) ActivosEnCategorias(ctx context.Context, categoriaIDs []string) ([]domain.Producto, error) {
	if len(categoriaIDs) == 0 {
		return nil, nil
	}
	return r.consultarProductos(ctx, `SELECT `+columnasProducto+` FROM producto p
		WHERE p.activo AND EXISTS (SELECT 1 FROM producto_categoria pc
		                           WHERE pc.producto_id = p.id AND pc.categoria_id IN (`+marcadores(len(categoriaIDs))+`))
		ORDER BY p.nombre, p.id`, comoArgs(categoriaIDs)...)
}

func (r Productos) consultarProductos(ctx context.Context, q string, args ...any) ([]domain.Producto, error) {
	rows, err := db.Conn(ctx, r.DB).QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Producto
	for rows.Next() {
		p, err := scanProducto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r Productos) CodigoEnUso(ctx context.Context, codigo, excluirID string) (bool, error) {
	var enUso bool
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM producto WHERE id <> ? AND (sku = ? OR codigo_barras = ?))`,
		excluirID, codigo, codigo).Scan(&enUso)
	return enUso, err
}

func (r Productos) IDPorCodigo(ctx context.Context, codigo string) (string, error) {
	var id string
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx,
		`SELECT id FROM producto WHERE sku = ? OR codigo_barras = ? ORDER BY sku = ? DESC LIMIT 1`,
		codigo, codigo, codigo).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrProductoNoEncontrado
	}
	return id, err
}

// --- Modelos de lectura ---

// consultaResumen calcula el stock: suma de movimientos para simples; para combos,
// cuántos se pueden armar con el stock de sus componentes (nunca negativo).
const consultaResumen = `
	SELECT p.id, p.sku, p.codigo_barras, p.nombre, p.tipo, p.empresa_id, e.nombre AS empresa, p.precio_venta, p.costo,
	       CASE WHEN p.tipo = 'COMBO' THEN COALESCE((
	              SELECT GREATEST(0, MIN(FLOOR(COALESCE(sc.stock, 0) / ci.cantidad)))
	              FROM combo_item ci LEFT JOIN v_stock_producto sc ON sc.producto_id = ci.producto_id
	              WHERE ci.combo_id = p.id), 0)
	            ELSE COALESCE(s.stock, 0) END AS stock,
	       p.stock_minimo, p.activo, p.version, p.descripcion, p.mostrar_en_tienda, p.created_at, p.updated_at
	FROM producto p
	LEFT JOIN empresa e ON e.id = p.empresa_id
	LEFT JOIN (SELECT producto_id, SUM(cantidad) AS stock FROM movimiento_stock GROUP BY producto_id) s
	       ON s.producto_id = p.id`

func scanDetalle(s interface{ Scan(...any) error }) (application.ProductoDetalle, error) {
	var d application.ProductoDetalle
	var tipo string
	var costo sql.Null[dinero.Dinero]
	err := s.Scan(&d.ID, &d.SKU, &d.CodigoBarras, &d.Nombre, &tipo, &d.EmpresaID, &d.Empresa, &d.PrecioVenta, &costo,
		&d.Stock, &d.StockMinimo, &d.Activo, &d.Version, &d.Descripcion, &d.MostrarEnTienda, &d.CreatedAt, &d.UpdatedAt)
	d.Tipo = domain.Tipo(tipo)
	if costo.Valid {
		d.Costo = &costo.V
	}
	return d, err
}

func (r Productos) Listar(ctx context.Context, f application.FiltroProductos, p paginacion.Solicitud) ([]application.ProductoResumen, int, error) {
	var where []string
	var args []any
	if f.Texto != "" {
		where = append(where, `(p.nombre LIKE ? OR p.sku LIKE ? OR p.codigo_barras LIKE ?)`)
		patron := patronLike(f.Texto)
		args = append(args, patron, patron, patron)
	}
	if len(f.CategoriaIDs) > 0 {
		where = append(where, `EXISTS (SELECT 1 FROM producto_categoria pc WHERE pc.producto_id = p.id AND pc.categoria_id IN (`+
			marcadores(len(f.CategoriaIDs))+`))`)
		args = append(args, comoArgs(f.CategoriaIDs)...)
	}
	if f.EmpresaID != "" {
		where = append(where, `p.empresa_id = ?`)
		args = append(args, f.EmpresaID)
	}
	if f.Tipo != "" {
		where = append(where, `p.tipo = ?`)
		args = append(args, string(f.Tipo))
	}
	switch f.Estado {
	case application.EstadoActivos:
		where = append(where, `p.activo`)
	case application.EstadoInactivos:
		where = append(where, `NOT p.activo`)
	}
	base := consultaResumen
	if len(where) > 0 {
		base += " WHERE " + strings.Join(where, " AND ")
	}
	exterior := ""
	if f.StockBajo {
		exterior = ` WHERE t.tipo = 'SIMPLE' AND t.stock <= t.stock_minimo`
	}

	conn := db.Conn(ctx, r.DB)
	var total int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM (`+base+`) t`+exterior, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := `SELECT * FROM (` + base + `) t` + exterior + ` ORDER BY ` + ordenSQL(f.Orden) + ` LIMIT ? OFFSET ?`
	rows, err := conn.QueryContext(ctx, q, append(args, p.Limite(), p.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []application.ProductoResumen{}
	for rows.Next() {
		d, err := scanDetalle(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, d.ProductoResumen)
	}
	return out, total, rows.Err()
}

func ordenSQL(o application.Orden) string {
	switch o {
	case application.OrdenRecientes:
		return `t.created_at DESC, t.id DESC`
	case application.OrdenPrecio:
		return `t.precio_venta ASC, t.nombre, t.id`
	case application.OrdenStock:
		return `t.stock ASC, t.nombre, t.id`
	default:
		return `t.nombre, t.id`
	}
}

func (r Productos) ObtenerDetalle(ctx context.Context, id string) (application.ProductoDetalle, error) {
	conn := db.Conn(ctx, r.DB)
	d, err := scanDetalle(conn.QueryRowContext(ctx, consultaResumen+` WHERE p.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return application.ProductoDetalle{}, domain.ErrProductoNoEncontrado
	}
	if err != nil {
		return application.ProductoDetalle{}, err
	}

	d.Categorias = []application.CategoriaRef{}
	rows, err := conn.QueryContext(ctx, `
		SELECT c.id, c.nombre FROM producto_categoria pc JOIN categoria c ON c.id = pc.categoria_id
		WHERE pc.producto_id = ? ORDER BY c.nombre`, id)
	if err != nil {
		return application.ProductoDetalle{}, err
	}
	for rows.Next() {
		var c application.CategoriaRef
		if err := rows.Scan(&c.ID, &c.Nombre); err != nil {
			rows.Close()
			return application.ProductoDetalle{}, err
		}
		d.Categorias = append(d.Categorias, c)
	}
	rows.Close()

	d.Items = []application.ComboItemDTO{}
	rows, err = conn.QueryContext(ctx, `
		SELECT p.id, p.sku, p.nombre, ci.cantidad, p.precio_venta, COALESCE(s.stock, 0)
		FROM combo_item ci
		JOIN producto p ON p.id = ci.producto_id
		LEFT JOIN v_stock_producto s ON s.producto_id = p.id
		WHERE ci.combo_id = ? ORDER BY p.nombre`, id)
	if err != nil {
		return application.ProductoDetalle{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var it application.ComboItemDTO
		if err := rows.Scan(&it.ProductoID, &it.SKU, &it.Nombre, &it.Cantidad, &it.PrecioVenta, &it.Stock); err != nil {
			return application.ProductoDetalle{}, err
		}
		d.Items = append(d.Items, it)
	}
	return d, rows.Err()
}

func (r Productos) HistorialPrecios(ctx context.Context, productoID string, p paginacion.Solicitud) ([]application.CambioPrecioDTO, int, error) {
	conn := db.Conn(ctx, r.DB)
	var total int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM precio_historial WHERE producto_id = ?`, productoID).
		Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn.QueryContext(ctx, `
		SELECT id, precio_anterior, precio_nuevo, origen, COALESCE(motivo, ''), fecha FROM precio_historial
		WHERE producto_id = ? ORDER BY fecha DESC, id DESC LIMIT ? OFFSET ?`, productoID, p.Limite(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []application.CambioPrecioDTO{}
	for rows.Next() {
		var c application.CambioPrecioDTO
		var origen string
		if err := rows.Scan(&c.ID, &c.PrecioAnterior, &c.PrecioNuevo, &origen, &c.Motivo, &c.Fecha); err != nil {
			return nil, 0, err
		}
		c.Origen = domain.OrigenPrecio(origen)
		out = append(out, c)
	}
	return out, total, rows.Err()
}
