// Package catalogo: Productos (SKU y código de barras), categorías jerárquicas, empresas, combos y cambios de precio en cascada.
//
// Capas (arquitectura hexagonal):
//   - domain:      entidades y reglas de negocio puras. No importa nada externo.
//   - application: casos de uso y puertos (interfaces) que el dominio necesita.
//   - adapters:    implementaciones de los puertos (http, mysql, sqlite).
package catalogo
