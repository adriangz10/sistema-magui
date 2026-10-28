# 0002. Concurrencia en la edición de compras: versión optimista en la compra y bloqueo pesimista de lotes

- Estado: Propuesto
- Fecha: 2026-10-27
- Decisores: Software Architect (pendiente aprobación en G2)
- Historias relacionadas: US-001a (versión), US-001b (bloqueo de lotes), US-001c

## Contexto
Dos riesgos distintos:
1. **Dos personas editan la misma compra** (RN-EC-23): la segunda no debe pisar a la primera sin enterarse. El patrón del
   repo para esto es `version` + 409 `version_desactualizada` (producto, empresa, categoría:
   `UPDATE … WHERE id = ? AND version = ?`).
2. **Una edición baja la cantidad de un lote mientras se vende de ese lote** (RN-EC-07, escenario "Venta simultánea
   contra una baja"): el lote no puede quedar negativo y la edición debe evaluarse con el stock real.

Hoy **no hay ningún bloqueo** en el backend (`grep FOR UPDATE` = 0). MySQL 8.4 / InnoDB en `REPEATABLE READ`
(por defecto). `RegistrarSalida` (ventas, sync de ventas offline) calcula FEFO con una lectura no bloqueante y
`ajustarLote` igual: ya hoy dos salidas simultáneas del mismo lote pueden dejarlo en negativo.

Dato técnico clave: en `REPEATABLE READ` un `SELECT` común lee la foto tomada en la primera lectura de la tx, aunque
otra tx haya confirmado después. Para decidir sobre el stock "de verdad" hay que usar una **lectura bloqueante**
(`FOR UPDATE` / `FOR SHARE`), que siempre ve lo último confirmado. Además, insertar en `movimiento_stock` con `lote_id`
toma un bloqueo compartido sobre la fila de `lote` (chequeo de FK): si la edición tiene el lote con `FOR UPDATE`, nadie
puede insertar movimientos de ese lote hasta que termine.

## Alternativas consideradas
### A. Solo versión optimista en la compra
Simple, pero no protege el stock: la versión de la compra no cambia cuando se vende del lote.
### B. Versión optimista en la compra + bloqueo pesimista de los lotes afectados (solo en la edición)
Protege a la edición de leer stock viejo. No protege a una venta que leyó el stock antes de que la edición confirme:
la venta espera el bloqueo y después inserta con su cálculo viejo (el lote puede quedar en −1).
### C. B + que toda salida de stock que decide por lote también bloquee los lotes del producto (ventas, ajustes)
Serializa por producto todas las escrituras que dependen del stock de un lote. Cumple "el lote nunca queda en negativo".
Costo: tocar `inventario.RegistrarSalida` y `ajustarLote` (no la API de ventas) y aceptar un riesgo bajo de deadlock entre
operaciones de varios productos (InnoDB lo detecta y aborta una).
### D. Cambiar el aislamiento global a `READ COMMITTED` + bloqueos
Resuelve la foto vieja sin lecturas bloqueantes, pero cambia el comportamiento de todas las transacciones del sistema.
Demasiado amplio para esta historia.
### E. Columna `version` en `lote` con UPDATE condicional
Obliga a tocar la versión del lote en cada venta (escritura extra en el camino caliente) y a reintentar; más código que C.

## Decisión
Elegimos **C**, en dos etapas:

**Parte a (US-001a)** — versión optimista de la compra:
- `compra.version INT NOT NULL DEFAULT 1` y `compra.updated_at` (ver `docs/data/editar-compra.md`).
- Dentro de la tx, primero `SELECT … FROM compra WHERE id = ? FOR UPDATE` (serializa dos ediciones de la misma compra;
  la segunda espera y luego ve la versión nueva). Si `version` ≠ la enviada → 409 `version_desactualizada`
  ("Otra persona modificó esta compra, recargá"). Se compara **antes** de validar cambios: una edición "sin cambios"
  con versión vieja también da 409.
- El UPDATE final es `UPDATE compra SET …, version = version + 1 WHERE id = ? AND version = ?` y si afecta 0 filas
  devuelve el mismo 409 (defensa en profundidad).
- Una edición sin cambios no sube la versión (RN-EC-04).

**Parte b (US-001b)** — bloqueo de lotes:
- Orden de bloqueo fijo para evitar deadlocks entre ediciones: `compra` → `lote` (ordenados por `id`) → `producto`
  (lo escribe catálogo al final).
- Inventario agrega al repositorio:
  - `BloquearLotes(ctx, ids)`: `SELECT id FROM lote WHERE id IN (…) ORDER BY id FOR UPDATE`.
  - Lectura de stock bloqueante: `SELECT l.id, …, COALESCE(SUM(m.cantidad),0) FROM lote l LEFT JOIN movimiento_stock m
    ON m.lote_id = l.id WHERE l.id IN (…) GROUP BY l.id FOR SHARE` (lee lo último confirmado).
  - `LotesConStock(ctx, productoID, bloquear bool)`: con `bloquear`, primero
    `SELECT id FROM lote WHERE producto_id = ? ORDER BY id FOR UPDATE` y después la misma consulta con `FOR SHARE`.
- `RegistrarSalida` y `ajustarLote` pasan a usar la variante bloqueante (cambio interno de inventario; la API de ventas y
  ajustes no cambia). Así venta↔edición, venta↔venta y ajuste↔edición quedan serializadas por producto.
- Los errores de MySQL 1213 (deadlock) y 1205 (lock wait timeout) se traducen en `platform/db` a
  `errores.Conflicto("operacion_concurrente", "Otra operación estaba usando los mismos productos. Intentá de nuevo.")`
  → 409. No se reintenta automáticamente.
- SQLite: el servidor no usa SQLite para estos módulos (el escritorio tiene su propio esquema y adaptadores de
  ventas); no hace falta equivalente. `migrations/sqlite` se mantiene en paralelo solo a nivel de esquema.

## Consecuencias
- Positivas: se cumple RN-EC-07/RN-EC-23 y el escenario "Venta simultánea"; además se corrige la carrera preexistente
  venta↔venta sobre el mismo lote.
- Negativas / deuda aceptada:
  - Las ventas de un mismo producto quedan serializadas (en una tienda con 1–3 cajas el impacto es nulo).
  - Riesgo bajo de 409 `operacion_concurrente` en ventas de varios productos que coinciden con una edición o con otra
    venta en orden inverso. El frontend debe mostrar el mensaje y permitir reintentar (ya muestra 409 como toast).
  - Las ventas offline que suben después pueden dejar stock negativo (RN-EC-25, comportamiento existente).
  - "Compra más reciente" no se bloquea: si al mismo tiempo se registra otra compra del mismo producto, el último costo
    lo define la que confirme último. Aceptado (probabilidad muy baja, sin daño de stock).
- Qué hay que hacer ahora: parte a → versión y `FOR UPDATE` de la compra; parte b → métodos bloqueantes de inventario,
  cambio en `RegistrarSalida`/`ajustarLote`, traducción de 1213/1205, test de integración con dos conexiones.
- Cuándo revisarla: si aparecen 409 `operacion_concurrente` en los logs con frecuencia (> 1 por semana), evaluar
  reintento automático en `db.Transactor` o READ COMMITTED (nuevo ADR).

## Enmienda 2026-10-27 (revisión de código y QA de US-001b): READ COMMITTED y reintento acotado
Qué cambia respecto de lo escrito arriba (la opción D se adopta **acotada al `db.Transactor`**, no como cambio global de la base):
- **Aislamiento.** Las transacciones que abre `db.Transactor` sobre MySQL corren en `READ COMMITTED`. Motivo: el `FOR SHARE`
  sobre `movimiento_stock` en `REPEATABLE READ` toma bloqueos de brecha (next-key) sobre `idx_ms_lote` y provocaba deadlocks
  entre ventas de productos distintos con lotes intercalados. En SQLite (escritorio) no cambia nada.
- **Lecturas.** Ya no se usa `FOR SHARE`. Primero se bloquean los lotes con `FOR UPDATE` (siempre `WHERE id IN (…) ORDER BY id`,
  ids resueltos antes sin bloqueo) y después se lee el stock con un `SELECT` común, que en `READ COMMITTED` ve lo último
  confirmado. Cualquier otra transacción que quiera insertar un movimiento de esos lotes espera por el bloqueo compartido de
  la FK sobre la fila del lote. `LotesConStock(…, bloquear)` bloquea solo los lotes con stock distinto de cero (los agotados no
  pueden aportar unidades a una salida) y, si al leer aparece un lote que no se bloqueó, repite (máx. 5 veces).
- **Faltante.** El chequeo de stock insuficiente de `RegistrarSalida` suma los lotes ya leídos bajo bloqueo más los movimientos
  sin lote (`StockSinLote`), en lugar de una lectura aparte del stock total.
- **Reintento.** Si MySQL aborta la transacción más externa por **deadlock (1213)**, `EnTransaccion` la reejecuta hasta 3
  veces con pausa corta + jitter (10 ms × intento + 0–20 ms); lo mismo si inventario no logra estabilizar los lotes
  (`ErrOperacionConcurrente`, no implica esperar). Recién después devuelve 409 `operacion_concurrente` (mensaje genérico).
  La **espera de bloqueo agotada (1205) no se reintenta** (3 × lock wait podría superar los 30 s del proxy del escritorio y
  duplicar una venta offline): se traduce a 409 directo. Se traduce también si el rollback falla. El DSN fija
  `innodb_lock_wait_timeout=10` por sesión (`platform/config`), así un 1205 responde en ~10 s. El aislamiento es un campo
  explícito del `Transactor` (`NuevoTransactorMySQL`); SQLite no lo usa. Condición: el cuerpo de la transacción
  no puede tener efectos fuera de la base ni acumular estado entre intentos (auditado; solo `RevertirSalidas` acumulaba
  `repuestas` y ahora lo reinicia por intento).
- **Sync del escritorio.** `operacion_concurrente` es transitorio: `RecibirVentas` corta el envío con error (el escritorio reintenta
  la tanda, el registro es idempotente) y no marca la venta como rechazada.
- **Supuestos revisados.** `binlog_format` debe ser `ROW` (default de MySQL 8; con `STATEMENT` InnoDB no admite escrituras en
  `READ COMMITTED`). Catálogo y compras usan versión optimista (`UPDATE … WHERE version = ?`): no cambia. Ventas offline:
  idempotencia por clave primaria: no cambia. Estadísticas lee en autocommit, sin transacción. `ObtenerParaEditar` ya serializa
  por `FOR UPDATE` de la compra. Nadie en el repo depende de lecturas repetibles dentro de una transacción.
- Cuándo revisarla: igual que antes, si aparecen 409 `operacion_concurrente` en los logs (ahora ya tras 3 intentos).
