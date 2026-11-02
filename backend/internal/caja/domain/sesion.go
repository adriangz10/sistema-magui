package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"magui/internal/kernel/dinero"
	"magui/internal/kernel/errores"
)

// Medios que se concilian al cerrar la caja, en el orden en que se muestran (US-002, D-2).
var Medios = []Metodo{Efectivo, Transferencia, QR, Tarjeta}

// Movimientos de un medio de pago en el período de la caja.
type Movimientos struct {
	Ventas dinero.Dinero // cobrado en ventas confirmadas
	Gastos dinero.Dinero // gastos no anulados pagados con ese medio
}

// Totales del período de una caja, por medio de pago.
type Totales map[Metodo]Movimientos

// Arqueo de un medio de pago: lo que dice el sistema contra lo que contó la dueña.
type Arqueo struct {
	Metodo   Metodo
	Ventas   dinero.Dinero
	Gastos   dinero.Dinero
	Esperado dinero.Dinero
	Contado  *dinero.Dinero // nil mientras la caja está abierta
}

// Diferencia = contado − esperado (positiva sobra, negativa falta). nil si todavía no se contó.
func (a Arqueo) Diferencia() *dinero.Dinero {
	if a.Contado == nil {
		return nil
	}
	d := a.Contado.Restar(a.Esperado)
	return &d
}

// CalcularArqueo arma el arqueo de los cuatro medios (RN-2). contado puede ser nil (caja abierta).
func CalcularArqueo(inicial dinero.Dinero, t Totales, contado map[Metodo]dinero.Dinero) []Arqueo {
	out := make([]Arqueo, len(Medios))
	for i, m := range Medios {
		mov := t[m]
		esperado := mov.Ventas.Restar(mov.Gastos)
		if m == Efectivo {
			esperado = esperado.Sumar(inicial)
		}
		a := Arqueo{Metodo: m, Ventas: mov.Ventas, Gastos: mov.Gastos, Esperado: esperado}
		if c, ok := contado[m]; ok {
			a.Contado = &c
		}
		out[i] = a
	}
	return out
}

