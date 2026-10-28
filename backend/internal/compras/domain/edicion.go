package domain

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/reloj"
)

// MaxMotivo es el largo máximo del motivo de una edición.
const MaxMotivo = 255

// VersionDocumento es la versión del formato del documento de cambios (ADR 0005).
const VersionDocumento = 1

// CambiosCabecera es lo que se quiere dejar en la compra (US-001a). La fecha llega como
// día (AAAA-MM-DD) o como instante: exactamente uno de los dos.
type CambiosCabecera struct {
	Version       int
	Dia           *fecha.Fecha
	Instante      *time.Time
	EmpresaID     string // "" = sin empresa
	EmpresaNombre string // nombre de EmpresaID, lo completa el caso de uso si la empresa cambió
	Notas         string
	Origen        string // "" = WEB
	Motivo        string
	// Renglones es la lista completa de renglones pedida (US-001b); nil = los renglones no cambian.
	// Un Vencimiento nil significa "sin vencimiento" (no "sin cambio").
	Renglones *[]RenglonEdicion
	// NombresProductos (id -> nombre) los completa el caso de uso para el documento de cambios.
	NombresProductos map[string]string
}

// RenglonEdicion es un renglón del pedido de edición.
type RenglonEdicion struct {
	ID               string // "" = renglón nuevo (US-001c, todavía no disponible)
	ProductoID       string
	Cantidad         int
	CostoUnitario    dinero.Dinero
	PrecioVentaNuevo *dinero.Dinero
	Vencimiento      *fecha.Fecha
}

// RenglonModificado es un renglón existente que cambió.
type RenglonModificado struct {
	Indice       int  // posición en el `items` enviado (para mensajes por renglón)
	Item         Item // valores nuevos
	Anterior     Item
	Producto     string
	Diferencia   int // cantidad nueva - anterior
	CambioCosto  bool
	CambioVenc   bool
	CambioPrecio bool
}

// LoteRenglon es el lote de un renglón con su stock actual (lectura bloqueante).
type LoteRenglon struct {
	LoteID string
	Stock  int
}

// CambioCampo es un campo que cambió, con su valor anterior y el nuevo (null = vacío).
// AntesID/DespuesID solo se usan en `empresa`; ausentes equivalen a null.
type CambioCampo struct {
	Campo     string  `json:"campo"`
	Antes     any     `json:"antes"`
	Despues   any     `json:"despues"`
	AntesID   *string `json:"antesId,omitempty"`
	DespuesID *string `json:"despuesId,omitempty"`
}

// CambioRenglon son los cambios de un renglón (se completa en US-001b/c).
type CambioRenglon struct {
	Accion     string        `json:"accion"` // MODIFICADO, AGREGADO, QUITADO
	ItemID     string        `json:"itemId"`
	ProductoID string        `json:"productoId"`
	Producto   string        `json:"producto"`
	Cambios    []CambioCampo `json:"cambios"`
}

// Cambios es el documento de auditoría v1 (ADR 0005).
type Cambios struct {
	V         int             `json:"v"`
	Cabecera  []CambioCampo   `json:"cabecera"`
	Renglones []CambioRenglon `json:"renglones"`
}

// Edicion es una entrada del historial de ediciones de una compra.
type Edicion struct {
	ID       string
	CompraID string
	Fecha    time.Time
	Origen   Origen
	Motivo   *string
	Version  int // versión de la compra que dejó esta edición
	Cambios  Cambios
}

// MarshalJSON de Cambios nunca emite listas null.
func (c Cambios) MarshalJSON() ([]byte, error) {
	type plano Cambios
	if c.Cabecera == nil {
		c.Cabecera = []CambioCampo{}
	}
	if c.Renglones == nil {
		c.Renglones = []CambioRenglon{}
	}
	return json.Marshal(plano(c))
}

// LeerCambios decodifica el documento guardado; falla en voz alta con una versión desconocida.
func LeerCambios(b []byte) (Cambios, error) {
	var c Cambios
	if err := json.Unmarshal(b, &c); err != nil {
		return Cambios{}, fmt.Errorf("documento de cambios inválido: %w", err)
	}
	if c.V != VersionDocumento {
		return Cambios{}, fmt.Errorf("documento de cambios con versión %d no soportada", c.V)
	}
	if c.Cabecera == nil {
		c.Cabecera = []CambioCampo{}
	}
	if c.Renglones == nil {
		c.Renglones = []CambioRenglon{}
	}
	return c, nil
}

