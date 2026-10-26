// Package escritorio expone el núcleo de la app de escritorio para el proyecto Wails (desktop/),
// que es otro módulo de Go y no puede importar los paquetes internal/ directamente.
package escritorio

import (
	"io/fs"
	"net/http"

	interno "magui/internal/escritorio"
	"magui/internal/platform/web"
)

// Config de la app de escritorio (dirección del servidor, archivo local, app compilada, log).
type Config = interno.Config

// App es el escritorio armado: Handler para la ventana y Ejecutar para la sincronización.
type App = interno.App

// Abrir crea o abre la base local, aplica sus migraciones y arma la app.
var Abrir = interno.Abrir

// SPA sirve la app compilada (index.html para las rutas de la app).
func SPA(archivos fs.FS) http.Handler { return web.SPA(archivos) }
