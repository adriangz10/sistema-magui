//go:build pruebas

package main

import (
	"log/slog"
	"net/http"
)

// Compilación de pruebas (wails build -tags pruebas): anota en magui.log cada pedido que hace la
// ventana, para comprobar desde afuera que la app carga y habla con el núcleo local.
func init() {
	envolverHandler = func(h http.Handler, log *slog.Logger) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &registro{ResponseWriter: w, status: http.StatusOK}
			h.ServeHTTP(rec, r)
			log.Info("pedido", "metodo", r.Method, "ruta", r.URL.RequestURI(), "status", rec.status)
		})
	}
}

type registro struct {
	http.ResponseWriter
	status int
}

func (r *registro) WriteHeader(s int) { r.status = s; r.ResponseWriter.WriteHeader(s) }
