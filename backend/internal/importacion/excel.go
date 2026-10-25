// Package importacion carga el Excel histórico de la clienta ("STOCK VENCIMIENTOS.xlsx")
// en el esquema v2 de MySQL. Es una herramienta de única vez (adaptador de entrada),
// por eso escribe SQL directo en lugar de pasar por los casos de uso.
//
// Es idempotente: los productos cuyo SKU ya existe se omiten.
package importacion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	hojaProductos = "Productos"
	hojaNombres   = "Hoja 1" // tiene los nombres de los kits que faltan en "Productos"
)

// Opciones de la importación.
type Opciones struct {
	// Empresa a la que se asignan todos los productos importados (opcional). Se crea si no existe.
	Empresa string
	// Ahora permite fijar la fecha en tests.
	Ahora func() time.Time
}

// Resultado resume lo importado.
type Resultado struct {
	Productos      int
	Omitidos       int // SKU ya existente
	Categorias     int // nuevas
	Lotes          int
	Unidades       int
	ConVencimiento int
}

// Fila es un producto leído del Excel, ya limpio.
type Fila struct {
	NumeroFila      int
	SKU             string
	Nombre          string
	Stock           int
	PrecioCentavos  int64
	Vencimiento     *time.Time
	Categorias      [][]string // cada ruta: ["PERFUMERÍA", "Perfumería femenina"]
	PesoKg          *float64
	AltoCm          *float64
	AnchoCm         *float64
	ProfundidadCm   *float64
	MostrarEnTienda bool
	Descripcion     string
}

// LeerExcel lee y valida las filas del archivo sin tocar la base.
func LeerExcel(ruta string) ([]Fila, error) {
	f, err := excelize.OpenFile(ruta)
	if err != nil {
		return nil, fmt.Errorf("abrir excel: %w", err)
	}
	defer f.Close()

	filas, err := f.GetRows(hojaProductos, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, fmt.Errorf("leer hoja %q: %w", hojaProductos, err)
	}
	if len(filas) < 2 {
		return nil, errors.New("la hoja Productos no tiene datos")
	}
	col, err := mapearColumnas(filas[0])
	if err != nil {
		return nil, err
	}

	// Nombres de respaldo de "Hoja 1", alineados por número de fila
	var nombres [][]string
	if idx, _ := f.GetSheetIndex(hojaNombres); idx >= 0 {
		nombres, _ = f.GetRows(hojaNombres)
	}

	var out []Fila
	var errs []error
	for i := 1; i < len(filas); i++ {
		r := filas[i]
		if filaVacia(r) {
			continue
		}
		fila, err := parsearFila(r, col, i+1)
		if fila.Nombre == "" && i < len(nombres) && len(nombres[i]) > 0 {
			fila.Nombre = limpiar(nombres[i][0])
		}
		if err == nil && fila.Nombre == "" {
			err = fmt.Errorf("fila %d (SKU %s): sin nombre", i+1, fila.SKU)
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, fila)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

type columnas map[string]int

// mapearColumnas ubica cada dato por el título de la columna (sin importar mayúsculas ni tildes).
func mapearColumnas(enc []string) (columnas, error) {
	alias := map[string][]string{
		"nombre":      {"nombre"},
		"stock":       {"stock"},
		"sku":         {"sku"},
		"precio":      {"precio"},
		"vencimiento": {"fecha de vec", "fecha de vencimiento", "vencimiento"},
		"categorias":  {"categorias"},
		"peso":        {"peso"},
		"alto":        {"alto"},
		"ancho":       {"ancho"},
		"profundidad": {"profundidad"},
		"mostrar":     {"mostrar en tienda"},
		"descripcion": {"descripcion"},
	}
	indice := make(map[string]int, len(enc))
	for i, h := range enc {
		indice[normalizar(h)] = i
	}
	col := columnas{}
	for clave, nombres := range alias {
		for _, n := range nombres {
			if i, ok := indice[n]; ok {
				col[clave] = i
				break
			}
		}
	}
	// En el Excel original la columna de stock no tiene título (dice "1"); es la segunda.
	if _, ok := col["stock"]; !ok {
		col["stock"] = 1
	}
	for _, req := range []string{"nombre", "sku", "precio", "categorias"} {
		if _, ok := col[req]; !ok {
			return nil, fmt.Errorf("falta la columna %q en la hoja Productos", req)
		}
	}
	return col, nil
}

func parsearFila(r []string, col columnas, nro int) (Fila, error) {
	celda := func(clave string) string {
		i, ok := col[clave]
		if !ok || i >= len(r) {
			return ""
		}
		return strings.TrimSpace(r[i])
	}
	fila := Fila{
		NumeroFila:      nro,
		SKU:             limpiar(celda("sku")),
		Nombre:          limpiar(celda("nombre")),
		Descripcion:     celda("descripcion"),
		MostrarEnTienda: true,
	}
	if fila.SKU == "" {
		return fila, fmt.Errorf("fila %d: sin SKU", nro)
	}
	errFila := func(campo string, err error) error {
		return fmt.Errorf("fila %d (SKU %s): %s inválido: %w", nro, fila.SKU, campo, err)
	}

	precio, err := strconv.ParseFloat(celda("precio"), 64)
	if err != nil || precio < 0 {
		return fila, errFila("precio", fmt.Errorf("%q", celda("precio")))
	}
	fila.PrecioCentavos = int64(math.Round(precio * 100))

	if s := celda("stock"); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v < 0 || v != math.Trunc(v) {
			return fila, errFila("stock", fmt.Errorf("%q", s))
		}
		fila.Stock = int(v)
	}

	if s := celda("vencimiento"); s != "" {
		serial, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fila, errFila("vencimiento", err)
		}
		t, err := excelize.ExcelDateToTime(serial, false)
		if err != nil {
			return fila, errFila("vencimiento", err)
		}
		d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		fila.Vencimiento = &d
	}

	for clave, dst := range map[string]**float64{
		"peso": &fila.PesoKg, "alto": &fila.AltoCm, "ancho": &fila.AnchoCm, "profundidad": &fila.ProfundidadCm,
	} {
		if s := celda(clave); s != "" {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return fila, errFila(clave, err)
			}
			*dst = &v
		}
	}

	if s := normalizar(celda("mostrar")); s == "no" {
		fila.MostrarEnTienda = false
	}

	fila.Categorias = parsearCategorias(celda("categorias"))
	return fila, nil
}

