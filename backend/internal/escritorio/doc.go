// Package escritorio es el núcleo de la app de escritorio: cobra aunque no haya internet.
//
//   - Las ventas se cobran siempre en local, con las mismas reglas del servidor (se reutiliza el
//     caso de uso de ventas con adaptadores SQLite), y quedan en una cola.
//   - Un proceso sube la cola a /api/sync/ventas y baja la copia del catálogo de /api/sync/catalogo.
//   - Las demás pantallas hablan directo con el servidor (el handler local reenvía /api/*);
//     sin conexión responden 503 "sin_conexion".
//
// Capas: application (sincronización, estado, búsqueda local, ingreso sin conexión) y adapters
// (sqlite: base local; remoto: cliente del servidor; http: handler local).
package escritorio
