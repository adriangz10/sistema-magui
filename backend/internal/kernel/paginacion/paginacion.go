// Package paginacion estandariza las listas paginadas (máximo 10 filas por página).
package paginacion

// TamanoMaximo es el máximo de filas por página que pidió la clienta.
const TamanoMaximo = 10

// Solicitud es la página pedida, ya normalizada.
type Solicitud struct {
	Pagina int
	Tamano int
}

// Nueva normaliza: página mínima 1; tamaño entre 1 y TamanoMaximo (por defecto TamanoMaximo).
func Nueva(pagina, tamano int) Solicitud {
	if pagina < 1 {
		pagina = 1
	}
	if tamano < 1 || tamano > TamanoMaximo {
		tamano = TamanoMaximo
	}
	return Solicitud{Pagina: pagina, Tamano: tamano}
}

// Offset es la cantidad de filas a saltear (para SQL LIMIT/OFFSET).
func (s Solicitud) Offset() int { return (s.Pagina - 1) * s.Tamano }

// Limite es la cantidad de filas a traer.
func (s Solicitud) Limite() int { return s.Tamano }

// Pagina es una página de resultados, con el formato JSON acordado en las convenciones.
type Pagina[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	Size       int `json:"size"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// NuevaPagina arma la respuesta. Items nunca es nil (en JSON sale [] y no null).
func NuevaPagina[T any](items []T, s Solicitud, total int) Pagina[T] {
	if items == nil {
		items = []T{}
	}
	totalPaginas := 0
	if total > 0 {
		totalPaginas = (total + s.Tamano - 1) / s.Tamano
	}
	return Pagina[T]{Items: items, Page: s.Pagina, Size: s.Tamano, Total: total, TotalPages: totalPaginas}
}
