// Package caja: Gastos y retiros de dinero de la caja que no son compra de mercadería.
//
// Capas (arquitectura hexagonal):
//   - domain:      entidades y reglas de negocio puras. No importa nada externo.
//   - application: casos de uso y puertos (interfaces) que el dominio necesita.
//   - adapters:    implementaciones de los puertos (http, mysql, sqlite).
package caja
