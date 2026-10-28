# US-001 — Editar una compra registrada (historia paraguas)
Épica: EP-01 Compras de mercadería · Prioridad: Should · Estimación: ver sub-historias · Estado: Dividida (entrega por partes)

> Convención: `US-NNN-slug.md`, correlativo, un archivo por historia, IDs nunca reutilizados. Las reglas de esta
> funcionalidad usan el prefijo `RN-EC-NN` (Editar Compra). Una historia demasiado grande se divide en sub-historias
> `US-NNNa/b/c`, cada una con su archivo, que son **rebanadas verticales** (backend + pantalla, usable sola). Este archivo
> guarda el contexto, las reglas y las decisiones comunes; los criterios Gherkin viven en cada sub-historia.

## Historia
Como dueña del negocio, quiero editar una compra ya registrada (fecha, empresa, notas y renglones), para corregir errores
de carga sin dejar el stock, los costos ni los totales mal.

## Entrega por partes (orden)
1. [US-001a](US-001a-editar-cabecera-compra.md) — Editar cabecera + versión + auditoría + botón "Editar" + historial de ediciones.
2. [US-001b](US-001b-editar-renglones-compra.md) — Editar renglones existentes (cantidad con bloqueo, costo, vencimiento, precio) + último costo/precio/historial.
3. [US-001c](US-001c-agregar-quitar-renglones-compra.md) — Agregar y quitar renglones + recálculo de último costo al quitar o cambiar fecha.

## Estado actual (lo que hace el código hoy)
- `POST /compras` guarda `compra` + `compra_item`, crea **un lote por renglón** (`lote.compra_item_id`) con un movimiento
  `COMPRA` (+cantidad, `referencia_id` = compra) y llama a `RegistrarCostosCompra`: pisa `producto.costo` ("último costo") y,
  si vino `precioVentaNuevo` distinto del actual, cambia `producto.precio_venta` con una fila en `precio_historial`
  (origen `COMPRA`, motivo "Compra del dd/mm/aaaa"). Todo en una transacción.
- No existe PUT/DELETE de compras. `compra` no tiene `version` ni `updated_at`.
- Stock = suma de `movimiento_stock` (tipos `INICIAL, COMPRA, VENTA, ANULACION, AJUSTE`; enum en MySQL, CHECK en SQLite).
  Las salidas salen de lotes por FEFO. El ajuste manual a un lote ya impide dejarlo en negativo (`stock_insuficiente`).
- Las ventas guardan una foto del costo (`venta_item.costo_unitario`, `venta.costo_total`).
- Estadísticas lee `compra.total` por `fecha` en vivo. Las compras no entran en ganancia ni en `efectivoNeto`. No existe
  cierre/arqueo de caja (fuera de alcance en `docs/PLAN.md`).
- Acceso con una sola clave compartida: no hay usuarios.

## Decisiones del cliente (cerradas)
- D-1: Se puede editar todo: fecha, empresa, notas y renglones.
- D-2: Bajar o quitar unidades ya consumidas del lote se BLOQUEA con mensaje claro.
- D-3 (P-01): los cambios de cantidad usan un tipo nuevo de movimiento `EDICION_COMPRA`.
- D-4 (P-02): "compra más reciente" se define por `fecha` de la compra; desempate por `created_at`.
- D-5 (P-06): auditoría con tabla de ediciones (fecha, origen, valores antes/después), visible en el detalle. Sin "quién".
- D-6 (P-03): el precio de venta del producto nunca se revierte por una edición.
- D-7 (P-04): sin plazo máximo para editar.
- D-8 (P-05): no se recalcula la ganancia histórica (las ventas son "foto" del momento).
- D-9 (P-07): anular/eliminar una compra completa es otra historia, fuera de alcance.
- D-10 (P-08): se mantiene lo actual: el mismo producto puede estar en varios renglones.
- D-11 (P-09): el motivo de la edición es opcional (hasta 255 caracteres).

## Reglas de negocio
Se indica entre corchetes en qué sub-historia se implementan.

