package domain

import "strings"

// Origen indica desde qué versión del sistema se hizo una edición.
type Origen string

const (
	OrigenWeb     Origen = "WEB"
	OrigenMovil   Origen = "MOVIL"
	OrigenDesktop Origen = "DESKTOP"
)

// ParseOrigen normaliza el origen (vacío = WEB).
func ParseOrigen(s string) (Origen, bool) {
	o := Origen(strings.ToUpper(strings.TrimSpace(s)))
	switch o {
	case "":
		return OrigenWeb, true
	case OrigenWeb, OrigenMovil, OrigenDesktop:
		return o, true
	}
	return "", false
}
