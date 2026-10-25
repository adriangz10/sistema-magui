// Package domain contiene las entidades y reglas del catálogo: empresas, categorías,
// productos (simples y combos) y ajustes de precio.
package domain

import "magui/internal/kernel/errores"

var (
	ErrEmpresaNoEncontrada   = errores.NoEncontrado("empresa_no_encontrada", "La empresa no existe")
	ErrEmpresaDuplicada      = errores.Conflicto("empresa_duplicada", "Ya existe una empresa con ese nombre")
	ErrCategoriaNoEncontrada = errores.NoEncontrado("categoria_no_encontrada", "La categoría no existe")
	ErrCategoriaDuplicada    = errores.Conflicto("categoria_duplicada", "Ya existe una categoría con ese nombre en el mismo nivel")
	ErrCategoriaCiclo        = errores.Invalido("categoria_ciclo", "Una categoría no puede quedar dentro de sí misma o de sus subcategorías")
	ErrCategoriaConHijos     = errores.Conflicto("categoria_con_hijos", "No se puede borrar: tiene subcategorías")
	ErrCategoriaConProductos = errores.Conflicto("categoria_con_productos", "No se puede borrar: tiene productos asignados")
	ErrProductoNoEncontrado  = errores.NoEncontrado("producto_no_encontrado", "El producto no existe")
	ErrSKUDuplicado          = errores.Conflicto("sku_duplicado", "Ya existe un producto con ese SKU")
	ErrCodigoBarrasDuplicado = errores.Conflicto("codigo_barras_duplicado", "Ya existe un producto con ese código de barras")
	ErrVersionDesactualizada = errores.Conflicto("version_desactualizada",
		"Otro dispositivo modificó estos datos. Recargá la pantalla e intentá de nuevo.")
)

// ErrComboInvalido describe por qué un combo no se puede guardar.
func ErrComboInvalido(motivo string) error {
	return errores.Invalido("combo_invalido", motivo)
}
