// Package web sirve el frontend compilado (Angular) desde la misma API.
package web

import (
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
)

// Archivos con hash en el nombre (main-3AW3IKRT.js): nunca cambian, se pueden guardar un año.
var conHash = regexp.MustCompile(`-[A-Z0-9]{8}\.(js|css)$`)

// Archivos que siempre deben pedirse de nuevo para detectar versiones nuevas.
var sinCache = map[string]bool{
	"index.html": true, "ngsw.json": true, "ngsw-worker.js": true, "safety-worker.js": true,
	"worker-basic.min.js": true, "manifest.webmanifest": true,
}

// SPA sirve los archivos de dir y, para las rutas de la app (sin extensión, ej. /vender),
// devuelve index.html para que Angular resuelva la pantalla.
func SPA(dir fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
			return
		}
		nombre := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if nombre == "" {
			nombre = "index.html"
		}
		if info, err := fs.Stat(dir, nombre); err == nil && !info.IsDir() {
			servir(w, r, dir, nombre)
			return
		}
		if path.Ext(nombre) != "" {
			http.NotFound(w, r) // un archivo que no existe (no es una ruta de la app)
			return
		}
		servir(w, r, dir, "index.html")
	})
}

func servir(w http.ResponseWriter, r *http.Request, dir fs.FS, nombre string) {
	switch {
	case sinCache[nombre]:
		w.Header().Set("Cache-Control", "no-cache")
	case conHash.MatchString(nombre):
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	default:
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	if nombre == "manifest.webmanifest" {
		w.Header().Set("Content-Type", "application/manifest+json")
	}
	http.ServeFileFS(w, r, dir, nombre)
}

// Seguridad agrega encabezados de seguridad a todas las respuestas.
func Seguridad(next http.Handler) http.Handler {
	const csp = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; " +
		"font-src 'self' data:; connect-src 'self'; worker-src 'self'; manifest-src 'self'; " +
		"frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Content-Security-Policy", csp)
		next.ServeHTTP(w, r)
	})
}
