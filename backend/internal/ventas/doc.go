// Package ventas: Ventas del punto de venta con pagos combinados (efectivo, transferencia, QR, tarjeta) y anulaciones.
//
// Capas (arquitectura hexagonal):
//   - domain:      entidades y reglas de negocio puras. No importa nada externo.
//   - application: casos de uso y puertos (interfaces) que el dominio necesita.
//   - adapters:    implementaciones de los puertos (http, mysql, sqlite).
package ventas
