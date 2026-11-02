# US-002 Apertura, cierre y conciliación de caja

**Como** dueña del negocio (única usuaria del sistema)
**quiero** abrir la caja con el dinero inicial, cerrarla contando lo que hay y ver si sobra o falta
**para** controlar el dinero del día y tener un historial de cada caja con sus ventas.

## Decisiones del cliente (2026-11-02)
- D-1. **Sin caja abierta no se vende** desde la web ni el celular (error `caja_cerrada`). La app de escritorio
  sin conexión no puede verificarlo: sigue cobrando y esas ventas se asignan a la caja según su hora.
- D-2. **Se concilia cada medio de pago** (efectivo, transferencia, QR y tarjeta), no solo el efectivo.
- D-3. **Editar** una caja permite cambiar monto inicial, montos contados y notas. Los horarios de apertura y cierre
  no se editan. Cada edición pide un motivo y queda en el historial con el antes y el después.
- D-4. La caja es de una sola usuaria (la dueña): no se registra "quién"; hay como máximo **una caja abierta**.

## Reglas de negocio
- RN-1. Ventas de una caja = ventas cuya hora está entre la apertura (inclusive) y el cierre (exclusive).
  Si la caja sigue abierta, hasta ahora.
- RN-2. Para cada medio: **esperado** = ventas confirmadas (lo cobrado con ese medio) − gastos no anulados
  hechos con ese medio en el período. En efectivo además se suma el **monto inicial**.
- RN-3. **Diferencia** = contado − esperado. Positiva = sobrante, negativa = faltante, cero = cuadra.
- RN-4. Al cerrar se guarda la foto del arqueo (ventas, gastos, esperado y contado por medio).
  Al editar se vuelve a calcular el esperado con los datos de ese momento (ej. una venta anulada después).
- RN-5. Montos ≥ 0. Al cerrar hay que informar el contado de los cuatro medios.
- RN-6. Edición con control de versión: si la caja cambió mientras se editaba, se rechaza (`version_desactualizada`).

## Criterios de aceptación
```gherkin
Escenario: Abrir la caja
  Dado que no hay una caja abierta
  Cuando abro la caja con $10.000 de monto inicial
  Entonces queda una caja abierta desde ahora con monto inicial $10.000

Escenario: No se puede abrir dos cajas
  Dado que hay una caja abierta
  Cuando intento abrir otra
  Entonces veo "Ya hay una caja abierta"

Escenario: Vender sin caja abierta
  Dado que no hay una caja abierta
  Cuando cobro una venta desde la web
  Entonces la venta se rechaza con "La caja está cerrada"

Escenario: Cerrar con faltante
  Dado una caja abierta con $10.000 iniciales, $5.000 en ventas en efectivo y $1.000 de gastos en efectivo
  Cuando la cierro informando $13.500 en efectivo
  Entonces el esperado en efectivo es $14.000 y hay un faltante de $500

Escenario: Editar una caja cerrada
  Dado una caja cerrada
  Cuando cambio el efectivo contado e indico el motivo
  Entonces se recalcula la diferencia y el historial muestra el valor anterior, el nuevo y el motivo

Escenario: Ver las ventas de una caja cerrada
  Dado una caja cerrada en el historial
  Cuando elijo "Ver ventas"
  Entonces veo las ventas hechas entre su apertura y su cierre
```

## API
| Método | Ruta | Uso |
|---|---|---|
| GET | `/api/caja/sesiones?page=` | Historial de cajas (más reciente primero) con totales |
| GET | `/api/caja/sesiones/actual` | Caja abierta con arqueo en vivo, o `null` |
| GET | `/api/caja/sesiones/{id}` | Detalle: arqueo por medio + historial de ediciones |
| POST | `/api/caja/sesiones` | Abrir `{montoInicial, notas?}` |
| POST | `/api/caja/sesiones/{id}/cerrar` | Cerrar `{version, contado: {EFECTIVO, TRANSFERENCIA, QR, TARJETA}, notas?}` |
| PATCH | `/api/caja/sesiones/{id}` | Editar `{version, motivo, montoInicial, contado?, notas}` |
| GET | `/api/ventas?desde=<ISO>&hasta=<ISO>` | Ventas de la caja (ahora `desde`/`hasta` aceptan también fecha y hora ISO) |

## Datos
Migración `00007_caja_sesion`: tablas `caja_sesion`, `caja_arqueo` (una fila por medio al cerrar) y
`caja_edicion` (auditoría con `cambios` JSON, mismo criterio que ADR 0005).

## Preguntas abiertas
- ¿La app de escritorio debería impedir cobrar cuando tiene conexión y la caja está cerrada? Hoy no lo verifica.
- ¿Se quiere imprimir o exportar el cierre de caja (ticket / PDF)?
