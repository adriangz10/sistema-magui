// Package inventario: Lotes con vencimiento y movimientos de stock. El stock es la suma de movimientos.
//
// Capas (arquitectura hexagonal):
//   - domain:      entidades y reglas de negocio puras. No importa nada externo.
//   - application: casos de uso y puertos (interfaces) que el dominio necesita.
//   - adapters:    implementaciones de los puertos (http, mysql, sqlite).
package inventario
