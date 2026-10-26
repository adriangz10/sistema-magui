package application

import (
	"time"

	"magui/internal/kernel/dinero"
)

// --- Pull: copia del catálogo ---

// ProductoCopia es lo que el escritorio necesita de un producto para venderlo sin conexión.
type ProductoCopia struct {
	ID           string        `json:"id"`
	SKU          string        `json:"sku"`
	CodigoBarras *string       `json:"codigoBarras"`
	Nombre       string        `json:"nombre"`
	Tipo         string        `json:"tipo"` // SIMPLE o COMBO
	Empresa      *string       `json:"empresa"`
	PrecioVenta  dinero.Dinero `json:"precioVenta"`
	// Stock al momento de la copia (en combos, cuántos se pueden armar).
	Stock   int  `json:"stock"`
	Activo  bool `json:"activo"`
	Version int  `json:"version"`
}

// ComponenteCopia: el combo ComboID lleva Cantidad unidades de ProductoID.
type ComponenteCopia struct {
	ComboID    string `json:"comboId"`
	ProductoID string `json:"productoId"`
	Cantidad   int    `json:"cantidad"`
}

// CopiaCatalogo es el catálogo completo en un momento dado.
type CopiaCatalogo struct {
	GeneradaEn  time.Time         `json:"generadaEn"`
	Productos   []ProductoCopia   `json:"productos"`
	Componentes []ComponenteCopia `json:"componentes"`
}

// --- Push: ventas hechas sin conexión ---

type ItemOffline struct {
	ProductoID string        `json:"productoId"`
	Cantidad   int           `json:"cantidad"`
	Precio     dinero.Dinero `json:"precio"` // el que se cobró
}

type PagoOffline struct {
	Metodo string        `json:"metodo"`
	Monto  dinero.Dinero `json:"monto"`
}

// VentaOffline es una venta cobrada en el escritorio.
type VentaOffline struct {
	ID    string        `json:"id"` // UUID generado en el escritorio
	Fecha time.Time     `json:"fecha"`
	Items []ItemOffline `json:"items"`
	Pagos []PagoOffline `json:"pagos"`
}

// EstadoEnvio es lo que pasó con cada venta enviada.
type EstadoEnvio string

const (
	Registrada EstadoEnvio = "registrada" // se guardó ahora
	YaExistia  EstadoEnvio = "ya_existia" // se había recibido antes (reenvío)
	Rechazada  EstadoEnvio = "rechazada"  // datos inválidos: reenviarla no sirve
)

// ErrorEnvio explica por qué se rechazó una venta.
type ErrorEnvio struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ResultadoEnvio struct {
	ID     string      `json:"id"`
	Estado EstadoEnvio `json:"estado"`
	Error  *ErrorEnvio `json:"error,omitempty"`
}

// EnvioVentas es el cuerpo de POST /api/sync/ventas.
type EnvioVentas struct {
	Ventas []VentaOffline `json:"ventas"`
}

// RespuestaEnvio es la respuesta de POST /api/sync/ventas, un resultado por venta y en el mismo orden.
type RespuestaEnvio struct {
	Resultados []ResultadoEnvio `json:"resultados"`
}

// MaxVentasPorEnvio limita el tamaño de cada envío (el escritorio manda de a tandas).
const MaxVentasPorEnvio = 50
