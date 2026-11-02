package domain

import (
	"magui/internal/kernel/errores"
	"magui/internal/kernel/imagen"
)

// TamanoMaximoImagen de la foto de un producto. La app la achica antes de subirla (~100 KB).
const TamanoMaximoImagen = 1 << 20

// ValidarImagen revisa la foto por su contenido real (PNG, JPG o WEBP; sin SVG).
func ValidarImagen(datos []byte) (tipo string, err error) {
	if len(datos) == 0 {
		return "", ErrImagenVacia
	}
	if len(datos) > TamanoMaximoImagen {
		return "", ErrImagenGrande
	}
	if tipo = imagen.Detectar(datos); tipo == "" {
		return "", ErrImagenTipo
	}
	return tipo, nil
}

var (
	ErrSinImagen    = errores.NoEncontrado("sin_imagen", "El producto no tiene foto")
	ErrImagenVacia  = errores.Invalido("imagen_vacia", "Elegí una imagen")
	ErrImagenGrande = errores.Invalido("imagen_muy_grande", "La imagen es muy pesada: máximo 1 MB")
	ErrImagenTipo   = errores.Invalido("imagen_tipo", "Usá una imagen PNG, JPG o WEBP")
)