// parsearCategorias: "A > B | C" -> [[A B] [C]], sin duplicados.
func parsearCategorias(s string) [][]string {
	var out [][]string
	vistas := map[string]bool{}
	for ruta := range strings.SplitSeq(s, "|") {
		var partes []string
		for p := range strings.SplitSeq(ruta, ">") {
			if p = limpiar(p); p != "" {
				partes = append(partes, p)
			}
		}
		if len(partes) == 0 {
			continue
		}
		clave := normalizar(strings.Join(partes, ">"))
		if vistas[clave] {
			continue
		}
		vistas[clave] = true
		out = append(out, partes)
	}
	return out
}

// Importar escribe las filas en MySQL dentro de una única transacción.
func Importar(ctx context.Context, conn *sql.DB, filas []Fila, opt Opciones) (Resultado, error) {
	ahora := time.Now
	if opt.Ahora != nil {
		ahora = opt.Ahora
	}
	var res Resultado

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return res, err
	}
	defer func() { _ = tx.Rollback() }()

	var empresaID *string
	if e := limpiar(opt.Empresa); e != "" {
		id, err := buscarOCrearEmpresa(ctx, tx, e)
		if err != nil {
			return res, err
		}
		empresaID = &id
	}

	cats, err := cargarCategorias(ctx, tx)
	if err != nil {
		return res, err
	}

	fecha := ahora().UTC()
	for _, f := range filas {
		var existe int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM producto WHERE sku = ?`, f.SKU).Scan(&existe); err != nil {
			return res, err
		}
		if existe > 0 {
			res.Omitidos++
			continue
		}

		prodID := nuevoID()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO producto (id, sku, nombre, descripcion, empresa_id, tipo, precio_venta,
			                      peso_kg, alto_cm, ancho_cm, profundidad_cm, mostrar_en_tienda)
			VALUES (?, ?, ?, ?, ?, 'SIMPLE', ?, ?, ?, ?, ?, ?)`,
			prodID, f.SKU, f.Nombre, nullString(f.Descripcion), empresaID, centavosADecimal(f.PrecioCentavos),
			f.PesoKg, f.AltoCm, f.AnchoCm, f.ProfundidadCm, f.MostrarEnTienda)
		if err != nil {
			return res, fmt.Errorf("fila %d (SKU %s): %w", f.NumeroFila, f.SKU, err)
		}
		res.Productos++

		asignadas := map[string]bool{}
		for _, ruta := range f.Categorias {
			catID, nuevas, err := cats.asegurarRuta(ctx, tx, ruta)
			if err != nil {
				return res, err
			}
			res.Categorias += nuevas
			if asignadas[catID] {
				continue
			}
			asignadas[catID] = true
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO producto_categoria (producto_id, categoria_id) VALUES (?, ?)`, prodID, catID); err != nil {
				return res, err
			}
		}

		if f.Stock > 0 {
			loteID := nuevoID()
			var venc *string
			if f.Vencimiento != nil {
				s := f.Vencimiento.Format(time.DateOnly)
				venc = &s
				res.ConVencimiento++
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO lote (id, producto_id, fecha_vencimiento) VALUES (?, ?, ?)`,
				loteID, prodID, venc); err != nil {
				return res, err
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO movimiento_stock (id, producto_id, lote_id, cantidad, tipo, nota, fecha)
				VALUES (?, ?, ?, ?, 'INICIAL', 'Importación desde Excel', ?)`,
				nuevoID(), prodID, loteID, f.Stock, fecha); err != nil {
				return res, err
			}
			res.Lotes++
			res.Unidades += f.Stock
		}
	}

	if err := tx.Commit(); err != nil {
		return res, err
	}
	return res, nil
}

func buscarOCrearEmpresa(ctx context.Context, tx *sql.Tx, nombre string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `SELECT id FROM empresa WHERE nombre = ?`, nombre).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	id = nuevoID()
	_, err = tx.ExecContext(ctx, `INSERT INTO empresa (id, nombre) VALUES (?, ?)`, id, nombre)
	return id, err
}

// cacheCategorias evita consultar la base por cada producto. Clave: padreID + "/" + nombre normalizado.
type cacheCategorias map[string]string

func cargarCategorias(ctx context.Context, tx *sql.Tx) (cacheCategorias, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, COALESCE(padre_id, ''), nombre FROM categoria`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	c := cacheCategorias{}
	for rows.Next() {
		var id, padre, nombre string
		if err := rows.Scan(&id, &padre, &nombre); err != nil {
			return nil, err
		}
		c[padre+"/"+normalizar(nombre)] = id
	}
	return c, rows.Err()
}

