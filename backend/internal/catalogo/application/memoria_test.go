package application

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"magui/internal/catalogo/domain"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/kernel/tx"
)

// Repositorios en memoria para probar los casos de uso sin base de datos.

type memEmpresas struct{ m map[string]domain.Empresa }

func (r *memEmpresas) Crear(_ context.Context, e domain.Empresa) error { r.m[e.ID] = e; return nil }
func (r *memEmpresas) Actualizar(_ context.Context, e domain.Empresa, v int) error {
	if r.m[e.ID].Version != v {
		return domain.ErrVersionDesactualizada
	}
	r.m[e.ID] = e
	return nil
}
func (r *memEmpresas) ObtenerPorID(_ context.Context, id string) (domain.Empresa, error) {
	e, ok := r.m[id]
	if !ok {
		return e, domain.ErrEmpresaNoEncontrada
	}
	return e, nil
}
func (r *memEmpresas) ExisteNombre(_ context.Context, n, excluir string) (bool, error) {
	for _, e := range r.m {
		if e.ID != excluir && strings.EqualFold(e.Nombre, n) {
			return true, nil
		}
	}
	return false, nil
}
func (r *memEmpresas) Listar(_ context.Context, f FiltroEmpresas, p paginacion.Solicitud) ([]domain.Empresa, int, error) {
	var out []domain.Empresa
	for _, e := range r.m {
		if (!f.SoloActivas || e.Activo) && strings.Contains(strings.ToLower(e.Nombre), strings.ToLower(f.Texto)) {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nombre < out[j].Nombre })
	return pagina(out, p), len(out), nil
}

type memCategorias struct {
	m     map[string]domain.Categoria
	prods *memProductos
}

func (r *memCategorias) Crear(_ context.Context, c domain.Categoria) error { r.m[c.ID] = c; return nil }
func (r *memCategorias) Actualizar(_ context.Context, c domain.Categoria, v int) error {
	if r.m[c.ID].Version != v {
		return domain.ErrVersionDesactualizada
	}
	r.m[c.ID] = c
	return nil
}
func (r *memCategorias) Eliminar(_ context.Context, id string) error { delete(r.m, id); return nil }
func (r *memCategorias) ListarTodas(context.Context) ([]domain.Categoria, error) {
	var out []domain.Categoria
	for _, c := range r.m {
		out = append(out, c)
	}
	return out, nil
}
func (r *memCategorias) ProductosPorCategoria(context.Context) (map[string]int, error) {
	out := map[string]int{}
	for _, p := range r.prods.m {
		for _, c := range p.CategoriaIDs {
			out[c]++
		}
	}
	return out, nil
}

type memProductos struct {
	m         map[string]domain.Producto
	historial []domain.CambioPrecio
}

