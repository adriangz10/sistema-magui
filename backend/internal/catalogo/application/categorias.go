package application

import (
	"context"
	"sort"
	"strings"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"magui/internal/catalogo/domain"
)

// ArbolCategorias devuelve todas las categorías anidadas, ordenadas alfabéticamente.
func (s *Servicio) ArbolCategorias(ctx context.Context) ([]CategoriaNodo, error) {
	cats, err := s.categorias.ListarTodas(ctx)
	if err != nil {
		return nil, err
	}
	conteo, err := s.categorias.ProductosPorCategoria(ctx)
	if err != nil {
		return nil, err
	}
	arbol := domain.NuevoArbol(cats)
	var armar func(ids []string) []CategoriaNodo
	armar = func(ids []string) []CategoriaNodo {
		nodos := make([]CategoriaNodo, 0, len(ids))
		for _, id := range ids {
			c, _ := arbol.Obtener(id)
			nodos = append(nodos, CategoriaNodo{
				CategoriaDTO: categoriaDTO(c), Productos: conteo[id], Hijos: armar(arbol.Hijos(id)),
			})
		}
		ordenarNodos(nodos)
		return nodos
	}
	return armar(arbol.Raices()), nil
}

var colador = collate.New(language.Spanish, collate.IgnoreCase, collate.IgnoreDiacritics)

func ordenarNodos(n []CategoriaNodo) {
	sort.SliceStable(n, func(i, j int) bool { return colador.CompareString(n[i].Nombre, n[j].Nombre) < 0 })
}

// CrearCategoria da de alta una categoría (padreID nil = categoría principal).
func (s *Servicio) CrearCategoria(ctx context.Context, nombre string, padreID *string) (CategoriaDTO, error) {
	c, err := domain.NuevaCategoria(s.ids.Nuevo(), nombre, padreID, s.reloj.Ahora())
	if err != nil {
		return CategoriaDTO{}, err
	}
	arbol, err := s.arbol(ctx)
	if err != nil {
		return CategoriaDTO{}, err
	}
	if err := validarUbicacion(arbol, c); err != nil {
		return CategoriaDTO{}, err
	}
	if err := s.categorias.Crear(ctx, c); err != nil {
		return CategoriaDTO{}, err
	}
	return categoriaDTO(c), nil
}

// ActualizarCategoria renombra o mueve una categoría.
func (s *Servicio) ActualizarCategoria(ctx context.Context, id, nombre string, padreID *string, version int) (CategoriaDTO, error) {
	arbol, err := s.arbol(ctx)
	if err != nil {
		return CategoriaDTO{}, err
	}
	c, ok := arbol.Obtener(id)
	if !ok {
		return CategoriaDTO{}, domain.ErrCategoriaNoEncontrada
	}
	if c.Version != version {
		return CategoriaDTO{}, domain.ErrVersionDesactualizada
	}
	if err := c.Modificar(nombre, padreID, s.reloj.Ahora()); err != nil {
		return CategoriaDTO{}, err
	}
	if !arbol.PuedeMoverA(c.ID, c.PadreID) {
		return CategoriaDTO{}, domain.ErrCategoriaCiclo
	}
	if err := validarUbicacion(arbol, c); err != nil {
		return CategoriaDTO{}, err
	}
	if err := s.categorias.Actualizar(ctx, c, version); err != nil {
		return CategoriaDTO{}, err
	}
	return categoriaDTO(c), nil
}

// EliminarCategoria borra una categoría vacía (sin subcategorías ni productos).
func (s *Servicio) EliminarCategoria(ctx context.Context, id string) error {
	arbol, err := s.arbol(ctx)
	if err != nil {
		return err
	}
	if !arbol.Existe(id) {
		return domain.ErrCategoriaNoEncontrada
	}
	if len(arbol.Hijos(id)) > 0 {
		return domain.ErrCategoriaConHijos
	}
	conteo, err := s.categorias.ProductosPorCategoria(ctx)
	if err != nil {
		return err
	}
	if conteo[id] > 0 {
		return domain.ErrCategoriaConProductos
	}
	return s.categorias.Eliminar(ctx, id)
}

func (s *Servicio) arbol(ctx context.Context) (domain.Arbol, error) {
	cats, err := s.categorias.ListarTodas(ctx)
	if err != nil {
		return domain.Arbol{}, err
	}
	return domain.NuevoArbol(cats), nil
}

// validarUbicacion: el padre existe y no hay otra categoría con el mismo nombre en ese nivel.
func validarUbicacion(arbol domain.Arbol, c domain.Categoria) error {
	var hermanos []string
	if c.PadreID != nil {
		if !arbol.Existe(*c.PadreID) {
			return domain.ErrCategoriaNoEncontrada
		}
		hermanos = arbol.Hijos(*c.PadreID)
	} else {
		hermanos = arbol.Raices()
	}
	for _, h := range hermanos {
		otra, _ := arbol.Obtener(h)
		if otra.ID != c.ID && colador.CompareString(otra.Nombre, c.Nombre) == 0 {
			return domain.ErrCategoriaDuplicada
		}
	}
	return nil
}

// rutaTexto devuelve "PERFUMERÍA > Femenina".
func rutaTexto(arbol domain.Arbol, id string) string {
	return strings.Join(arbol.Ruta(id), " > ")
}
