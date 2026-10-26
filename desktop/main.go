// App de escritorio de Magui (Wails). Muestra la misma app que la web, pero el "servidor" que ve
// la pantalla es el núcleo local (magui/escritorio): cobra aunque no haya internet y sube las
// ventas solo cuando vuelve la conexión. No abre ningún puerto: Wails le pasa los pedidos en memoria.
//
// Compilar (desde desktop/):  wails build -clean -skipbindings
package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"magui/escritorio"
)

// La compilación de escritorio del frontend (npm run build:escritorio) queda en web/browser.
//
//go:embed all:web/browser
var archivos embed.FS

// servidorPorDefecto se fija al compilar para la clienta:
//
//	wails build -ldflags "-X main.servidorPorDefecto=https://magui.ejemplo.com"
var servidorPorDefecto = "http://localhost:8080"

// version se fija al compilar (instalador.ps1): -X main.version=1.0.0
var version = "dev"

// envolverHandler lo cambia pruebas.go (solo con -tags pruebas) para anotar los pedidos.
var envolverHandler = func(h http.Handler, _ *slog.Logger) http.Handler { return h }

// configuracion es el archivo config.json de la carpeta de datos (se puede editar a mano).
type configuracion struct {
	Servidor string `json:"servidor"`
}

func main() {
	carpeta, err := carpetaDatos()
	if err != nil {
		fatal(err)
	}
	logArchivo, err := os.OpenFile(filepath.Join(carpeta, "magui.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fatal(err)
	}
	defer logArchivo.Close()
	log := slog.New(slog.NewTextHandler(logArchivo, nil))

	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()

	var handler http.Handler
	var app *escritorio.App
	cfg, err := leerConfiguracion(carpeta)
	if err == nil {
		sub, _ := fs.Sub(archivos, "web/browser")
		app, err = escritorio.Abrir(ctx, escritorio.Config{
			Servidor: cfg.Servidor,
			Datos:    filepath.Join(carpeta, "magui.db"),
			Frontend: escritorio.SPA(sub),
			Log:      log,
		})
	}
	if err != nil {
		log.Error("no se pudo abrir el escritorio", "error", err)
		handler = paginaError(err, carpeta)
	} else {
		defer app.Cerrar()
		handler = app.Handler
		go app.Ejecutar(ctx)
		log.Info("escritorio iniciado", "version", version, "servidor", cfg.Servidor, "datos", carpeta)
	}

	var ventana context.Context
	err = wails.Run(&options.App{
		Title:            "Magui",
		Width:            1280,
		Height:           820,
		MinWidth:         1024,
		MinHeight:        680,
		BackgroundColour: &options.RGBA{R: 250, G: 250, B: 250, A: 255},
		AssetServer:      &assetserver.Options{Handler: envolverHandler(handler, log)},
		// Una sola ventana: si se abre otra vez, se muestra la que ya está abierta
		// (dos procesos sincronizando la misma base no tienen sentido).
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "ar.magui.escritorio",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) {
				if ventana != nil {
					runtime.WindowUnminimise(ventana)
					runtime.WindowShow(ventana)
				}
			},
		},
		OnStartup:  func(c context.Context) { ventana = c },
		OnShutdown: func(context.Context) { cancelar() },
		Windows: &windows.Options{
			// Sesión y preferencias de la pantalla junto a los datos de la app.
			WebviewUserDataPath: filepath.Join(carpeta, "webview"),
		},
	})
	if err != nil {
		log.Error("wails", "error", err)
	}
}

// carpetaDatos es %APPDATA%\Magui (se crea si no existe). MAGUI_DATOS la cambia (pruebas).
func carpetaDatos() (string, error) {
	if dir := os.Getenv("MAGUI_DATOS"); dir != "" {
		return dir, os.MkdirAll(dir, 0o700)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "Magui")
	return dir, os.MkdirAll(dir, 0o700)
}

// leerConfiguracion lee config.json; si no existe lo crea con el servidor por defecto.
func leerConfiguracion(carpeta string) (configuracion, error) {
	ruta := filepath.Join(carpeta, "config.json")
	b, err := os.ReadFile(ruta)
	if errors.Is(err, fs.ErrNotExist) {
		cfg := configuracion{Servidor: servidorPorDefecto}
		b, _ = json.MarshalIndent(cfg, "", "  ")
		return cfg, os.WriteFile(ruta, b, 0o600)
	}
	if err != nil {
		return configuracion{}, err
	}
	var cfg configuracion
	if err := json.Unmarshal(b, &cfg); err != nil {
		return configuracion{}, fmt.Errorf("%s no es JSON válido: %w", ruta, err)
	}
	if cfg.Servidor == "" {
		cfg.Servidor = servidorPorDefecto
	}
	return cfg, nil
}

// paginaError muestra en la ventana por qué no arrancó (en vez de cerrarse sin avisar).
func paginaError(err error, carpeta string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `<!doctype html><html lang="es"><meta charset="utf-8"><title>Magui</title>
<body style="font-family:system-ui,sans-serif;max-width:40rem;margin:4rem auto;padding:0 1rem;color:#1f2937">
<h1 style="color:#be185d">Magui no pudo arrancar</h1><p>%s</p>
<p>Revisá la dirección del servidor en <code>%s</code> y volvé a abrir la app.
El detalle queda en <code>%s</code>.</p></body></html>`,
			html.EscapeString(err.Error()), html.EscapeString(filepath.Join(carpeta, "config.json")),
			html.EscapeString(filepath.Join(carpeta, "magui.log")))
	})
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "magui:", err)
	os.Exit(1)
}
