# 0001. Orquestar la edición de una compra desde el módulo compras, en una sola transacción y a través de puertos

- Estado: Propuesto
- Fecha: 2026-10-27
- Decisores: Software Architect (pendiente aprobación en G2)
- Historias relacionadas: US-001, US-001a, US-001b, US-001c

## Contexto
Editar una compra toca datos de tres módulos: `compras` (compra, compra_item, auditoría), `inventario` (lote,
movimiento_stock) y `catalogo` (producto.costo, producto.precio_venta, precio_historial). RN-EC-03 exige todo o nada.

Hoy `RegistrarCompra` ya resuelve el mismo problema: `compras/application` define los puertos `Catalogo` e
`Inventario`, `compras/adapters/enlaces` los implementa llamando a los servicios de los otros módulos en el mismo
proceso, y la transacción viaja en el `context` (`db.Transactor` reutiliza la tx si ya hay una). Las convenciones
(`docs/CONVENCIONES.md`) prohíben que un módulo importe el `domain` de otro.

Lo que falta en los otros módulos:
- Inventario no sabe ajustar la cantidad, el costo ni el vencimiento de un lote **de una compra** ni leer su stock con bloqueo.
- Catálogo ya tiene `RegistrarCostosCompra(cambios, motivo)`: pisa el costo y, si viene un precio distinto al actual,
  cambia el precio y deja historial con origen `COMPRA`. Es exactamente lo que necesitan RN-EC-16..19 si compras decide
  **qué** mandar.
- Nadie sabe cuál es "la compra más reciente de un producto" (RN-EC-15). Es un dato de compras (compra.fecha, created_at).

## Alternativas consideradas
### A. Compras orquesta (caso de uso `EditarCompra`) y llama a puertos nuevos de inventario; catálogo se reutiliza tal cual
Pros: mismo patrón que `RegistrarCompra` y `AnularVenta`; una sola transacción; las reglas de "más reciente" y de "qué
cambió" quedan en compras, que es dueña del dato. Contras: compras crece (el caso de uso más largo del sistema).
### B. Eventos de dominio en proceso (compras publica `CompraEditada`, inventario y catálogo reaccionan)
Pros: menos acoplamiento directo. Contras: no hay bus de eventos en el repo; para todo-o-nada los handlers deben correr
síncronos dentro de la misma tx (es la opción A con indirección); el 409 `compra_consumida` necesita la respuesta de
inventario **antes** de escribir, lo que no encaja con "publicar y olvidar". Agrega infraestructura nueva sin necesidad.
### C. Hacer la edición con SQL directo desde el repositorio de compras (UPDATE de lote, INSERT de movimientos, UPDATE de producto)
Pros: rápido de escribir. Contras: rompe los límites de módulo (compras escribiría tablas de inventario y catálogo),
duplica reglas (stock no negativo, historial de precios) y deja sin tests de unidad la lógica.
### D. Saga / compensaciones
Descartada: es un monolito con una sola base; una tx local es más simple y más fuerte.

## Decisión
Elegimos **A**.

1. `compras/application.Servicio.EditarCompra(ctx, ComandoEditarCompra)` abre **una** transacción y en ella:
   1. Lee la compra con bloqueo (`SELECT … FOR UPDATE`) y compara `version` (ver ADR 0002).
   2. Arma el plan de edición con una función pura de dominio `domain.PlanificarEdicion(actual, cambios, ahora)`:
      valida (RN-EC-02, producto fijo, ids de renglón), calcula diferencias por renglón, total nuevo, renglones
      agregados/quitados y si hay cambios (RN-EC-04). Si no hay cambios, devuelve sin escribir nada.
   3. Valida empresa y productos nuevos por el puerto `Catalogo` (existente).
   4. Bloquea los lotes afectados y lee su stock por el puerto `Inventario` (nuevo) y verifica consumo en dominio
      (`domain.VerificarConsumo`), juntando **todos** los renglones bloqueados en un único 409 `compra_consumida`.
   5. Guarda compra/renglones (repositorio propio), aplica los cambios de lote (puerto `Inventario`), recalcula costo y
      precio (puerto `Catalogo` → `RegistrarCostosCompra` con motivo "Edición de compra del dd/mm/aaaa") y guarda la
      auditoría (`compra_edicion`, ver ADR 0005).