// PlanEdicion es el resultado de aplicar los cambios a la compra actual.
type PlanEdicion struct {
	SinCambios bool
	Compra     Compra // cabecera nueva, con Version+1 y UpdatedAt = ahora
	Origen     Origen
	Motivo     *string
	Cambios    Cambios
	// Modificados son los renglones existentes que cambiaron, en el orden de la compra.
	Modificados []RenglonModificado
}

// ValidarEdicion revisa lo que no necesita la base: versión, fecha, notas, motivo y origen.
// Junta todos los problemas en un solo error de campos.
func ValidarEdicion(c CambiosCabecera, ahora time.Time) error {
	var v errores.Validacion
	v.Si(c.Version < 1, "version", "Falta la versión de la compra que se está editando")
	if _, ok := ParseOrigen(c.Origen); !ok {
		v.Agregar("origen", "Use WEB, MOVIL o DESKTOP")
	}
	v.Si(utf8.RuneCountInString(strings.TrimSpace(c.Motivo)) > MaxMotivo, "motivo", "Máximo 255 caracteres")
	v.Si(utf8.RuneCountInString(strings.TrimSpace(c.Notas)) > MaxNotas, "notas", "Máximo 500 caracteres")
	switch {
	case c.Dia == nil && c.Instante == nil:
		v.Agregar("fecha", "Indicá la fecha")
	case c.Dia != nil && c.Dia.IsZero():
		v.Agregar("fecha", "Indicá la fecha")
	case c.Dia != nil && c.Dia.Despues(fecha.DelInstante(ahora, reloj.ZonaArgentina)):
		v.Agregar("fecha", "La fecha no puede ser futura")
	case c.Instante != nil && c.Instante.After(ahora.Add(time.Hour)):
		v.Agregar("fecha", "La fecha no puede ser futura")
	}
	if err := v.Err(); err != nil {
		return err
	}
	if c.Renglones != nil {
		return validarRenglones(*c.Renglones)
	}
	return nil
}

// validarRenglones revisa lo que se puede saber sin la compra: valores de cada renglón, ids
// repetidos y renglones nuevos (todavía no disponibles).
func validarRenglones(rs []RenglonEdicion) error {
	var v errores.Validacion
	items := make([]Item, len(rs))
	vistos := map[string]bool{}
	nuevos := false
	for i := range rs {
		// "vencimiento": "" llega como fecha cero: es "sin vencimiento" (se normaliza en el pedido
		// mismo para que no cuente como un cambio falso al compararlo con el lote).
		if rs[i].Vencimiento != nil && rs[i].Vencimiento.IsZero() {
			rs[i].Vencimiento = nil
		}
		r := rs[i]
		items[i] = Item{ID: r.ID, ProductoID: r.ProductoID, Cantidad: r.Cantidad, CostoUnitario: r.CostoUnitario,
			PrecioVentaNuevo: r.PrecioVentaNuevo, Vencimiento: r.Vencimiento}
		if strings.TrimSpace(r.ID) == "" {
			nuevos = true
		} else if vistos[r.ID] {
			v.Agregar(fmt.Sprintf("items[%d].id", i), "El renglón está repetido")
		}
		vistos[r.ID] = true
	}
	validarItems(&v, items)
	if err := v.Err(); err != nil {
		return err
	}
	if nuevos {
		return ErrRenglonesNuevosNoDisponibles
	}
	return nil
}

