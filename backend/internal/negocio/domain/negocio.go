// Package domain contiene los datos del negocio: nombre, logo y configuración del ticket (US-003).
package domain

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"magui/internal/kernel/errores"
	"magui/internal/kernel/imagen"
)

// Ticket configura el comprobante que se imprime por cada venta (no es factura).
type Ticket struct {
	Habilitado       bool
	ImprimirAlCobrar bool
	AnchoPapel       int // mm: 58 u 80
	MostrarLogo      bool
	Datos            *string // dirección, teléfono, CUIT... una por línea
	Pie              *string
}

// Negocio son los datos que identifican al comercio en la app y en el ticket.
type Negocio struct {
	Nombre      string
	Ticket      Ticket
	TieneLogo   bool
	LogoVersion int
}

// Anchos de papel de las ticketeras térmicas habituales.
var AnchosPapel = []int{58, 80}

func textoOpcional(s string, max int, campo string, v *errores.Validacion) *string {
	// Se respetan los saltos de línea, pero no las líneas vacías ni los espacios de más.
	var lineas []string
	for l := range strings.SplitSeq(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if l = strings.Join(strings.Fields(l), " "); l != "" {
			lineas = append(lineas, l)
		}
	}
	s = strings.Join(lineas, "\n")
	v.Si(utf8.RuneCountInString(s) > max, campo, "Máximo "+strconv.Itoa(max)+" caracteres")
	v.Si(len(lineas) > 6, campo, "Máximo 6 líneas")
	if s == "" {
		return nil
	}
	return &s
}

// Datos que puede editar la dueña.
type Cambios struct {
	Nombre string
	Ticket Ticket
	Datos  string
	Pie    string
}

// Aplicar valida y aplica los cambios de nombre y ticket.
func (n *Negocio) Aplicar(c Cambios) error {
	var v errores.Validacion
	nombre := strings.Join(strings.Fields(c.Nombre), " ")
	v.Si(nombre == "", "nombre", "Ingresá el nombre del negocio")
	v.Si(utf8.RuneCountInString(nombre) > 60, "nombre", "Máximo 60 caracteres")
	anchoValido := false
	for _, a := range AnchosPapel {
		anchoValido = anchoValido || a == c.Ticket.AnchoPapel
	}
	v.Si(!anchoValido, "ticket.anchoPapel", "Elegí 58 u 80 mm")
	datos := textoOpcional(c.Datos, 300, "ticket.datos", &v)
	pie := textoOpcional(c.Pie, 200, "ticket.pie", &v)
	if err := v.Err(); err != nil {
		return err
	}
	t := c.Ticket
	t.Datos, t.Pie = datos, pie
	if !t.Habilitado {
		t.ImprimirAlCobrar = false
	}
	n.Nombre, n.Ticket = nombre, t
	return nil
}

// TamanoMaximoLogo: suficiente para un logo nítido en pantalla y en el ticket.
const TamanoMaximoLogo = 512 << 10

// ValidarLogo revisa el archivo por su contenido real, no por lo que dice el navegador.
func ValidarLogo(datos []byte) (tipo string, err error) {
	if len(datos) == 0 {
		return "", ErrLogoVacio
	}
	if len(datos) > TamanoMaximoLogo {
		return "", ErrLogoGrande
	}
	if tipo = imagen.Detectar(datos); tipo == "" {
		return "", ErrLogoTipo
	}
	return tipo, nil
}

var (
	ErrSinLogo    = errores.NoEncontrado("sin_logo", "El negocio no tiene logo")
	ErrLogoVacio  = errores.Invalido("logo_vacio", "Elegí una imagen")
	ErrLogoGrande = errores.Invalido("logo_muy_grande", "La imagen es muy pesada: máximo 512 KB")
	ErrLogoTipo   = errores.Invalido("logo_tipo", "Usá una imagen PNG, JPG o WEBP")
)