2. **Puertos nuevos de compras hacia inventario** (en `compras/application`, implementados en `compras/adapters/enlaces`):
   - `LotesDeRenglones(ctx, compraItemIDs) (map[itemID]LoteRenglon, error)` — sin bloqueo, para mostrar `consumido`
     en `GET /compras/{id}`.
   - `BloquearLotesDeRenglones(ctx, compraItemIDs) (map[itemID]LoteRenglon, error)` — dentro de la tx, con bloqueo.
   - `AjustarLoteDeRenglon(ctx, AjusteRenglon{CompraID, CompraItemID, LoteID, Diferencia, Costo, Vencimiento, Nota})`.
   - `Ingresar` (existente) para renglones nuevos (movimiento `COMPRA`, RN-EC-10).
   `LoteRenglon = {LoteID, CompraItemID, ProductoID, Stock}`.
3. **Casos de uso nuevos en inventario** (`inventario/application`), que son los que llaman los enlaces:
   - `LotesDeCompra(ctx, compraItemIDs, bloquear bool) ([]domain.LoteStock, error)`.
   - `AjustarLoteDeCompra(ctx, AjusteLoteCompra)`: si `Diferencia != 0` registra un movimiento `EDICION_COMPRA`
     (ADR 0003) en el **mismo lote**, con `referencia_id` = compra; si cambian costo o vencimiento actualiza el lote en
     el lugar (RN-EC-11). Vuelve a verificar `stock + diferencia >= 0` (defensa en profundidad → `stock_insuficiente`).
4. **Catálogo no cambia.** Compras decide qué mandar a `RegistrarCostos` (solo productos afectados, solo si la compra es
   la más reciente, y `PrecioVenta` solo si el precio del renglón cambió). El puerto `ProductoCompra` suma `Costo` y
   `PrecioVenta` (ya están en `InfoProducto`) para no mandar cambios que no cambian nada (evita subir la versión del
   producto sin motivo).
5. **"Compra más reciente"** se resuelve en el repositorio de compras: `CompraMasReciente(ctx, productoID)` devuelve
   `{CompraID, Costo}` del último renglón vigente (mayor `compra_item.id`) de la compra con mayor `(fecha, created_at, id)`.
   Se evalúa antes y después de escribir, dentro de la misma tx (ve sus propias escrituras).
6. `Origen` (WEB/MOVIL/DESKTOP) se define en `compras/domain` con el mismo contrato que en ventas (no se importa
   `ventas/domain`). Duplicación chica aceptada; si aparece un tercer uso, se mueve a `kernel`.

## Consecuencias
- Positivas: un solo patrón para todo el backend; todo-o-nada garantizado por la tx; la lógica de "qué cambió" y
  "qué es lo más reciente" es pura y testeable con fakes; catálogo no se toca.
- Negativas / deuda aceptada:
  - `POST /compras` sigue pisando el último costo siempre (aunque la compra cargada tenga fecha anterior a otra). La
    edición usa "más reciente por fecha" (D-4). Queda una diferencia de criterio entre registrar y editar
    (ver Preguntas abiertas en `docs/architecture/editar-compra.md`).
  - `Origen` duplicado en compras y ventas.
- Qué hay que hacer ahora: ver plan por parte en `docs/architecture/editar-compra.md`.
- Cuándo revisarla: si se suma "anular compra" (D-9) reutilizar este mismo caso de uso como "quitar todos los renglones"
  sin la regla de mínimo 1; si los módulos se separan en servicios, esta orquestación pasa a ser una saga (nuevo ADR).
