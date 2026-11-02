package mysql

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cathttp "magui/internal/catalogo/adapters/http"
	"magui/internal/catalogo/application"
	"magui/internal/catalogo/domain"
	"magui/internal/kernel/paginacion"
	"magui/internal/platform/httpserver"
)

var pngChico = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

// US-004: foto opcional de cada producto.
func TestImagenProductoMySQL(t *testing.T) {
	conn := baseTemporal(t)
	ctx := context.Background()
	svc := servicio(conn)
	p := debe(svc.CrearProducto(ctx, simple("F1", "Kaiak", "1000")))
	if p.ImagenVersion != 0 {
		t.Fatalf("producto nuevo con foto: %d", p.ImagenVersion)
	}

	v1 := debe(svc.CambiarImagen(ctx, p.ID, pngChico))
	if v1.ImagenVersion != 1 {
		t.Errorf("versión = %d", v1.ImagenVersion)
	}
	datos, tipo, err := svc.Imagen(ctx, p.ID)
	if err != nil || tipo != "image/png" || !bytes.Equal(datos, pngChico) {
		t.Errorf("leer: %q %v", tipo, err)
	}
	// El listado y el detalle traen la versión para armar la URL.
	pag := debe(svc.ListarProductos(ctx, application.ConsultaProductos{Texto: "kaiak"}, paginacion.Solicitud{Pagina: 1, Tamano: 10}))
	if len(pag.Items) != 1 || pag.Items[0].ImagenVersion != 1 {
		t.Errorf("listado = %+v", pag.Items)
	}

	// Quitar y volver a subir: la versión nunca se repite (la URL vieja queda en caché).
	if v := debe(svc.CambiarImagen(ctx, p.ID, nil)); v.ImagenVersion != 0 {
		t.Errorf("quitar: %d", v.ImagenVersion)
	}
	if _, _, err := svc.Imagen(ctx, p.ID); !errors.Is(err, domain.ErrSinImagen) {
		t.Errorf("sin foto: %v", err)
	}
	if det := debe(svc.ObtenerProducto(ctx, p.ID)); det.ImagenVersion != 0 {
		t.Errorf("detalle sin foto: %d", det.ImagenVersion)
	}
	if v := debe(svc.CambiarImagen(ctx, p.ID, pngChico)); v.ImagenVersion != 2 {
		t.Errorf("volver a subir: %d", v.ImagenVersion)
	}

	if _, err := svc.CambiarImagen(ctx, "no-existe", pngChico); !errors.Is(err, domain.ErrProductoNoEncontrado) {
		t.Errorf("producto inexistente: %v", err)
	}
	if _, err := svc.CambiarImagen(ctx, p.ID, []byte("<svg/>")); !errors.Is(err, domain.ErrImagenTipo) {
		t.Errorf("svg: %v", err)
	}
	// Al borrar el producto se va la foto (no hay borrado de productos en la app, pero la FK lo asegura).
	if _, err := conn.Exec(`DELETE FROM producto_imagen WHERE producto_id = ?`, p.ID); err != nil {
		t.Fatal(err)
	}
}

// La ruta pública de la foto convive con las de /productos (protegidas) en el router real.
func TestImagenProductoRutasMySQL(t *testing.T) {
	conn := baseTemporal(t)
	ctx := context.Background()
	svc := servicio(conn)
	p := debe(svc.CrearProducto(ctx, simple("F2", "Jabón", "500")))
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer ok" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	h := httpserver.NewRouter(log, httpserver.Options{Autenticacion: auth}, cathttp.NuevoHandler(svc, log))
	pedir := func(metodo, ruta string, cuerpo []byte, sesion bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(metodo, ruta, bytes.NewReader(cuerpo))
		if sesion {
			req.Header.Set("Authorization", "Bearer ok")
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	if rec := pedir("PUT", "/api/productos/"+p.ID+"/imagen", pngChico, false); rec.Code != 401 {
		t.Errorf("subir sin sesión: %d", rec.Code)
	}
	if rec := pedir("PUT", "/api/productos/"+p.ID+"/imagen", pngChico, true); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"imagenVersion":1`) {
		t.Errorf("subir: %d %s", rec.Code, rec.Body)
	}
	if rec := pedir("PUT", "/api/productos/"+p.ID+"/imagen", make([]byte, domain.TamanoMaximoImagen+5), true); rec.Code != 422 ||
		!strings.Contains(rec.Body.String(), "imagen_muy_grande") {
		t.Errorf("muy grande: %d %s", rec.Code, rec.Body)
	}
	rec := pedir("GET", "/api/imagenes/productos/"+p.ID+"?v=1", nil, false)
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("foto pública: %d %q %q", rec.Code, rec.Header().Get("Content-Type"), rec.Header().Get("Cache-Control"))
	}
	if rec := pedir("GET", "/api/productos/"+p.ID, nil, false); rec.Code != 401 {
		t.Errorf("la ficha del producto sigue protegida: %d", rec.Code)
	}
	if rec := pedir("DELETE", "/api/productos/"+p.ID+"/imagen", nil, true); rec.Code != 200 {
		t.Errorf("quitar: %d", rec.Code)
	}
	if rec := pedir("GET", "/api/imagenes/productos/"+p.ID, nil, false); rec.Code != 404 {
		t.Errorf("sin foto: %d", rec.Code)
	}
}
