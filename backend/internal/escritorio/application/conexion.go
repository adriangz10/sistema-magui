package application

import (
	"sync"
	"time"
)

// Conexion recuerda si el servidor responde. La actualizan el reenvío de pedidos y la sincronización.
type Conexion struct {
	mu       sync.Mutex
	enLinea  bool
	cambio   time.Time
	alVolver func() // se llama al pasar de "sin conexión" a "en línea"
}

// NuevaConexion arranca suponiendo que hay conexión (el primer pedido lo confirma).
func NuevaConexion(alVolver func()) *Conexion {
	return &Conexion{enLinea: true, alVolver: alVolver}
}

func (c *Conexion) EnLinea() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enLinea
}

// SinRespuestaReciente: hace menos de "ventana" que se perdió la conexión. Sirve para no esperar el
// tiempo de espera del servidor en cada búsqueda mientras no hay señal.
func (c *Conexion) SinRespuestaReciente(ahora time.Time, ventana time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.enLinea && ahora.Sub(c.cambio) < ventana
}

func (c *Conexion) Informar(hay bool, ahora time.Time) {
	c.mu.Lock()
	volvio := hay && !c.enLinea
	if hay != c.enLinea || !hay {
		c.cambio = ahora // sin conexión: cada falla renueva la ventana
	}
	c.enLinea = hay
	c.mu.Unlock()
	if volvio && c.alVolver != nil {
		c.alVolver()
	}
}