func (r *memProductos) Crear(_ context.Context, p domain.Producto) error { r.m[p.ID] = p; return nil }
func (r *memProductos) Actualizar(_ context.Context, p domain.Producto, v int) error {
	if r.m[p.ID].Version != v {
		return domain.ErrVersionDesactualizada
	}
	r.m[p.ID] = p
	return nil
}
func (r *memProductos) ObtenerPorID(_ context.Context, id string) (domain.Producto, error) {
	p, ok := r.m[id]
	if !ok {
		return p, domain.ErrProductoNoEncontrado
	}
	return p, nil
}
func (r *memProductos) ObtenerVarios(_ context.Context, ids []string) ([]domain.Producto, error) {
	var out []domain.Producto
	for _, id := range ids {
		if p, ok := r.m[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r *memProductos) CodigoEnUso(_ context.Context, codigo, excluir string) (bool, error) {
	for _, p := range r.m {
		if p.ID != excluir && (p.SKU == codigo || (p.CodigoBarras != nil && *p.CodigoBarras == codigo)) {
			return true, nil
		}
	}
	return false, nil
}
func (r *memProductos) IDPorCodigo(_ context.Context, codigo string) (string, error) {
	for _, p := range r.m {
		if p.SKU == codigo || (p.CodigoBarras != nil && *p.CodigoBarras == codigo) {
			return p.ID, nil
		}
	}
	return "", domain.ErrProductoNoEncontrado
}
func (r *memProductos) Listar(_ context.Context, f FiltroProductos, p paginacion.Solicitud) ([]ProductoResumen, int, error) {
	var out []ProductoResumen
	for _, pr := range r.m {
		if f.Estado == EstadoActivos && !pr.Activo || f.Estado == EstadoInactivos && pr.Activo {
			continue
		}
		if f.Texto != "" && !strings.Contains(strings.ToLower(pr.Nombre+" "+pr.SKU), strings.ToLower(f.Texto)) {
			continue
		}
		if len(f.CategoriaIDs) > 0 && !slices.ContainsFunc(pr.CategoriaIDs, func(c string) bool { return slices.Contains(f.CategoriaIDs, c) }) {
			continue
		}
		out = append(out, resumen(pr))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nombre < out[j].Nombre })
	return pagina(out, p), len(out), nil
}
func (r *memProductos) ObtenerDetalle(ctx context.Context, id string) (ProductoDetalle, error) {
	p, err := r.ObtenerPorID(ctx, id)
	if err != nil {
		return ProductoDetalle{}, err
	}
	d := ProductoDetalle{ProductoResumen: resumen(p), Descripcion: p.Descripcion, MostrarEnTienda: p.MostrarEnTienda}
	for _, c := range p.CategoriaIDs {
		d.Categorias = append(d.Categorias, CategoriaRef{ID: c})
	}
	for _, it := range p.Items {
		comp := r.m[it.ProductoID]
		d.Items = append(d.Items, ComboItemDTO{ProductoID: comp.ID, SKU: comp.SKU, Nombre: comp.Nombre, Cantidad: it.Cantidad})
	}
	return d, nil
}
func (r *memProductos) Exportar(context.Context) ([]ProductoResumen, []ComponenteCombo, error) {
	var ps []ProductoResumen
	var cs []ComponenteCombo
	for _, p := range r.m {
		ps = append(ps, resumen(p))
		for _, it := range p.Items {
			cs = append(cs, ComponenteCombo{ComboID: p.ID, ProductoID: it.ProductoID, Cantidad: it.Cantidad})
		}
	}
	return ps, cs, nil
}
func (r *memProductos) ActivosEnCategorias(_ context.Context, ids []string) ([]domain.Producto, error) {
	var out []domain.Producto
	for _, p := range r.m {
		if p.Activo && slices.ContainsFunc(p.CategoriaIDs, func(c string) bool { return slices.Contains(ids, c) }) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nombre < out[j].Nombre })
	return out, nil
}
func (r *memProductos) ActualizarPrecios(_ context.Context, ps []domain.Producto) error {
	for _, p := range ps {
		r.m[p.ID] = p
	}
	return nil
}
func (r *memProductos) RegistrarCambiosPrecio(_ context.Context, c []domain.CambioPrecio) error {
	r.historial = append(r.historial, c...)
	return nil
}
func (r *memProductos) HistorialPrecios(_ context.Context, id string, p paginacion.Solicitud) ([]CambioPrecioDTO, int, error) {
	var out []CambioPrecioDTO
	for _, c := range r.historial {
		if c.ProductoID == id {
			out = append(out, CambioPrecioDTO{ID: c.ID, PrecioAnterior: c.PrecioAnterior, PrecioNuevo: c.PrecioNuevo, Origen: c.Origen})
		}
	}
	return pagina(out, p), len(out), nil
}

func resumen(p domain.Producto) ProductoResumen {
	return ProductoResumen{ID: p.ID, SKU: p.SKU, CodigoBarras: p.CodigoBarras, Nombre: p.Nombre, Tipo: p.Tipo,
		EmpresaID: p.EmpresaID, PrecioVenta: p.PrecioVenta, Costo: p.Costo, StockMinimo: p.StockMinimo,
		Activo: p.Activo, Version: p.Version}
}

func pagina[T any](items []T, p paginacion.Solicitud) []T {
	if p.Offset() >= len(items) {
		return nil
	}
	return items[p.Offset():min(p.Offset()+p.Limite(), len(items))]
}

// idsSecuenciales genera "id-1", "id-2", ... para tests legibles.
type idsSecuenciales struct{ n int }

func (g *idsSecuenciales) Nuevo() string { g.n++; return fmt.Sprintf("id-%d", g.n) }

type entorno struct {
	svc        *Servicio
	empresas   *memEmpresas
	categorias *memCategorias
	productos  *memProductos
}

func nuevoEntorno() entorno {
	prods := &memProductos{m: map[string]domain.Producto{}}
	e := entorno{
		empresas:   &memEmpresas{m: map[string]domain.Empresa{}},
		categorias: &memCategorias{m: map[string]domain.Categoria{}, prods: prods},
		productos:  prods,
	}
	e.svc = NuevoServicio(Dependencias{
		Empresas: e.empresas, Categorias: e.categorias, Productos: e.productos,
		Tx: tx.SinTransaccion{}, IDs: &idsSecuenciales{},
		Reloj: reloj.Fijo{T: time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)},
	})
	return e
}