**Alcance y todo-o-nada**
- RN-EC-01 [a,b,c]: Se puede editar fecha, empresa, notas y renglones. El **producto de un renglón existente no se cambia**:
  se quita el renglón y se agrega otro.
- RN-EC-02 [a,b,c]: Mismas validaciones que al registrar: fecha no futura (tolerancia 1 hora), notas <= 500 caracteres,
  al menos 1 renglón, máximo 300, cantidad > 0, costo >= 0, precio de reventa > 0 si se informa, producto existente y no combo.
- RN-EC-03 [a,b,c]: Todo o nada en una sola transacción (compra, renglones, lotes, movimientos, producto, historial, auditoría).
- RN-EC-04 [a,b,c]: Una edición sin cambios no genera movimientos, historial, auditoría ni cambia la versión.

**Stock y lotes**
- RN-EC-05 [b,c]: El stock nunca se edita directo: cada cambio de cantidad es un movimiento sobre **el mismo lote** del
  renglón, por la diferencia, con `referencia_id` = compra y nota "Edición de compra del dd/mm/aaaa". Nunca se modifican ni
  borran movimientos existentes.
- RN-EC-06 [b,c]: Tipo del movimiento: `EDICION_COMPRA` (positivo si sube, negativo si baja).
- RN-EC-07 [b]: Bloqueo por consumo. Consumido = cantidad original - stock actual del lote (suma de todos sus movimientos).
  La nueva cantidad no puede ser menor a lo consumido (`stock del lote + diferencia >= 0`). Si no se cumple se rechaza
  **toda** la edición con 409 `compra_consumida` y un mensaje por renglón, p. ej. "Kaiak Clásico: de las 4 unidades de
  esta compra ya salieron 3 (ventas o ajustes). Solo podés bajar hasta 3 unidades."
- RN-EC-08 [b]: Subir la cantidad suma stock al mismo lote.
- > Actualización 2026-10-27 (cliente, P-A2): al quitar el renglón se descuenta solo la cantidad del renglón, no lo sumado por ajustes manuales.
- RN-EC-09 [c]: Quitar un renglón solo si el lote no tiene consumo; se registra un `EDICION_COMPRA` negativo por todo el
  stock del lote. El lote y sus movimientos se conservan como historia.
- RN-EC-10 [c]: Renglón nuevo = lote nuevo con su vencimiento y costo y un movimiento `COMPRA` positivo.
- RN-EC-11 [b]: Cambiar costo o vencimiento actualiza el lote en el lugar (sin movimiento), aunque esté parcialmente
  consumido; afecta solo al stock restante y a ventas futuras (margen, FEFO, alertas).
- RN-EC-12 [b]: Las ventas ya hechas no se recalculan.
- RN-EC-13 [b,c]: Un combo no puede ser renglón. El stock de un combo cambia solo (deriva de sus componentes); el consumo
  de un lote vía ventas de combos cuenta como consumo.
- RN-EC-14 [b,c]: Se puede editar una compra con productos inactivos y la edición no los reactiva. Agregar un renglón con
  producto inactivo se permite como hoy en el backend (la pantalla ofrece activos).

**Costo y precio del producto**
- RN-EC-15 [b,c]: Una compra es "la más reciente" de un producto si ninguna otra compra que lo contenga tiene `fecha`
  posterior (igual fecha: mayor `created_at`), evaluado con el estado posterior a la edición.
- RN-EC-16 [b,c]: Último costo: solo si es la más reciente del producto, pasa a ser el costo del renglón editado/agregado
  (si hay varios renglones del producto, el último de la lista). Si no, el producto no se toca.
- RN-EC-17 [b,c]: Precio de venta: solo si es la más reciente **y** el `precioVentaNuevo` del renglón cambió (o se informó
  por primera vez) **y** difiere del precio actual del producto. Si el precio del renglón no cambió no se reaplica.
