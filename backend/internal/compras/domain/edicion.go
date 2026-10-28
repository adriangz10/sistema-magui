package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

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
	ConRenglones  bool // el pedido trae `items` (todavía no se admiten)
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
	if c.ConRenglones {
		return ErrRenglonesNoEditables
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

	if len(cab) == 0 {
		return PlanEdicion{SinCambios: true, Compra: actual, Origen: origen, Motivo: motivo}, nil
	}
	nueva.Version = actual.Version + 1
	nueva.UpdatedAt = ahora
	return PlanEdicion{
		Compra: nueva, Origen: origen, Motivo: motivo,
		Cambios: Cambios{V: VersionDocumento, Cabecera: cab, Renglones: []CambioRenglon{}},
	}, nil
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
