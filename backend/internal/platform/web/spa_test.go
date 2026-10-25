package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

var archivos = fstest.MapFS{
	"index.html":           {Data: []byte("<app-root>")},
	"main-3AW3IKRT.js":     {Data: []byte("console.log(1)")},
	"ngsw.json":            {Data: []byte("{}")},
	"manifest.webmanifest": {Data: []byte("{}")},
	"icons/icon-72x72.png": {Data: []byte("png")},
}

func pedir(t *testing.T, metodo, ruta string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	Seguridad(SPA(archivos)).ServeHTTP(rec, httptest.NewRequest(metodo, ruta, nil))
	return rec
}

func TestSPA(t *testing.T) {
	casos := []struct {
		ruta, cuerpo, cache string
		status              int
	}{
		{"/", "<app-root>", "no-cache", 200},
		{"/vender", "<app-root>", "no-cache", 200},        // ruta de la app -> index.html
		{"/productos/123", "<app-root>", "no-cache", 200}, // ruta anidada
		{"/main-3AW3IKRT.js", "console.log(1)", "public, max-age=31536000, immutable", 200},
		{"/ngsw.json", "{}", "no-cache", 200},
		{"/icons/icon-72x72.png", "png", "public, max-age=3600", 200},
		{"/no-existe.js", "", "", 404},     // archivo inexistente: no devuelve index
		{"/../../etc/passwd", "", "", 400}, // rutas con ".." se rechazan
	}
	for _, c := range casos {
		rec := pedir(t, http.MethodGet, c.ruta)
		if rec.Code != c.status {
			t.Errorf("%s: status %d, want %d", c.ruta, rec.Code, c.status)
			continue
		}
		if c.status == 200 && (rec.Body.String() != c.cuerpo || rec.Header().Get("Cache-Control") != c.cache) {
			t.Errorf("%s: cuerpo %q cache %q", c.ruta, rec.Body.String(), rec.Header().Get("Cache-Control"))
		}
	}
	if ct := pedir(t, http.MethodGet, "/manifest.webmanifest").Header().Get("Content-Type"); ct != "application/manifest+json" {
		t.Errorf("tipo del manifiesto = %q", ct)
	}
	if rec := pedir(t, http.MethodPost, "/vender"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: %d", rec.Code)
	}
}

func TestSeguridad(t *testing.T) {
	h := pedir(t, http.MethodGet, "/").Header()
	for _, k := range []string{"X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy", "Referrer-Policy"} {
		if h.Get(k) == "" {
			t.Errorf("falta %s", k)
		}
	}
}