- RN-EC-18 [b,c]: El cambio de precio agrega una fila a `precio_historial` (origen `COMPRA`, motivo "Edición de compra del
  dd/mm/aaaa"). Las filas previas no se tocan. Vaciar un `precioVentaNuevo` no revierte el precio (D-6).
- RN-EC-19 [c]: Si al quitar un renglón o cambiar la fecha cambia cuál es la compra más reciente de un producto, el último
  costo se recalcula con la nueva más reciente; si no queda ninguna, se mantiene. No se revierte el precio de venta.

**Total, reportes y caja**
- RN-EC-20 [b,c]: `total` = suma de cantidad x costo de los renglones resultantes, recalculado en el servidor.
- RN-EC-21 [a,b,c]: Estadísticas muestra el valor actual de `compra.total` según la `fecha` vigente; editar monto o fecha
  cambia períodos pasados y puede mover la compra de un período a otro. Ganancia y efectivo neto no cambian. No hay cierres.
- RN-EC-22 [a,b,c]: Los movimientos conservan su fecha real; solo cambia la fecha de la compra.

**Concurrencia y auditoría**
- RN-EC-23 [a,b,c]: La compra tiene `version`. El cliente envía la que leyó; si cambió, 409 `version_desactualizada` y no
  se aplica nada. Verificación de consumo y movimientos dentro de la transacción con los lotes bloqueados (`FOR UPDATE`).
- RN-EC-24 [a,b,c]: Cada edición efectiva guarda fecha/hora, **origen** (WEB, MOVIL o DESKTOP, enviado por el cliente como
  en ventas; por defecto WEB), motivo opcional y los valores antes/después (cabecera y, desde b, renglones). Sin "quién".
  El detalle de la compra muestra "Historial de ediciones".
- RN-EC-25 [b,c]: Las compras requieren conexión (no hay edición offline). Si una venta offline sube después de una baja de
  cantidad, el stock puede quedar negativo (comportamiento existente de ventas offline).

## Impacto en datos (para el diseño)
- [a] `compra`: agregar `version` y `updated_at`. Tabla `compra_edicion` (id, compra_id, fecha, origen, motivo, antes JSON,
  después JSON). Migración en paralelo MySQL y SQLite.
- [b] `movimiento_stock.tipo`: nuevo valor `EDICION_COMPRA` (enum MySQL, CHECK SQLite, `invdomain.TipoMovimiento`, filtro de
  `/inventario/movimientos`, etiquetas en `frontend/.../inventario/estados.ts`, `docs/API.md`). Inventario: casos de uso para
  ajustar cantidad/costo/vencimiento de un lote desde una compra y leer consumo con bloqueo. Catálogo: aplicar costo/precio
  con motivo de edición y consultar la compra más reciente por producto.
- [c] Quitar renglón: `lote.compra_item_id` tiene FK a `compra_item`. Recomendación: baja lógica (`quitado_en`) filtrada
  en `Obtener`, `Unidades` y `Productos`. Alternativa: `compra_item_id` NULL. Decide el arquitecto.

## API (a validar por arquitectura)
`PUT /compras/{id}` con `{fecha, empresaId, notas, version, origen?, motivo?, items?}`. `items` ausente = renglones sin
cambios (US-001a). Desde US-001b, `items:[{id?, productoId, cantidad, costoUnitario, precioVentaNuevo?, vencimiento?}]`:
con `id` = existente, sin `id` = nuevo (c), los existentes omitidos = quitados (c). Respuesta: detalle con `version`,
`editadaEn` y `ediciones`. Errores: 409 `compra_consumida`, 409 `version_desactualizada`, 404 `compra_no_encontrada`, 422.

## Preguntas abiertas
Ninguna bloqueante: P-01 a P-09 resueltas (ver Decisiones). Pendiente de diseño técnico (no de negocio): baja lógica del
renglón vs `compra_item_id` NULL.

## Fuera de alcance
- Anular o eliminar una compra completa (historia aparte).
- Recalcular ganancias o costos de ventas ya hechas.
- Cambiar el producto de un renglón existente.
- Identificar a la persona que edita.
- Edición sin conexión desde el escritorio.
