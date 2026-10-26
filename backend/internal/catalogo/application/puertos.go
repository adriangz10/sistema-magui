package application

import (
	"context"

	"magui/internal/catalogo/domain"
	"magui/internal/kernel/paginacion"
)

// RepositorioEmpresas persiste empresas.
type RepositorioEmpresas interface {
	Crear(ctx context.Context, e domain.Empresa) error
	// Actualizar falla con domain.ErrVersionDesactualizada si la versión guardada no es versionAnterior.
	Actualizar(ctx context.Context, e domain.Empresa, versionAnterior int) error
	ObtenerPorID(ctx context.Context, id string) (domain.Empresa, error)
	ExisteNombre(ctx context.Context, nombre, excluirID string) (bool, error)
	Listar(ctx context.Context, f FiltroEmpresas, p paginacion.Solicitud) ([]domain.Empresa, int, error)
}

// RepositorioCategorias persiste categorías.
type RepositorioCategorias interface {
	Crear(ctx context.Context, c domain.Categoria) error
	Actualizar(ctx context.Context, c domain.Categoria, versionAnterior int) error
	Eliminar(ctx context.Context, id string) error
	ListarTodas(ctx context.Context) ([]domain.Categoria, error)
	// ProductosPorCategoria devuelve cuántos productos tiene asignados cada categoría.
	ProductosPorCategoria(ctx context.Context) (map[string]int, error)
}

// RepositorioProductos persiste productos (con sus categorías y componentes de combo).
type RepositorioProductos interface {
	Crear(ctx context.Context, p domain.Producto) error
	Actualizar(ctx context.Context, p domain.Producto, versionAnterior int) error
	ObtenerPorID(ctx context.Context, id string) (domain.Producto, error)
	// ObtenerVarios devuelve los productos encontrados (los inexistentes se omiten).
	ObtenerVarios(ctx context.Context, ids []string) ([]domain.Producto, error)
	// CodigoEnUso indica si el código ya es SKU o código de barras de otro producto.
	CodigoEnUso(ctx context.Context, codigo, excluirID string) (bool, error)
	// IDPorCodigo busca un producto por SKU o código de barras exactos (lector de códigos).
	IDPorCodigo(ctx context.Context, codigo string) (string, error)

	Listar(ctx context.Context, f FiltroProductos, p paginacion.Solicitud) ([]ProductoResumen, int, error)
	// ObtenerDetalle devuelve la ficha; Categorias trae ID y Nombre (la ruta la completa el servicio).
	ObtenerDetalle(ctx context.Context, id string) (ProductoDetalle, error)

	// ActivosEnCategorias devuelve los productos activos asignados a alguna de las categorías.
	ActivosEnCategorias(ctx context.Context, categoriaIDs []string) ([]domain.Producto, error)
	// ActualizarPrecios guarda precio, versión y fecha de modificación de cada producto.
	ActualizarPrecios(ctx context.Context, productos []domain.Producto) error

	// Exportar devuelve todos los productos (con stock) y los componentes de todos los combos.
	Exportar(ctx context.Context) ([]ProductoResumen, []ComponenteCombo, error)

	RegistrarCambiosPrecio(ctx context.Context, cambios []domain.CambioPrecio) error
	HistorialPrecios(ctx context.Context, productoID string, p paginacion.Solicitud) ([]CambioPrecioDTO, int, error)
}
