package http_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"magui/internal/catalogo"
	catapp "magui/internal/catalogo/application"
	catdomain "magui/internal/catalogo/domain"
	"magui/internal/compras"
	comapp "magui/internal/compras/application"
	"magui/internal/inventario"
	"magui/internal/kernel/dinero"
	"magui/internal/platform/db"
	sincro "magui/internal/sync"
	"magui/internal/sync/application"
	"magui/internal/ventas"
)

// Integración de la sincronización sobre MySQL real (MAGUI_TEST_MYSQL_DSN).

func baseTemporal(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("MAGUI_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("MAGUI_TEST_MYSQL_DSN no definido")
	}
	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	nombre := fmt.Sprintf("magui_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + nombre + " CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec("DROP DATABASE " + nombre); admin.Close() })
	conn, err := db.OpenMySQL(context.Background(), dsn+nombre+"?parseTime=true&loc=UTC")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	m, _ := db.NewMigrator(conn, db.MySQL)
	if err := m.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	return conn
}

func debe[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func d(s string) dinero.Dinero { return dinero.MustParse(s) }

func pedir[T any](t *testing.T, srv http.Handler, metodo, ruta string, cuerpo any) (int, T) {
	t.Helper()
	var body io.Reader
	if cuerpo != nil {
		b, _ := json.Marshal(cuerpo)
		body = bytes.NewReader(b)
	}
	req := httptest.NewRequest(metodo, ruta, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s %s: %d %s", metodo, ruta, rec.Code, rec.Body.String())
	}
	return rec.Code, out
}

func TestSincronizacionMySQL(t *testing.T) {
	conn := baseTemporal(t)
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cat := catalogo.NuevoModuloMySQL(conn, log).Servicio
	inv := inventario.NuevoModuloMySQL(conn, cat, log).Servicio
	com := compras.NuevoModuloMySQL(conn, cat, inv, log).Servicio
	ven := ventas.NuevoModuloMySQL(conn, cat, inv, nil, log).Servicio
	r := chi.NewRouter()
	sincro.NuevoModulo(cat, ven, log).Handler.Routes(r)

	nuevo := func(sku, nombre, precio string) catapp.ProductoDetalle {
		return debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoSimple,
			Datos: catdomain.DatosProducto{SKU: sku, Nombre: nombre, PrecioVenta: d(precio)}}))
	}
	kaiak, jabon := nuevo("P1", "Kaiak", "59200"), nuevo("P2", "Jabón", "5500")
	combo := debe(cat.CrearProducto(ctx, catapp.ComandoProducto{Tipo: catdomain.TipoCombo, Datos: catdomain.DatosProducto{
		Nombre: "Combo Madre", PrecioVenta: d("62000"),
		Items: []catdomain.ComboItem{{ProductoID: kaiak.ID, Cantidad: 1}, {ProductoID: jabon.ID, Cantidad: 2}}}}))
	debe(com.RegistrarCompra(ctx, comapp.ComandoCompra{Items: []comapp.ItemCmd{
		{ProductoID: kaiak.ID, Cantidad: 3, CostoUnitario: d("30000")},
		{ProductoID: jabon.ID, Cantidad: 10, CostoUnitario: d("2500")},
	}}))

	// --- Pull: copia del catálogo ---
	code, copia := pedir[application.CopiaCatalogo](t, r, http.MethodGet, "/sync/catalogo", nil)
	if code != 200 || len(copia.Productos) != 3 || len(copia.Componentes) != 2 || copia.GeneradaEn.IsZero() {
		t.Fatalf("copia: %d %+v", code, copia)
	}
	stock := map[string]int{}
	for _, p := range copia.Productos {
		stock[p.ID] = p.Stock
	}
	if stock[kaiak.ID] != 3 || stock[jabon.ID] != 10 || stock[combo.ID] != 3 {
		t.Errorf("stock en la copia = %v", stock)
	}

	// --- Push: dos ventas offline (una con precio viejo, otra combo) y una inválida ---
	ayer := time.Now().UTC().Add(-20 * time.Hour).Truncate(time.Second)
	v1 := application.VentaOffline{ID: "0192f3a0-0000-7000-8000-000000000001", Fecha: ayer,
		Items: []application.ItemOffline{{ProductoID: kaiak.ID, Cantidad: 4, Precio: d("55000")}}, // 4 > stock: se registra igual
		Pagos: []application.PagoOffline{{Metodo: "EFECTIVO", Monto: d("200000")}, {Metodo: "QR", Monto: d("20000")}}}
	v2 := application.VentaOffline{ID: "0192f3a0-0000-7000-8000-000000000002", Fecha: ayer.Add(time.Hour),
		Items: []application.ItemOffline{{ProductoID: combo.ID, Cantidad: 1, Precio: d("62000")}},
		Pagos: []application.PagoOffline{{Metodo: "TARJETA", Monto: d("62000")}}}
	mala := application.VentaOffline{ID: "0192f3a0-0000-7000-8000-000000000003", Fecha: ayer,
		Items: []application.ItemOffline{{ProductoID: jabon.ID, Cantidad: 1, Precio: d("5500")}},
		Pagos: []application.PagoOffline{{Metodo: "BITCOIN", Monto: d("5500")}}}

	envio := application.EnvioVentas{Ventas: []application.VentaOffline{v1, v2, mala}}
	code, res := pedir[application.RespuestaEnvio](t, r, http.MethodPost, "/sync/ventas", envio)
	if code != 200 || len(res.Resultados) != 3 {
		t.Fatalf("envío: %d %+v", code, res)
	}
	esperado := []application.EstadoEnvio{application.Registrada, application.Registrada, application.Rechazada}
	for i, e := range esperado {
		if res.Resultados[i].Estado != e || res.Resultados[i].ID != envio.Ventas[i].ID {
			t.Errorf("resultado %d = %+v, want %s", i, res.Resultados[i], e)
		}
	}
	if e := res.Resultados[2].Error; e == nil || e.Code != "datos_invalidos" {
		t.Errorf("error de la inválida = %+v", e)
	}

	det := debe(ven.ObtenerVenta(ctx, v1.ID))
	if det.Total.String() != "220000.00" || det.Origen != "DESKTOP" || !det.Fecha.Equal(ayer) || det.Vuelto.String() != "0.00" {
		t.Errorf("venta 1 = total %s origen %s fecha %v vuelto %s", det.Total, det.Origen, det.Fecha, det.Vuelto)
	}
	if det.CostoTotal.String() != "120000.00" { // 3 del lote a 30000 + 1 sin lote al último costo
		t.Errorf("costo venta 1 = %s", det.CostoTotal)
	}

	// --- Reenvío (se cortó la conexión antes de recibir la respuesta): no duplica ---
	code, res = pedir[application.RespuestaEnvio](t, r, http.MethodPost, "/sync/ventas", application.EnvioVentas{Ventas: []application.VentaOffline{v1, v2}})
	if code != 200 || res.Resultados[0].Estado != application.YaExistia || res.Resultados[1].Estado != application.YaExistia {
		t.Errorf("reenvío = %d %+v", code, res)
	}
	var ventasN int
	_ = conn.QueryRow(`SELECT COUNT(*) FROM venta`).Scan(&ventasN)
	if ventasN != 2 {
		t.Errorf("ventas = %d, want 2", ventasN)
	}

	// El stock refleja las ventas offline: Kaiak 3 - 4 - 1 (combo) = -2; jabón 10 - 2 = 8
	_, copia = pedir[application.CopiaCatalogo](t, r, http.MethodGet, "/sync/catalogo", nil)
	for _, p := range copia.Productos {
		stock[p.ID] = p.Stock
	}
	if stock[kaiak.ID] != -2 || stock[jabon.ID] != 8 || stock[combo.ID] != 0 {
		t.Errorf("stock después = %v", stock)
	}

	// Demasiadas ventas en un envío
	muchas := make([]application.VentaOffline, application.MaxVentasPorEnvio+1)
	code, _ = pedir[map[string]any](t, r, http.MethodPost, "/sync/ventas", application.EnvioVentas{Ventas: muchas})
	if code != http.StatusUnprocessableEntity {
		t.Errorf("demasiadas: %d", code)
	}
}
