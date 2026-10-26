package http

import (
	"magui/internal/kernel/dinero"
	sincro "magui/internal/sync/application"
)

// productoLocal tiene la forma del producto que devuelve el servidor (ProductoDetalle),
// con lo que la copia no guarda vacío: así la pantalla no distingue de dónde vino.
type productoLocal struct {
	sincro.ProductoCopia
	EmpresaID       *string        `json:"empresaId"`
	Costo           *dinero.Dinero `json:"costo"`
	StockMinimo     int            `json:"stockMinimo"`
	Descripcion     *string        `json:"descripcion"`
	MostrarEnTienda bool           `json:"mostrarEnTienda"`
	Categorias      []any          `json:"categorias"`
	Items           []any          `json:"items"`
	// DesdeCopia avisa que el dato viene de la copia local (sin conexión).
	DesdeCopia bool `json:"desdeCopia"`
}

func local(p sincro.ProductoCopia) productoLocal {
	return productoLocal{ProductoCopia: p, Categorias: []any{}, Items: []any{}, DesdeCopia: true}
}
