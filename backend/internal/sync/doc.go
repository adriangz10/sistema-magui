// Package sync sincroniza el escritorio con el servidor central.
//
// El escritorio vende sin conexión con una copia del catálogo y guarda cada venta en una
// cola local. Al volver la conexión:
//   - pull: descarga la copia completa del catálogo (GET /api/sync/catalogo);
//   - push: envía las ventas pendientes (POST /api/sync/ventas). El servidor las registra
//     con el ID generado en el escritorio, así que reenviar una venta no la duplica.
//
// Capas: application (casos de uso, modelos del contrato y puertos) y adapters
// (enlaces con catálogo y ventas; http).
package sync
