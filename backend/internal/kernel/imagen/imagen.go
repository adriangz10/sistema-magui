// Package imagen valida imágenes subidas por la usuaria (logo del negocio, fotos de productos).
package imagen

import "net/http"

// Tipos aceptados. SVG no: puede llevar scripts.
var Tipos = map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true}

// Detectar devuelve el tipo real según el contenido (no lo que dice el navegador),
// o "" si no es una imagen aceptada.
func Detectar(datos []byte) string {
	if len(datos) == 0 {
		return ""
	}
	tipo := http.DetectContentType(datos)
	if !Tipos[tipo] {
		return ""
	}
	return tipo
}
