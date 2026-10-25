// Package kernel es el "núcleo compartido": tipos de negocio genéricos que usan todos
// los módulos (dinero, errores, paginación, identificadores, reloj).
//
// Reglas: no depende de ningún módulo ni de infraestructura (HTTP, SQL). Solo stdlib
// y, como única excepción, la librería de UUID.
package kernel
