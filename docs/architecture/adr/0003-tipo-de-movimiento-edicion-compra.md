# 0003. Registrar los cambios de cantidad de una compra con un tipo de movimiento nuevo `EDICION_COMPRA`

- Estado: Propuesto
- Fecha: 2026-10-27
- Decisores: Software Architect; decisión de negocio D-3 (cliente)
- Historias relacionadas: US-001b, US-001c

## Contexto
El stock es la suma de `movimiento_stock` y nunca se edita (RN-EC-05). Al subir, bajar o quitar un renglón hace falta un
movimiento por la diferencia sobre el mismo lote. El cliente pidió un tipo propio (D-3). El tipo es `ENUM` en MySQL,
`TEXT + CHECK` en SQLite, `invdomain.TipoMovimiento` en Go y `TipoMovimiento` en el frontend.

## Alternativas consideradas
### A. Tipo nuevo `EDICION_COMPRA` (positivo o negativo)
Pros: se filtra y se explica solo en "Movimientos"; no se confunde con ajustes manuales. Contras: migración del enum
(en SQLite implica reconstruir la tabla, ver abajo).
### B. Reusar `AJUSTE` con nota "Edición de compra…"
Sin migración, pero mezcla correcciones de compra con roturas/conteos; descartado por el cliente (D-3).
### C. Reusar `COMPRA` (positivo y negativo)
Un `COMPRA` negativo es confuso y rompería cualquier reporte futuro de "unidades compradas" por movimientos.
### D. Tabla de referencia de tipos en lugar de enum/CHECK
Más flexible a futuro, pero es un cambio de modelo grande para un valor nuevo. No se justifica hoy.

## Decisión
Elegimos **A**.
- Valor: `EDICION_COMPRA`. Signo: + si sube la cantidad, − si baja o se quita el renglón. Siempre con `lote_id` del
  renglón, `referencia_id` = id de la compra, `nota` = "Edición de compra del dd/mm/aaaa" (fecha de la compra ya
  editada, hora Argentina), `fecha` = ahora (RN-EC-22).
- Renglón agregado (RN-EC-10) usa `COMPRA`, igual que al registrar.
- MySQL: `ALTER TABLE movimiento_stock MODIFY COLUMN tipo ENUM(…, 'EDICION_COMPRA') NOT NULL, ALGORITHM=INSTANT`
  (agregar al final de un ENUM es solo metadatos en 8.4; `ALGORITHM=INSTANT` hace fallar la migración si no lo fuera,
  en lugar de copiar la tabla en silencio).
- SQLite: no se puede cambiar un CHECK con ALTER; se reconstruye `movimiento_stock` (crear tabla nueva, copiar,
  borrar, renombrar, recrear índices y las vistas `v_stock_lote`/`v_stock_producto`). Nada referencia a
  `movimiento_stock` por FK, así que no hace falta apagar `foreign_keys`. SQL completo en `docs/data/editar-compra.md`.
- Go: `MovEdicionCompra TipoMovimiento = "EDICION_COMPRA"` y `Valido()` lo acepta (habilita el filtro
  `GET /inventario/movimientos?tipo=EDICION_COMPRA`).
- Frontend: `TipoMovimiento` y `TIPOS_MOVIMIENTO` en `core/api/inventario.api.ts` con la etiqueta "Edición de compra".
- `RevertirSalidas` (anulación de ventas) no se ve afectado: busca por `tipo = VENTA`.

## Consecuencias
- Positivas: trazabilidad clara; ningún movimiento existente se modifica.
- Negativas: el rollback (Down) del enum no puede borrar el valor si hay filas: el Down convierte esas filas a `AJUSTE`
  con la nota prefijada `[EDICION_COMPRA]` (el stock no cambia) y después achica el enum.
- Qué hay que hacer ahora: migración 00005 en `mysql/` y `sqlite/` (parte b), constante en dominio, etiqueta en frontend,
  `docs/API.md` (lista de tipos).
- Cuándo revisarla: si se agregan más tipos seguido (ej. anulación de compra), evaluar tabla de referencia (opción D).
