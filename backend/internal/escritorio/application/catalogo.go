package application

import (
	"context"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"magui/internal/kernel/errores"
	"magui/internal/kernel/paginacion"
	sincro "magui/internal/sync/application"
)

// ConsultaLocal es la búsqueda de productos que hace la pantalla Vender sin conexión.
type ConsultaLocal struct {
	Texto  string // en nombre, SKU o código de barras; sin distinguir mayúsculas ni tildes
	Estado string // activos (por defecto), inactivos o todos
	Tipo   string // SIMPLE, COMBO o vacío
}

// ErrProductoNoEncontrado tiene el mismo código que en el servidor.
var ErrProductoNoEncontrado = errores.NoEncontrado("producto_no_encontrado", "El producto no existe")

// CatalogoLocal busca en la copia del catálogo.
type CatalogoLocal struct{ cola Cola }

func NuevoCatalogoLocal(c Cola) *CatalogoLocal { return &CatalogoLocal{cola: c} }

// Buscar filtra la copia, ordena por nombre y pagina (igual que el servidor).
func (c *CatalogoLocal) Buscar(ctx context.Context, q ConsultaLocal, p paginacion.Solicitud) (paginacion.Pagina[sincro.ProductoCopia], error) {
	todos, err := c.cola.Productos(ctx)
	if err != nil {
		return paginacion.Pagina[sincro.ProductoCopia]{}, err
	}
	texto := normalizar(q.Texto)
	var hallados []sincro.ProductoCopia
	for _, pr := range todos {
		switch {
		case q.Estado == "inactivos" && pr.Activo, q.Estado != "inactivos" && q.Estado != "todos" && !pr.Activo:
			continue
		case q.Tipo != "" && !strings.EqualFold(q.Tipo, pr.Tipo):
			continue
		case texto != "" && !coincide(pr, texto):
			continue
		}
		hallados = append(hallados, pr)
	}
	desde := min(p.Offset(), len(hallados))
	hasta := min(desde+p.Limite(), len(hallados))
	return paginacion.NuevaPagina(append([]sincro.ProductoCopia{}, hallados[desde:hasta]...), p, len(hallados)), nil
}

// PorCodigo busca por SKU o código de barras exactos (lector de códigos).
func (c *CatalogoLocal) PorCodigo(ctx context.Context, codigo string) (sincro.ProductoCopia, error) {
	codigo = strings.TrimSpace(codigo)
	todos, err := c.cola.Productos(ctx)
	if err != nil {
		return sincro.ProductoCopia{}, err
	}
	for _, p := range todos {
		if strings.EqualFold(p.SKU, codigo) || (p.CodigoBarras != nil && *p.CodigoBarras == codigo) {
			return p, nil
		}
	}
	return sincro.ProductoCopia{}, ErrProductoNoEncontrado
}

func coincide(p sincro.ProductoCopia, texto string) bool {
	if strings.Contains(normalizar(p.Nombre), texto) || strings.Contains(normalizar(p.SKU), texto) {
		return true
	}
	return p.CodigoBarras != nil && strings.Contains(normalizar(*p.CodigoBarras), texto)
}

// normalizar pasa a minúsculas y quita tildes ("Clásico" -> "clasico").
func normalizar(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, strings.ToLower(strings.TrimSpace(s)))
	if err != nil {
		return strings.ToLower(s)
	}
	return out
}