func (c cacheCategorias) asegurarRuta(ctx context.Context, tx *sql.Tx, ruta []string) (id string, nuevas int, err error) {
	padre := ""
	for _, nombre := range ruta {
		clave := padre + "/" + normalizar(nombre)
		catID, ok := c[clave]
		if !ok {
			catID = nuevoID()
			var padreID *string
			if padre != "" {
				p := padre
				padreID = &p
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO categoria (id, nombre, padre_id) VALUES (?, ?, ?)`, catID, nombre, padreID); err != nil {
				return "", nuevas, fmt.Errorf("crear categoría %q: %w", nombre, err)
			}
			c[clave] = catID
			nuevas++
		}
		padre = catID
	}
	return padre, nuevas, nil
}

// --- utilidades ---

func nuevoID() string { return uuid.Must(uuid.NewV7()).String() }

func centavosADecimal(c int64) string { return fmt.Sprintf("%d.%02d", c/100, c%100) }

func nullString(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// limpiar quita espacios sobrantes (inicio, fin y repetidos).
func limpiar(s string) string { return strings.Join(strings.Fields(s), " ") }

// normalizar: minúsculas, sin tildes y sin espacios sobrantes (para comparar).
func normalizar(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	r, _, _ := transform.String(t, limpiar(s))
	return strings.ToLower(r)
}

func filaVacia(r []string) bool {
	for _, v := range r {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}
