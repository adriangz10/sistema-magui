# Backlog priorizado

Orden = orden de entrega. Cada historia es una rebanada vertical (backend + pantalla) usable sola.

| # | Historia | Resumen | Prioridad | Estimación | Estado | Depende de |
|---|---|---|---|---|---|---|
| 1 | [US-001a](stories/US-001a-editar-cabecera-compra.md) | Editar cabecera de compra (fecha, empresa, notas) + versión + auditoría + botón "Editar" + historial de ediciones | Should | 5 | Borrador | - |
| 2 | [US-001b](stories/US-001b-editar-renglones-compra.md) | Editar renglones existentes (cantidad con bloqueo por consumo y movimiento EDICION_COMPRA, costo, vencimiento, precio) + último costo/precio/historial | Should | 8 | Borrador | US-001a |
| 3 | [US-001c](stories/US-001c-agregar-quitar-renglones-compra.md) | Agregar y quitar renglones + recálculo del último costo al quitar o cambiar fecha | Should | 5 | Borrador | US-001b |
| 4 | [US-002](stories/US-002-apertura-cierre-caja.md) | Apertura y cierre de caja: conciliación por medio de pago (sobrante/faltante), historial, edición con auditoría, ventas de cada caja y bloqueo de ventas sin caja abierta | Should | 8 | En revisión | - |
| 5 | [US-003](stories/US-003-lector-ticket-marca.md) | Lector de códigos en toda la pantalla Vender, ticket de cada venta (58/80 mm, al cobrar o con botón, reimpresión), nombre y logo del negocio | Should | 5 | En revisión | - |
| 6 | [US-004](stories/US-004-foto-producto.md) | Foto opcional de cada producto (se achica sola al subirla), visible en Productos y en el buscador de Vender | Should | 3 | En revisión | - |
| 7 | [US-005](stories/US-005-vender-con-teclado.md) | Vender sin mouse: foco en el buscador, F3–F6 para la forma de pago, Enter en el monto cobra | Should | 2 | En revisión | US-003 |

Historia paraguas con reglas y decisiones comunes: [US-001](stories/US-001-editar-compra.md).

## Ideas aún sin historia
- Anular/eliminar una compra completa (fuera de alcance de US-001, a pedido del cliente).
