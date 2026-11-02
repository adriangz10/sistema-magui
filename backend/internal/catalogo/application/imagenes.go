package application

import (
	"context"

	"magui/internal/catalogo/domain"
)

// RepositorioImagenes guarda la foto opcional de cada producto (US-004).
type RepositorioImagenes interface {
	// Guardar reemplaza la foto (datos nil la quita) y devuelve la nueva versión (0 = sin foto).
	// Si el producto no existe devuelve domain.ErrProductoNoEncontrado.
	Guardar(ctx context.Context, productoID string, datos []byte, tipo string) (int, error)
	Obtener(ctx context.Context, productoID string) (datos []byte, tipo string, err error)
}

// ImagenProducto indica la versión de la foto (0 = sin foto) para armar la URL.
type ImagenProducto struct {
	ImagenVersion int `json:"imagenVersion"`
}

// CambiarImagen valida y guarda la foto de un producto. datos nil la quita.
func (s *Servicio) CambiarImagen(ctx context.Context, productoID string, datos []byte) (ImagenProducto, error) {
	tipo := ""
	if datos != nil {
		var err error
		if tipo, err = domain.ValidarImagen(datos); err != nil {
			return ImagenProducto{}, err
		}
	}
	v, err := s.imagenes.Guardar(ctx, productoID, datos, tipo)
	return ImagenProducto{ImagenVersion: v}, err
}

// Imagen devuelve la foto de un producto.
func (s *Servicio) Imagen(ctx context.Context, productoID string) ([]byte, string, error) {
	return s.imagenes.Obtener(ctx, productoID)
}