// PlanificarEdicion calcula la compra resultante y el documento de cambios. Es pura: no toca
// la base. Ignora origen y motivo para decidir si hubo cambios (RN-A-02).
func PlanificarEdicion(actual Compra, c CambiosCabecera, ahora time.Time) (PlanEdicion, error) {
	if err := ValidarEdicion(c, ahora); err != nil {
		return PlanEdicion{}, err
	}
	origen, _ := ParseOrigen(c.Origen)
	var motivo *string
	if m := strings.TrimSpace(c.Motivo); m != "" {
		motivo = &m
	}

	nueva := actual
	nueva.Items = actual.Items

	// Fecha: el mismo día (Argentina) conserva el instante original.
	switch {
	case c.Instante != nil:
		nueva.Fecha = c.Instante.UTC()
	case c.Dia.String() == fecha.DelInstante(actual.Fecha, reloj.ZonaArgentina).String():
		nueva.Fecha = actual.Fecha
	case c.Dia.String() == fecha.DelInstante(ahora, reloj.ZonaArgentina).String():
		nueva.Fecha = ahora.UTC()
	default:
		nueva.Fecha = c.Dia.InicioEn(reloj.ZonaArgentina).Add(12 * time.Hour).UTC()
	}

	// Empresa y notas.
	nueva.EmpresaID, nueva.Empresa = nil, ""
	if e := strings.TrimSpace(c.EmpresaID); e != "" {
		nueva.EmpresaID, nueva.Empresa = &e, c.EmpresaNombre
		if actual.EmpresaID != nil && *actual.EmpresaID == e {
			nueva.Empresa = actual.Empresa
		}
	}
	nueva.Notas = nil
	if n := strings.TrimSpace(c.Notas); n != "" {
		nueva.Notas = &n
	}

	var mods []RenglonModificado
	if c.Renglones != nil {
		var err error
		if mods, err = planificarRenglones(actual, *c.Renglones, c.NombresProductos); err != nil {
			return PlanEdicion{}, err
		}
		nueva.Items = aplicarModificados(actual.Items, mods)
		nueva.Total = dinero.Cero
		for _, it := range nueva.Items {
			nueva.Total = nueva.Total.Sumar(it.Subtotal())
		}
	}

	var cab []CambioCampo
	if !nueva.Fecha.Equal(actual.Fecha) {
		cab = append(cab, CambioCampo{Campo: "fecha", Antes: instanteISO(actual.Fecha), Despues: instanteISO(nueva.Fecha)})
	}
	if !igualPtr(actual.EmpresaID, nueva.EmpresaID) {
		cc := CambioCampo{Campo: "empresa", AntesID: actual.EmpresaID, DespuesID: nueva.EmpresaID}
		if actual.EmpresaID != nil {
			cc.Antes = actual.Empresa
		}
		if nueva.EmpresaID != nil {
			cc.Despues = nueva.Empresa
		}
		cab = append(cab, cc)
	}
	if !igualPtr(actual.Notas, nueva.Notas) {
		cab = append(cab, CambioCampo{Campo: "notas", Antes: valorPtr(actual.Notas), Despues: valorPtr(nueva.Notas)})
	}

	if len(cab) == 0 && len(mods) == 0 {
		return PlanEdicion{SinCambios: true, Compra: actual, Origen: origen, Motivo: motivo}, nil
	}
	rengs := make([]CambioRenglon, len(mods))
	for i, m := range mods {
		rengs[i] = cambioDeRenglon(m)
	}
	nueva.Version = actual.Version + 1
	nueva.UpdatedAt = ahora
	return PlanEdicion{
		Compra: nueva, Origen: origen, Motivo: motivo, Modificados: mods,
		Cambios: Cambios{V: VersionDocumento, Cabecera: cab, Renglones: rengs},
	}, nil
}

