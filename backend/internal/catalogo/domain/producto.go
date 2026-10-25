package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
)

// Tipo de producto.
type Tipo string

const (
	TipoSimple Tipo = "SIMPLE"
	TipoCombo  Tipo = "COMBO" // compuesto por otros productos; no tiene stock propio
)

// ComboItem es un componente de un combo.
type ComboItem struct {
	ProductoID string
	Cantidad   int
}

// Producto es un artículo que se vende.
type Producto struct {
	ID              string
	SKU             string
	CodigoBarras    *string
	Nombre          string
	Descripcion     *string
	EmpresaID       *string
	Tipo            Tipo
	PrecioVenta     dinero.Dinero
	Costo           *dinero.Dinero // último costo de compra conocido
	StockMinimo     int
	MostrarEnTienda bool
	Activo          bool
	CategoriaIDs    []string
	Items           []ComboItem // solo combos
	Version         int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// DatosProducto son los campos editables de un producto.
type DatosProducto struct {
	SKU             string
	CodigoBarras    string
	Nombre          string
	Descripcion     string
	EmpresaID       string
	PrecioVenta     dinero.Dinero
	Costo           *dinero.Dinero
	StockMinimo     int
	MostrarEnTienda bool
	CategoriaIDs    []string
	Items           []ComboItem
}

// NuevoProducto crea un producto activo. Si no viene SKU se genera uno a partir del ID.
func NuevoProducto(id string, tipo Tipo, d DatosProducto, ahora time.Time) (Producto, error) {
	if tipo != TipoSimple && tipo != TipoCombo {
		return Producto{}, errores.Invalido("tipo_invalido", "El tipo debe ser SIMPLE o COMBO")
	}
	if strings.TrimSpace(d.SKU) == "" {
		d.SKU = SKUAutomatico(tipo, id)
	}
	p := Producto{ID: id, Tipo: tipo, Activo: true, CreatedAt: ahora}
	if _, err := p.Modificar(d, ahora); err != nil {
		return Producto{}, err
	}
	p.Version = 1
	return p, nil
}

// SKUAutomatico genera un SKU legible: "MG-1A2B3C4D" o "CB-1A2B3C4D" para combos.
func SKUAutomatico(tipo Tipo, id string) string {
	prefijo := "MG-"
	if tipo == TipoCombo {
		prefijo = "CB-"
	}
	limpio := strings.ToUpper(strings.ReplaceAll(id, "-", ""))
	if len(limpio) > 8 {
		limpio = limpio[len(limpio)-8:]
	}
	return prefijo + limpio
}

// Modificar valida y aplica los datos. Si el precio cambió devuelve el precio anterior
// (para registrarlo en el historial).
func (p *Producto) Modificar(d DatosProducto, ahora time.Time) (precioAnterior *dinero.Dinero, err error) {
	var v errores.Validacion

	nombre := LimpiarTexto(d.Nombre)
	v.Si(nombre == "", "nombre", "El nombre es obligatorio")
	v.Si(utf8.RuneCountInString(nombre) > 255, "nombre", "Máximo 255 caracteres")

	sku := strings.TrimSpace(d.SKU)
	v.Si(sku == "", "sku", "El SKU es obligatorio")
	v.Si(utf8.RuneCountInString(sku) > 40, "sku", "Máximo 40 caracteres")
	v.Si(strings.ContainsFunc(sku, unicode.IsSpace), "sku", "No puede tener espacios")

	codigo := strings.TrimSpace(d.CodigoBarras)
	v.Si(utf8.RuneCountInString(codigo) > 64, "codigoBarras", "Máximo 64 caracteres")
	v.Si(strings.ContainsFunc(codigo, unicode.IsSpace), "codigoBarras", "No puede tener espacios")

	v.Si(!d.PrecioVenta.EsPositivo(), "precioVenta", "El precio debe ser mayor a cero")
	v.Si(d.Costo != nil && d.Costo.EsNegativo(), "costo", "El costo no puede ser negativo")
	v.Si(d.StockMinimo < 0, "stockMinimo", "No puede ser negativo")

	categorias := sinDuplicados(d.CategoriaIDs)

	switch p.Tipo {
	case TipoSimple:
		v.Si(len(d.Items) > 0, "items", "Solo los combos tienen componentes")
	case TipoCombo:
		if err := validarItemsCombo(p.ID, d.Items); err != nil {
			return nil, err
		}
	}
	if err := v.Err(); err != nil {
		return nil, err
	}

	if p.Version > 0 && p.PrecioVenta != d.PrecioVenta {
		anterior := p.PrecioVenta
		precioAnterior = &anterior
	}

	p.SKU = sku
	p.CodigoBarras = textoOpcional(codigo)
	p.Nombre = nombre
	p.Descripcion = textoOpcional(strings.TrimSpace(d.Descripcion))
	p.EmpresaID = textoOpcional(strings.TrimSpace(d.EmpresaID))
	p.PrecioVenta = d.PrecioVenta
	p.Costo = d.Costo
	p.StockMinimo = d.StockMinimo
	p.MostrarEnTienda = d.MostrarEnTienda
	p.CategoriaIDs = categorias
	if p.Tipo == TipoCombo {
		p.Items = append([]ComboItem(nil), d.Items...)
	}
	p.UpdatedAt = ahora
	p.Version++
	return precioAnterior, nil
}

func validarItemsCombo(comboID string, items []ComboItem) error {
	if len(items) == 0 {
		return ErrComboInvalido("El combo necesita al menos un producto")
	}
	vistos := map[string]bool{}
	unidades := 0
	for _, it := range items {
		switch {
		case it.ProductoID == "":
			return ErrComboInvalido("Hay un componente sin producto")
		case it.ProductoID == comboID:
			return ErrComboInvalido("Un combo no puede contenerse a sí mismo")
		case it.Cantidad <= 0:
			return ErrComboInvalido("Las cantidades deben ser mayores a cero")
		case vistos[it.ProductoID]:
			return ErrComboInvalido("Hay productos repetidos: sumá la cantidad en una sola línea")
		}
		vistos[it.ProductoID] = true
		unidades += it.Cantidad
	}
	if unidades < 2 {
		return ErrComboInvalido("Un combo debe tener al menos 2 unidades en total")
	}
	return nil
}

// CambiarPrecio aplica un precio nuevo (cascada) y devuelve el anterior.
func (p *Producto) CambiarPrecio(nuevo dinero.Dinero, ahora time.Time) (dinero.Dinero, error) {
	if !nuevo.EsPositivo() {
		return 0, errores.Invalido("precio_resultante_invalido",
			fmt.Sprintf("El precio de %q quedaría en %s", p.Nombre, nuevo))
	}
	anterior := p.PrecioVenta
	p.PrecioVenta, p.UpdatedAt = nuevo, ahora
	p.Version++
	return anterior, nil
}

// CambiarEstado activa o desactiva (baja lógica: se conserva para el historial de ventas).
func (p *Producto) CambiarEstado(activo bool, ahora time.Time) {
	if p.Activo == activo {
		return
	}
	p.Activo, p.UpdatedAt = activo, ahora
	p.Version++
}

func textoOpcional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func sinDuplicados(ids []string) []string {
	out := make([]string, 0, len(ids))
	visto := map[string]bool{}
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" && !visto[id] {
			visto[id] = true
			out = append(out, id)
		}
	}
	return out
}
