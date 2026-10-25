// Package auth: Acceso único de la dueña para la versión web y móvil.
//
// Capas (arquitectura hexagonal):
//   - domain:      entidades y reglas de negocio puras. No importa nada externo.
//   - application: casos de uso y puertos (interfaces) que el dominio necesita.
//   - adapters:    implementaciones de los puertos (http, mysql, sqlite).
package auth
