package importacion

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// crearExcel arma un archivo con el mismo formato que el original de la clienta.
func crearExcel(t *testing.T, filas [][]any, nombresHoja1 []string) string {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", hojaProductos); err != nil {
		t.Fatal(err)
	}
	enc := []any{"Nombre", 1, "SKU", "Precio", "Precio oferta", "FECHA DE VEC", "DIAS RESTANTES", "ESTADO",
		"Valor atributo 2", "Nombre atributo 3", "Valor atributo 3", "Categorías", "Peso", "Alto", "Ancho",
		"Profundidad", "Mostrar en tienda", "Descripción"}
	if err := f.SetSheetRow(hojaProductos, "A1", &enc); err != nil {
		t.Fatal(err)
	}
	for i, r := range filas {
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		if err := f.SetSheetRow(hojaProductos, cell, &r); err != nil {
			t.Fatal(err)
		}
	}
	if nombresHoja1 != nil {
		if _, err := f.NewSheet(hojaNombres); err != nil {
			t.Fatal(err)
		}
		_ = f.SetCellValue(hojaNombres, "A1", "Nombre")
		for i, n := range nombresHoja1 {
			cell, _ := excelize.CoordinatesToCellName(1, i+2)
			_ = f.SetCellValue(hojaNombres, cell, n)
		}
	}
	ruta := filepath.Join(t.TempDir(), "stock.xlsx")
	if err := f.SaveAs(ruta); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func fila(nombre any, stock any, sku string, precio float64, venc any, cats string) []any {
	return []any{nombre, stock, sku, precio, precio * 0.6, venc, nil, nil, nil, nil, nil, cats,
		0.1, 10, 10, 20, "Si", "<div>desc</div>"}
}

func TestLeerExcel(t *testing.T) {
	venc := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	ruta := crearExcel(t, [][]any{
		fila("Kaiak  Clásico femenino ", 3, "P1", 59200, venc, "PERFUMERÍA > Perfumería femenina | REGALOS"),
		fila(nil, 1, "P2", 1500.5, nil, "REGALOS | REGALOS"), // sin nombre: se toma de Hoja 1
	}, []string{"Kaiak Clásico femenino", "Kit Regalo Ekos"})

	filas, err := LeerExcel(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if len(filas) != 2 {
		t.Fatalf("filas = %d", len(filas))
	}

	a := filas[0]
	if a.Nombre != "Kaiak Clásico femenino" || a.SKU != "P1" || a.Stock != 3 || a.PrecioCentavos != 5920000 {
		t.Errorf("fila 1 inesperada: %+v", a)
	}
	if a.Vencimiento == nil || !a.Vencimiento.Equal(venc) {
		t.Errorf("vencimiento = %v", a.Vencimiento)
	}
	if want := [][]string{{"PERFUMERÍA", "Perfumería femenina"}, {"REGALOS"}}; !reflect.DeepEqual(a.Categorias, want) {
		t.Errorf("categorías = %v", a.Categorias)
	}
	if a.ProfundidadCm == nil || *a.ProfundidadCm != 20 || !a.MostrarEnTienda {
		t.Errorf("medidas/mostrar: %+v", a)
	}

	b := filas[1]
	if b.Nombre != "Kit Regalo Ekos" {
		t.Errorf("nombre de respaldo = %q", b.Nombre)
	}
	if b.PrecioCentavos != 150050 || b.Vencimiento != nil {
		t.Errorf("fila 2 inesperada: %+v", b)
	}
	if len(b.Categorias) != 1 {
		t.Errorf("categorías duplicadas no eliminadas: %v", b.Categorias)
	}
}

func TestLeerExcelInformaErrores(t *testing.T) {
	ruta := crearExcel(t, [][]any{
		fila("A", 2.5, "P1", 100, nil, "X"), // stock con decimales
		fila("B", 1, "P2", -1, nil, "X"),    // precio negativo
		fila(nil, 1, "P3", 100, nil, "X"),   // sin nombre y sin Hoja 1
	}, nil)
	_, err := LeerExcel(ruta)
	if err == nil {
		t.Fatal("se esperaba error")
	}
	for _, sku := range []string{"P1", "P2", "P3"} {
		if !strings.Contains(err.Error(), sku) {
			t.Errorf("el error no menciona %s: %v", sku, err)
		}
	}
}

func TestNormalizar(t *testing.T) {
	if got := normalizar("  PERFUMERÍA   Femenina "); got != "perfumeria femenina" {
		t.Errorf("normalizar = %q", got)
	}
}

func TestCentavosADecimal(t *testing.T) {
	for c, want := range map[int64]string{5920000: "59200.00", 150050: "1500.50", 5: "0.05"} {
		if got := centavosADecimal(c); got != want {
			t.Errorf("%d -> %q, want %q", c, got, want)
		}
	}
}
