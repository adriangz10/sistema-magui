# Backlog priorizado

Orden = orden de entrega. Cada historia es una rebanada vertical (backend + pantalla) usable sola.

| # | Historia | Resumen | Prioridad | Estimación | Estado | Depende de |
|---|---|---|---|---|---|---|
| 1 | [US-001a](stories/US-001a-editar-cabecera-compra.md) | Editar cabecera de compra (fecha, empresa, notas) + versión + auditoría + botón "Editar" + historial de ediciones | Should | 5 | Borrador | - |
| 2 | [US-001b](stories/US-001b-editar-renglones-compra.md) | Editar renglones existentes (cantidad con bloqueo por consumo y movimiento EDICION_COMPRA, costo, vencimiento, precio) + último costo/precio/historial | Should | 8 | Borrador | US-001a |
| 3 | [US-001c](stories/US-001c-agregar-quitar-renglones-compra.md) | Agregar y quitar renglones + recálculo del último costo al quitar o cambiar fecha | Should | 5 | Borrador | US-001b |

Historia paraguas con reglas y decisiones comunes: [US-001](stories/US-001-editar-compra.md).

## Ideas aún sin historia
- Anular/eliminar una compra completa (fuera de alcance de US-001, a pedido del cliente).