// planificarRenglones compara los renglones pedidos con los de la compra. La lista pedida debe
// traer todos los renglones existentes (quitar llega con US-001c) y ninguno nuevo.
func planificarRenglones(actual Compra, pedidos []RenglonEdicion, nombres map[string]string) ([]RenglonModificado, error) {
	var v errores.Validacion
	porID := map[string]Item{}
	for _, it := range actual.Items {
		porID[it.ID] = it
	}
	pedido := map[string]int{}
	for i, r := range pedidos {
		ant, ok := porID[r.ID]
		if !ok {
			v.Agregar(fmt.Sprintf("items[%d].id", i), "El renglón no pertenece a esta compra")
			continue
		}
		pedido[r.ID] = i
		v.Si(r.ProductoID != ant.ProductoID, fmt.Sprintf("items[%d].productoId", i),
			"No se puede cambiar el producto de un renglón: quitalo y agregá otro")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	if len(pedido) < len(actual.Items) {
		return nil, ErrQuitarRenglonesNoDisponible
	}
	var mods []RenglonModificado
	for _, ant := range actual.Items { // orden de la compra
		i := pedido[ant.ID]
		r := pedidos[i]
		nuevo := Item{ID: ant.ID, ProductoID: ant.ProductoID, Cantidad: r.Cantidad, CostoUnitario: r.CostoUnitario,
			PrecioVentaNuevo: r.PrecioVentaNuevo, Vencimiento: r.Vencimiento}
		m := RenglonModificado{Indice: i, Item: nuevo, Anterior: ant, Producto: nombres[ant.ProductoID],
			Diferencia:   nuevo.Cantidad - ant.Cantidad,
			CambioCosto:  nuevo.CostoUnitario != ant.CostoUnitario,
			CambioVenc:   !igualFecha(ant.Vencimiento, nuevo.Vencimiento),
			CambioPrecio: !igualDinero(ant.PrecioVentaNuevo, nuevo.PrecioVentaNuevo)}
		if m.Diferencia != 0 || m.CambioCosto || m.CambioVenc || m.CambioPrecio {
			mods = append(mods, m)
		}
	}
	return mods, nil
}

func aplicarModificados(items []Item, mods []RenglonModificado) []Item {
	out := slices.Clone(items)
	for _, m := range mods {
		for i := range out {
			if out[i].ID == m.Item.ID {
				out[i] = m.Item
			}
		}
	}
	return out
}

func cambioDeRenglon(m RenglonModificado) CambioRenglon {
	cs := []CambioCampo{}
	if m.Diferencia != 0 {
		cs = append(cs, CambioCampo{Campo: "cantidad", Antes: m.Anterior.Cantidad, Despues: m.Item.Cantidad})
	}
	if m.CambioCosto {
		cs = append(cs, CambioCampo{Campo: "costoUnitario", Antes: m.Anterior.CostoUnitario.String(), Despues: m.Item.CostoUnitario.String()})
	}
	if m.CambioPrecio {
		cs = append(cs, CambioCampo{Campo: "precioVentaNuevo", Antes: valorDinero(m.Anterior.PrecioVentaNuevo), Despues: valorDinero(m.Item.PrecioVentaNuevo)})
	}
	if m.CambioVenc {
		cs = append(cs, CambioCampo{Campo: "vencimiento", Antes: valorFecha(m.Anterior.Vencimiento), Despues: valorFecha(m.Item.Vencimiento)})
	}
	return CambioRenglon{Accion: "MODIFICADO", ItemID: m.Item.ID, ProductoID: m.Item.ProductoID, Producto: m.Producto, Cambios: cs}
}

// VerificarConsumo comprueba que ningún renglón quede por debajo de lo ya consumido del lote
// (consumido = cantidad anterior - stock del lote). Si hay renglones rechazados devuelve un único
// 409 compra_consumida con todos (RN-B-02). `lotes` va por id de renglón.
func VerificarConsumo(plan PlanEdicion, lotes map[string]LoteRenglon) error {
	campos := map[string]string{}
	for _, m := range plan.Modificados {
		if m.Diferencia >= 0 {
			continue
		}
		l, ok := lotes[m.Item.ID]
		if !ok {
			return ErrRenglonSinLote(m.Producto)
		}
		if l.Stock+m.Diferencia >= 0 {
			continue
		}
		consumido := m.Anterior.Cantidad - l.Stock
		campos[fmt.Sprintf("items[%d].cantidad", m.Indice)] = fmt.Sprintf(
			"%s: de %s de esta compra ya %s %d (ventas o ajustes). Solo podés bajar hasta %s.",
			m.Producto, unidades(m.Anterior.Cantidad), salieron(consumido), consumido, unidadesMin(consumido))
	}
	if len(campos) == 0 {
		return nil
	}
	e := ErrCompraConsumida
	e.Campos = campos
	return &e
}

func unidades(n int) string {
	if n == 1 {
		return "la única unidad"
	}
	return fmt.Sprintf("las %d unidades", n)
}

func unidadesMin(n int) string {
	if n == 1 {
		return "1 unidad"
	}
	return fmt.Sprintf("%d unidades", n)
}

func salieron(n int) string {
	if n == 1 {
		return "salió"
	}
	return "salieron"
}

func igualFecha(a, b *fecha.Fecha) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.String() == b.String()
}

func igualDinero(a, b *dinero.Dinero) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func valorDinero(d *dinero.Dinero) any {
	if d == nil {
		return nil
	}
	return d.String()
}

func valorFecha(f *fecha.Fecha) any {
	if f == nil {
		return nil
	}
	return f.String()
}

// instanteISO da el formato del documento de cambios: UTC con milisegundos.
func instanteISO(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

func igualPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func valorPtr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