// Sesion es una caja: desde que se abre hasta que se cierra. Hay como máximo una abierta.
type Sesion struct {
	ID           string
	AbiertaEn    time.Time
	CerradaEn    *time.Time
	MontoInicial dinero.Dinero
	Notas        *string
	Arqueo       []Arqueo // foto guardada al cerrar; vacío mientras está abierta
	// CantidadVentas confirmadas en el período (se guarda al cerrar y al editar).
	CantidadVentas int
	Version        int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Abierta indica si la caja sigue abierta.
func (s Sesion) Abierta() bool { return s.CerradaEn == nil }

// Contado devuelve lo contado por medio (nil si está abierta).
func (s Sesion) Contado() map[Metodo]dinero.Dinero {
	if s.Abierta() {
		return nil
	}
	out := make(map[Metodo]dinero.Dinero, len(s.Arqueo))
	for _, a := range s.Arqueo {
		if a.Contado != nil {
			out[a.Metodo] = *a.Contado
		}
	}
	return out
}

const maxNotas = 500

func normalizarNotas(notas string, v *errores.Validacion) *string {
	notas = strings.TrimSpace(notas)
	v.Si(utf8.RuneCountInString(notas) > maxNotas, "notas", "Máximo 500 caracteres")
	if notas == "" {
		return nil
	}
	return &notas
}

// AbrirSesion valida y crea una caja abierta desde ahora.
func AbrirSesion(id string, inicial dinero.Dinero, notas string, ahora time.Time) (Sesion, error) {
	var v errores.Validacion
	v.Si(inicial.EsNegativo(), "montoInicial", "No puede ser negativo")
	n := normalizarNotas(notas, &v)
	if err := v.Err(); err != nil {
		return Sesion{}, err
	}
	ahora = aMilisegundos(ahora)
	return Sesion{ID: id, AbiertaEn: ahora, MontoInicial: inicial, Notas: n, Version: 1, CreatedAt: ahora, UpdatedAt: ahora}, nil
}

func validarContado(contado map[Metodo]dinero.Dinero, v *errores.Validacion) {
	for _, m := range Medios {
		c, ok := contado[m]
		v.Si(!ok, "contado."+string(m), "Ingresá cuánto hay (puede ser 0)")
		v.Si(ok && c.EsNegativo(), "contado."+string(m), "No puede ser negativo")
	}
}

// Cerrar guarda lo contado y la foto del arqueo calculada con los totales del período (RN-4).
func (s *Sesion) Cerrar(t Totales, contado map[Metodo]dinero.Dinero, notas string, ahora time.Time) error {
	if !s.Abierta() {
		return ErrCajaYaCerrada
	}
	var v errores.Validacion
	validarContado(contado, &v)
	n := normalizarNotas(notas, &v)
	if err := v.Err(); err != nil {
		return err
	}
	ahora = aMilisegundos(ahora)
	if ahora.Before(s.AbiertaEn) {
		ahora = s.AbiertaEn
	}
	s.CerradaEn = &ahora
	s.Notas = n
	s.Arqueo = CalcularArqueo(s.MontoInicial, t, contado)
	s.Version++
	s.UpdatedAt = ahora
	return nil
}

// Edicion es lo que se puede cambiar de una caja (D-3). Contado se ignora si la caja está abierta.
type Edicion struct {
	MontoInicial dinero.Dinero
	Contado      map[Metodo]dinero.Dinero
	Notas        string
	Motivo       string
}

// Cambio de un campo, para el historial. Los montos van como texto "0.00".
type Cambio struct {
	Campo   string  `json:"campo"`
	Antes   *string `json:"antes"`
	Despues *string `json:"despues"`
}

func texto(d dinero.Dinero) *string { s := d.String(); return &s }

func agregarSiCambia(out []Cambio, campo string, antes, despues *string) []Cambio {
	if (antes == nil) == (despues == nil) && (antes == nil || *antes == *despues) {
		return out
	}
	return append(out, Cambio{Campo: campo, Antes: antes, Despues: despues})
}

// Editar aplica los cambios y, si la caja está cerrada, recalcula el arqueo con los totales actuales.
// Devuelve la lista de campos que cambiaron (incluido el esperado recalculado).
func (s *Sesion) Editar(e Edicion, t Totales, ahora time.Time) ([]Cambio, error) {
	var v errores.Validacion
	motivo := strings.Join(strings.Fields(e.Motivo), " ")
	v.Si(motivo == "", "motivo", "Contá por qué se edita la caja")
	v.Si(utf8.RuneCountInString(motivo) > 255, "motivo", "Máximo 255 caracteres")
	v.Si(e.MontoInicial.EsNegativo(), "montoInicial", "No puede ser negativo")
	if !s.Abierta() {
		validarContado(e.Contado, &v)
	}
	notas := normalizarNotas(e.Notas, &v)
	if err := v.Err(); err != nil {
		return nil, err
	}

	var cambios []Cambio
	cambios = agregarSiCambia(cambios, "montoInicial", texto(s.MontoInicial), texto(e.MontoInicial))
	cambios = agregarSiCambia(cambios, "notas", s.Notas, notas)
	var arqueo []Arqueo
	if !s.Abierta() {
		arqueo = CalcularArqueo(e.MontoInicial, t, e.Contado)
		for i, nuevo := range arqueo {
			viejo := s.Arqueo[i]
			m := string(nuevo.Metodo)
			cambios = agregarSiCambia(cambios, "contado."+m, textoPtr(viejo.Contado), textoPtr(nuevo.Contado))
			cambios = agregarSiCambia(cambios, "esperado."+m, texto(viejo.Esperado), texto(nuevo.Esperado))
		}
	}
	// Lo que cambió solo por recálculo no alcanza: tiene que haber un cambio pedido por la dueña.
	pedido := false
	for _, c := range cambios {
		if !strings.HasPrefix(c.Campo, "esperado.") {
			pedido = true
		}
	}
	if !pedido {
		return nil, ErrSinCambios
	}

	s.MontoInicial = e.MontoInicial
	s.Notas = notas
	if arqueo != nil {
		s.Arqueo = arqueo
	}
	s.Version++
	s.UpdatedAt = ahora
	return cambios, nil
}

var (
	ErrCajaNoEncontrada      = errores.NoEncontrado("caja_no_encontrada", "La caja no existe")
	ErrCajaYaAbierta         = errores.Conflicto("caja_ya_abierta", "Ya hay una caja abierta. Cerrala antes de abrir otra.")
	ErrCajaYaCerrada         = errores.Conflicto("caja_ya_cerrada", "La caja ya estaba cerrada")
	ErrCajaCerrada           = errores.Conflicto("caja_cerrada", "La caja está cerrada. Abrila desde Caja para poder vender.")
	ErrVersionDesactualizada = errores.Conflicto("version_desactualizada", "La caja cambió mientras la editabas. Volvé a abrirla.")
	ErrSinCambios            = errores.Invalido("sin_cambios", "No hay cambios para guardar")
)

func textoPtr(d *dinero.Dinero) *string {
	if d == nil {
		return nil
	}
	return texto(*d)
}

// aMilisegundos: la base guarda milisegundos; así el período usado en el arqueo es exactamente
// el que se guarda y el que después se usa para listar las ventas de la caja.
func aMilisegundos(t time.Time) time.Time { return t.UTC().Truncate(time.Millisecond) }
