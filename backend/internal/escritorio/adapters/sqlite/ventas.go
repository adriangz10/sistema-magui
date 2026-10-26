package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/paginacion"
	"magui/internal/platform/db"
	sincro "magui/internal/sync/application"
	ventas "magui/internal/ventas/application"
	"magui/internal/ventas/domain"
)

// Adaptadores de los puertos de ventas sobre la base local: el escritorio cobra con el mismo
// caso de uso que el servidor (precios, combos, control de stock y pagos combinados).

// ErrNecesitaInternet: operaciones de ventas que solo se hacen contra el servidor.
var ErrNecesitaInternet = errores.Conflicto("necesita_internet", "Esta acción necesita conexión con el servidor")

// --- ventas.Catalogo: lee la copia del catálogo ---

type CatalogoVentas struct{ DB *sql.DB }

func (c CatalogoVentas) Productos(ctx context.Context, ids []string) (map[string]ventas.ProductoVenta, error) {
	out := map[string]ventas.ProductoVenta{}
	if len(ids) == 0 {
		return out, nil
	}
	conn := db.Conn(ctx, c.DB)
	rows, err := conn.QueryContext(ctx,
		`SELECT id, nombre, tipo, activo, precio_venta FROM copia_producto WHERE id IN (`+marcadores(len(ids))+`)`, comoArgs(ids)...)
	if err != nil {
		return nil, err
	}
	var combos []string
	for rows.Next() {
		var p ventas.ProductoVenta
		var tipo string
		var precio int64
		if err := rows.Scan(&p.ID, &p.Nombre, &tipo, &p.Activo, &precio); err != nil {
			rows.Close()
			return nil, err
		}
		p.Combo, p.Precio = tipo == "COMBO", dinero.DesdeCentavos(precio)
		if p.Combo {
			combos = append(combos, p.ID)
		}
		out[p.ID] = p // sin costo: lo calcula el servidor al recibir la venta
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	comps, err := componentes(ctx, conn, append([]string{}, combos...))
	if err != nil {
		return nil, err
	}
	for _, id := range combos {
		p := out[id]
		for _, x := range comps[id] {
			p.Componentes = append(p.Componentes, ventas.Componente{ProductoID: x.productoID, Cantidad: x.cantidad})
		}
		out[id] = p
	}
	return out, nil
}

// --- ventas.Inventario: controla el stock local y anota lo que sale ---

type InventarioVentas struct{ DB *sql.DB }

func (i InventarioVentas) Descontar(ctx context.Context, s ventas.SalidaVenta) ([]ventas.ParteSalida, error) {
	conn := db.Conn(ctx, i.DB)
	disponible := 0
	p, err := scanProducto(conn.QueryRowContext(ctx, consultaStock+` WHERE p.id = ?`, s.ProductoID))
	switch {
	case err == nil:
		disponible = p.Stock
	case !errors.Is(err, sql.ErrNoRows):
		return nil, err
	}
	if !s.PermitirSinStock && disponible < s.Cantidad {
		// mismo código y mensaje que el servidor: la pantalla ofrece "vender igual"
		return nil, errores.Conflicto("stock_insuficiente", fmt.Sprintf("No hay stock suficiente (disponible: %d)", max(0, disponible)))
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO salida_local (venta_id, producto_id, cantidad) VALUES (?, ?, ?)`,
		s.VentaID, s.ProductoID, s.Cantidad); err != nil {
		return nil, err
	}
	return []ventas.ParteSalida{{Cantidad: s.Cantidad}}, nil
}

func (InventarioVentas) Reponer(context.Context, string, string) error { return ErrNecesitaInternet }

// --- ventas.Repositorio: guarda la venta en la cola ---

type RepositorioVentas struct{ DB *sql.DB }

func (r RepositorioVentas) Guardar(ctx context.Context, v domain.Venta) error {
	conn := db.Conn(ctx, r.DB)
	v.Origen = domain.OrigenDesktop
	envio := sincro.VentaOffline{ID: v.ID, Fecha: v.Fecha.UTC()}
	for _, it := range v.Items {
		envio.Items = append(envio.Items, sincro.ItemOffline{ProductoID: it.ProductoID, Cantidad: it.Cantidad, Precio: it.PrecioUnitario})
	}
	for _, p := range v.Pagos {
		monto := p.Monto
		if p.MontoRecibido != nil { // efectivo: se envía lo que entregó el cliente, como en el cobro
			monto = *p.MontoRecibido
		}
		envio.Pagos = append(envio.Pagos, sincro.PagoOffline{Metodo: string(p.Metodo), Monto: monto})
	}
	det, err := r.armarDetalle(ctx, conn, v)
	if err != nil {
		return err
	}
	bEnvio, err := json.Marshal(envio)
	if err != nil {
		return err
	}
	bDet, err := json.Marshal(det)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx,
		`INSERT INTO venta_local (id, fecha, total, envio, detalle, creada_en) VALUES (?, ?, ?, ?, ?, ?)`,
		v.ID, fechaTexto(v.Fecha), v.Total.Centavos(), string(bEnvio), string(bDet), fechaTexto(v.CreatedAt))
	return err
}

// armarDetalle prepara lo que muestra la pantalla al cobrar. El costo y la ganancia los calcula
// el servidor al recibir la venta (CostoCompleto=false mientras tanto).
func (r RepositorioVentas) armarDetalle(ctx context.Context, conn db.Ejecutor, v domain.Venta) (ventas.VentaDetalle, error) {
	d := ventas.VentaDetalle{VentaResumen: ventas.VentaResumen{ID: v.ID, Fecha: v.Fecha, Total: v.Total,
		Estado: v.Estado, Origen: domain.OrigenDesktop, Productos: len(v.Items), Metodos: []domain.MetodoPago{}},
		Ganancia: v.Total, Items: []ventas.ItemDTO{}, Pagos: []ventas.PagoDTO{}}
	for _, it := range v.Items {
		dto := ventas.ItemDTO{ProductoID: it.ProductoID, Cantidad: it.Cantidad, PrecioUnitario: it.PrecioUnitario, Subtotal: it.Subtotal()}
		_ = conn.QueryRowContext(ctx, `SELECT sku, nombre, tipo FROM copia_producto WHERE id = ?`, it.ProductoID).
			Scan(&dto.SKU, &dto.Producto, &dto.Tipo)
		d.Items = append(d.Items, dto)
		d.Unidades += it.Cantidad
	}
	vistos := map[domain.MetodoPago]bool{}
	for _, p := range v.Pagos {
		d.Pagos = append(d.Pagos, ventas.PagoDTO{Metodo: p.Metodo, Monto: p.Monto, MontoRecibido: p.MontoRecibido, Vuelto: p.Vuelto})
		if p.Vuelto != nil {
			d.Vuelto = d.Vuelto.Sumar(*p.Vuelto)
		}
		if !vistos[p.Metodo] {
			vistos[p.Metodo] = true
			d.Metodos = append(d.Metodos, p.Metodo)
		}
	}
	return d, nil
}

func (r RepositorioVentas) Existe(ctx context.Context, id string) (bool, error) {
	var n int
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx, `SELECT COUNT(*) FROM venta_local WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}

func (r RepositorioVentas) Detalle(ctx context.Context, id string) (ventas.VentaDetalle, error) {
	var crudo string
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx, `SELECT detalle FROM venta_local WHERE id = ?`, id).Scan(&crudo)
	if errors.Is(err, sql.ErrNoRows) {
		return ventas.VentaDetalle{}, domain.ErrVentaNoEncontrada
	}
	if err != nil {
		return ventas.VentaDetalle{}, err
	}
	var d ventas.VentaDetalle
	return d, json.Unmarshal([]byte(crudo), &d)
}

func (r RepositorioVentas) Obtener(ctx context.Context, id string) (domain.Venta, error) {
	d, err := r.Detalle(ctx, id)
	if err != nil {
		return domain.Venta{}, err
	}
	return domain.Venta{ID: d.ID, Fecha: d.Fecha, Total: d.Total, Estado: d.Estado, Origen: d.Origen}, nil
}

// Anular se hace contra el servidor (ahí está el stock real y los lotes).
func (RepositorioVentas) MarcarAnulada(context.Context, domain.Venta) error {
	return ErrNecesitaInternet
}

// Listar devuelve las ventas cobradas en esta PC (la más reciente primero).
func (r RepositorioVentas) Listar(ctx context.Context, _ ventas.Filtro, p paginacion.Solicitud) ([]ventas.VentaResumen, int, error) {
	conn := db.Conn(ctx, r.DB)
	var total int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM venta_local`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn.QueryContext(ctx, `SELECT detalle FROM venta_local ORDER BY fecha DESC, id DESC LIMIT ? OFFSET ?`, p.Limite(), p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []ventas.VentaResumen{}
	for rows.Next() {
		var crudo string
		if err := rows.Scan(&crudo); err != nil {
			return nil, 0, err
		}
		var d ventas.VentaDetalle
		if err := json.Unmarshal([]byte(crudo), &d); err != nil {
			return nil, 0, err
		}
		out = append(out, d.VentaResumen)
	}
	return out, total, rows.Err()
}
