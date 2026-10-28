# 0004. Quitar un renglón de compra con baja lógica (`compra_item.quitado_en`), no desvinculando el lote

- Estado: Propuesto
- Fecha: 2026-10-27
- Decisores: Software Architect + Database Engineer
- Historias relacionadas: US-001c (RN-EC-09, RN-C-01..03)

## Contexto
Al quitar un renglón, el lote y sus movimientos se conservan como historia (RN-EC-09). `lote.compra_item_id` tiene FK a
`compra_item(id)` sin `ON DELETE` (= RESTRICT), así que no se puede borrar el renglón mientras el lote apunte a él.
Hoy solo el módulo compras lee `compra_item` (verificado: estadísticas lee `compra.total`; inventario solo guarda y
lee `lote.compra_item_id`).

## Alternativas consideradas
### A. Baja lógica: `compra_item.quitado_en DATETIME NULL`
Pros: el lote sigue apuntando a su renglón (se puede explicar de qué compra vino ese lote y a qué costo); mismo patrón
que `venta.anulada_en` y `gasto_caja.anulado_en`; sin cambios de FK; reversible. Contras: toda lectura de renglones
debe filtrar `quitado_en IS NULL` (4 consultas, todas en `compras/adapters/mysql`).
### B. Borrar el renglón y poner `lote.compra_item_id = NULL`
Pros: lecturas sin filtro. Contras: el lote queda igual a uno creado por un ajuste manual (se pierde de qué compra
vino, salvo por `referencia_id` del movimiento); compras tendría que escribir en `lote` (tabla de inventario) o pedirle a
inventario que desvincule; borra un dato que la auditoría necesita para explicar "Quitado: Ekos Jabón x10".
### C. Mover renglones quitados a una tabla `compra_item_quitado`
Más tablas y una copia de datos sin beneficio frente a A.

## Decisión
Elegimos **A**.
- `compra_item.quitado_en DATETIME(3) NULL` (SQLite `TEXT NULL`). NULL = vigente.
- El renglón quitado conserva su `cantidad` y `costo_unitario` (los `CHECK` siguen valiendo) y se marca con la hora
  de la edición. Su lote recibe un `EDICION_COMPRA` por **−cantidad del renglón** (en el caso normal es todo el stock
  del lote; si el lote tuvo un ajuste positivo, esas unidades extra quedan, ver Preguntas abiertas).
- Filtran `quitado_en IS NULL`: `Obtener` (renglones del detalle), `consultaResumen` (conteo de productos y
  unidades), `CompraMasReciente` y la lectura para editar. El `total` de la compra ya se guarda recalculado (RN-EC-20).
- Un renglón quitado no se puede "volver a poner": si se manda su `id` en `items` → 422 (se agrega uno nuevo).
- No se agrega índice: `compra_item` ya tiene el índice implícito de la FK `compra_id` y `idx_ci_producto`; el filtro
  `quitado_en IS NULL` se aplica sobre pocas filas por compra/producto.
- Un renglón agregado y quitado en la misma edición nunca llega al servidor (RN-C-03 se cumple en el frontend).

## Consecuencias
- Positivas: historia completa lote→renglón→compra; ninguna tabla de inventario cambia; auditoría consistente.
- Negativas: riesgo de olvidar el filtro en una consulta futura → se cubre con un test de integración que quita un
  renglón y verifica detalle, listado y "más reciente".
- Qué hay que hacer ahora: migración 00006 (parte c) y filtros en el repositorio.
- Cuándo revisarla: si otro módulo empieza a leer `compra_item`, exponer una vista `v_compra_item_vigente`.
