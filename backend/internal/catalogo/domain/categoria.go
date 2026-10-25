package domain

import (
	"slices"
	"time"
	"unicode/utf8"

	"magui/internal/kernel/errores"
)

// Categoria agrupa productos. Puede tener una categoría padre (ej. PERFUMERÍA > Femenina).
type Categoria struct {
	ID        string
	Nombre    string
	PadreID   *string
	Version   int
	UpdatedAt time.Time
}

// NuevaCategoria valida y crea una categoría.
func NuevaCategoria(id, nombre string, padreID *string, ahora time.Time) (Categoria, error) {
	c := Categoria{ID: id}
	if err := c.Modificar(nombre, padreID, ahora); err != nil {
		return Categoria{}, err
	}
	c.Version = 1
	return c, nil
}

// Modificar cambia nombre y padre (sin validar ciclos: eso requiere el árbol, ver Arbol).
func (c *Categoria) Modificar(nombre string, padreID *string, ahora time.Time) error {
	nombre = LimpiarTexto(nombre)
	var v errores.Validacion
	v.Si(nombre == "", "nombre", "El nombre es obligatorio")
	v.Si(utf8.RuneCountInString(nombre) > 100, "nombre", "Máximo 100 caracteres")
	if err := v.Err(); err != nil {
		return err
	}
	if padreID != nil && *padreID == "" {
		padreID = nil
	}
	if padreID != nil && *padreID == c.ID {
		return ErrCategoriaCiclo
	}
	c.Nombre, c.PadreID, c.UpdatedAt = nombre, padreID, ahora
	c.Version++
	return nil
}

// Arbol permite recorrer la jerarquía de categorías.
type Arbol struct {
	porID  map[string]Categoria
	hijos  map[string][]string
	raices []string
}

// NuevoArbol arma el árbol a partir de todas las categorías.
func NuevoArbol(cats []Categoria) Arbol {
	a := Arbol{porID: map[string]Categoria{}, hijos: map[string][]string{}}
	for _, c := range cats {
		a.porID[c.ID] = c
	}
	for _, c := range cats {
		if c.PadreID == nil {
			a.raices = append(a.raices, c.ID)
		} else {
			a.hijos[*c.PadreID] = append(a.hijos[*c.PadreID], c.ID)
		}
	}
	return a
}

// Existe indica si la categoría está en el árbol.
func (a Arbol) Existe(id string) bool { _, ok := a.porID[id]; return ok }

// Obtener devuelve la categoría.
func (a Arbol) Obtener(id string) (Categoria, bool) { c, ok := a.porID[id]; return c, ok }

// Hijos devuelve los IDs de las subcategorías directas.
func (a Arbol) Hijos(id string) []string { return a.hijos[id] }

// Raices devuelve los IDs de las categorías sin padre.
func (a Arbol) Raices() []string { return a.raices }

// ConDescendientes devuelve id y todas sus subcategorías (a cualquier profundidad).
func (a Arbol) ConDescendientes(id string) []string {
	out := []string{id}
	for i := 0; i < len(out); i++ {
		out = append(out, a.hijos[out[i]]...)
	}
	return out
}

// PuedeMoverA indica si id puede tener como padre a nuevoPadre sin formar un ciclo.
func (a Arbol) PuedeMoverA(id string, nuevoPadre *string) bool {
	if nuevoPadre == nil {
		return true
	}
	return !slices.Contains(a.ConDescendientes(id), *nuevoPadre)
}

// Ruta devuelve los nombres desde la raíz: ["PERFUMERÍA", "Perfumería femenina"].
func (a Arbol) Ruta(id string) []string {
	var ruta []string
	visto := map[string]bool{}
	for c, ok := a.porID[id]; ok && !visto[c.ID]; {
		visto[c.ID] = true
		ruta = append([]string{c.Nombre}, ruta...)
		if c.PadreID == nil {
			break
		}
		c, ok = a.porID[*c.PadreID]
	}
	return ruta
}
