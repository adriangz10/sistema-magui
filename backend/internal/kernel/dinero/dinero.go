// Package dinero representa montos en pesos sin errores de redondeo.
// Internamente son centavos (int64): 59200.00 -> 5920000.
package dinero

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Dinero es un monto en centavos.
type Dinero int64

// Cero es el monto nulo.
const Cero Dinero = 0

// ErrFormato indica que el texto no es un monto válido.
var ErrFormato = errors.New("monto inválido")

// DesdeCentavos crea un monto a partir de centavos.
func DesdeCentavos(c int64) Dinero { return Dinero(c) }

// DesdePesos crea un monto a partir de pesos enteros.
func DesdePesos(p int64) Dinero { return Dinero(p * 100) }

// Parse interpreta "59200", "59200.5", "59200.50", "-10.25" o "59200,50".
// Rechaza más de 2 decimales y separadores de miles.
func Parse(s string) (Dinero, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrFormato
	}
	s = strings.Replace(s, ",", ".", 1)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(strings.TrimPrefix(s, "-"), "+")

	entero, frac, _ := strings.Cut(s, ".")
	if entero == "" || len(frac) > 2 || !soloDigitos(entero) || !soloDigitos(frac) {
		return 0, fmt.Errorf("%w: %q", ErrFormato, s)
	}
	for len(frac) < 2 {
		frac += "0"
	}
	pesos, err := strconv.ParseInt(entero, 10, 64)
	if err != nil || pesos > math.MaxInt64/100-1 {
		return 0, fmt.Errorf("%w: fuera de rango", ErrFormato)
	}
	cent, _ := strconv.ParseInt(frac, 10, 64)
	v := pesos*100 + cent
	if neg {
		v = -v
	}
	return Dinero(v), nil
}

func soloDigitos(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// MustParse es Parse para constantes en tests; entra en pánico si el texto es inválido.
func MustParse(s string) Dinero {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Centavos devuelve el valor en centavos (para SQLite).
func (d Dinero) Centavos() int64 { return int64(d) }

// String devuelve "59200.00" (formato para MySQL DECIMAL y JSON).
func (d Dinero) String() string {
	v := int64(d)
	signo := ""
	if v < 0 {
		signo, v = "-", -v
	}
	return fmt.Sprintf("%s%d.%02d", signo, v/100, v%100)
}

func (d Dinero) Sumar(o Dinero) Dinero        { return d + o }
func (d Dinero) Restar(o Dinero) Dinero       { return d - o }
func (d Dinero) Multiplicar(n int) Dinero     { return d * Dinero(n) }
func (d Dinero) EsCero() bool                 { return d == 0 }
func (d Dinero) EsNegativo() bool             { return d < 0 }
func (d Dinero) EsPositivo() bool             { return d > 0 }
func (d Dinero) MenorQue(o Dinero) bool       { return d < o }
func (d Dinero) MayorQue(o Dinero) bool       { return d > o }
func (d Dinero) MayorOIgualQue(o Dinero) bool { return d >= o }

// Sumar suma una lista de montos.
func Sumar(montos ...Dinero) Dinero {
	var t Dinero
	for _, m := range montos {
		t += m
	}
	return t
}

// AplicarPorcentaje sube (o baja, si es negativo) el monto un porcentaje expresado en
// centésimas de punto: 1050 = +10,50 %. Redondea al centavo (mitad hacia arriba).
func (d Dinero) AplicarPorcentaje(centesimasDePunto int64) Dinero {
	return Dinero(dividirRedondeando(int64(d)*(10000+centesimasDePunto), 10000))
}

// RedondearA redondea al múltiplo más cercano de paso (ej. a los $100). Paso <= 0 no cambia nada.
func (d Dinero) RedondearA(paso Dinero) Dinero {
	if paso <= 0 {
		return d
	}
	return Dinero(dividirRedondeando(int64(d), int64(paso)) * int64(paso))
}

// PorcentajeDe devuelve qué porcentaje es d de total, en centésimas de punto (2550 = 25,50 %).
// Si total es cero devuelve 0.
func (d Dinero) PorcentajeDe(total Dinero) int64 {
	if total == 0 {
		return 0
	}
	return dividirRedondeando(int64(d)*10000, int64(total))
}

// dividirRedondeando divide redondeando la mitad lejos de cero.
func dividirRedondeando(a, b int64) int64 {
	if b < 0 {
		a, b = -a, -b
	}
	if a >= 0 {
		return (a + b/2) / b
	}
	return -((-a + b/2) / b)
}

// MarshalJSON escribe el monto como texto: "59200.00".
func (d Dinero) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

// UnmarshalJSON acepta texto ("59200.50") o número (59200.5).
func (d *Dinero) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		return nil
	}
	if strings.HasPrefix(s, `"`) {
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
	}
	v, err := Parse(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// Scan lee desde la base: DECIMAL de MySQL (texto) o INTEGER en centavos de SQLite.
func (d *Dinero) Scan(src any) error {
	switch v := src.(type) {
	case int64:
		*d = Dinero(v)
		return nil
	case []byte:
		return d.scanTexto(string(v))
	case string:
		return d.scanTexto(v)
	case nil:
		*d = 0
		return nil
	default:
		return fmt.Errorf("dinero: tipo de columna no soportado %T", src)
	}
}

func (d *Dinero) scanTexto(s string) error {
	v, err := Parse(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// No implementa driver.Valuer a propósito: cada adaptador debe elegir explícitamente
// String() (MySQL DECIMAL) o Centavos() (SQLite INTEGER). Así un monto nunca se guarda
// en pesos donde se esperaban centavos sin que nadie lo note.
