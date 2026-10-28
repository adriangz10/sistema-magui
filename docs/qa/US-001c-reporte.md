# Reporte de QA — US-001c Agregar y quitar renglones de una compra

Rama `feature/edit-purchase` (cambios sin commitear) · 2026-10-27 · Plan: `docs/qa/US-001c-test-plan.md`

## Veredicto: CONDICIONADO a la prueba con MySQL real
No se encontraron bugs críticos, altos ni medios, y la regresión automatizada está en verde. Pero la historia depende de SQL nuevo (baja lógica `quitado_en`, `QuitarItems` con verificación de filas, `CompraMasReciente` filtrando quitados, `FOR UPDATE`) y de la migración `00006` en MySQL, y nada de eso se ejecutó: no hay DSN ni Docker. `TestAgregarYQuitarRenglonesMySQL` y `TestCambiarFechaRecalculaUltimoCostoMySQL` existen pero hacen SKIP. No doy OK de QA hasta correrlos.

## Resultados de ejecución
| Suite | Comando | Resultado |
|---|---|---|
| Backend completo | `go test -count=1 ./...` (en `backend/`) | Verde (los `*MySQL` hacen SKIP) |
| Backend vet | `go vet ./...` | Sin hallazgos |
| Frontend unit | `npm test` | 13 archivos, 87 tests, todos pasan |
| Frontend build | `npm run build` | OK |

Tests agregados por QA (archivos nuevos):
- `backend/internal/compras/application/agregar_quitar_qa_test.go` (12 tests `TestQA*`): quitar el último de dos renglones del mismo producto con el catálogo en el costo del quitado (el test del desarrollador no lo ejercía), quitar+agregar el mismo producto, quitar todos los originales con un nuevo, fecha + quitar en una edición, quitar en compra que no es la más reciente, dos nuevos del mismo producto, total con decimales, versión vencida sin escribir, consumo de 1 unidad, tope 300 con 300 renglones reales (301 = 422; quitar 1 + agregar 1 = ok), JSON de auditoría, detalle tras quitar.
- `frontend/src/app/features/compras/edicion-compra.qa.spec.ts` (11 tests): nuevo descartado sin cambios, costo faltante en nuevo, tope 300 con quitados, total sin quitados con decimales, resumen, mapeo de errores (fuera de rango, dos nuevos + un quitado), historial AGREGADO/QUITADO sin "undefined"/"null".
Un fallo inicial de mi spec fue un error de cuenta mío (145.783,68 es correcto), no del producto.

## Bugs
No se encontraron bugs en lo ejecutable ni en la revisión de código. Se revisaron `domain/edicion.go`, `application/edicion.go` (orden: bloqueo y verificación antes de escribir, quitar sólo la cantidad del renglón, costos evaluados con la compra ya escrita), `repositorio.go` (todos los SQL de `compra_item` filtran `quitado_en IS NULL`; `QuitarItems` falla si no coinciden las filas), `servicio.go` de inventario (`SoloCantidad`) y el front.

## Observaciones (no son bugs confirmados)
- OBS-1 — Mensaje de consumo de una unidad. Con cantidad 1 consumida el mensaje es "X: ya salió la única unidad; no se puede quitar." (sin "podés bajar hasta"), distinto del texto del Gherkin, que es para N>1. Es coherente, solo hay que conocerlo.
- OBS-2 — Escenario "agregar y quitar en la misma edición" (RN-C-03) se cumple en el front (el nuevo descartado nunca viaja); el backend no tiene forma de recibirlo. Falta verificarlo con clic real.
- OBS-3 — El backend acepta agregar un producto **inactivo** por API (`productosDeEdicion` sólo valida existencia y combo). La historia sólo exige que el buscador no lo ofrezca, y lo hace (`estado: 'activos'`). Si el PO quiere bloquearlo en el servidor, es un cambio de regla.
- OBS-4 — Orden "último renglón del producto" por `ci.id DESC` depende de que los ids sean UUID v7 monotónicos (lo son: `google/uuid` v7 en el mismo proceso). Si algún día se importaran compras con otros ids, el criterio cambiaría.
- OBS-5 — Hereda BUG-001 de US-001b (sin tope de cantidad/total), ahora también aplicable a renglones nuevos. Severidad baja, sigue abierto.
- OBS-6 — `detalleRenglonHistorial` asume `cambios` como arreglo; el backend siempre lo emite no nulo, así que no hay problema real.
- OBS-7 — El 404/422 de producto inexistente/combo por HTTP real no tiene test (el fake HTTP acepta cualquier producto); el mapeo tipo -> status es el general de `httpx`.

## No probado (y por qué)
- **MySQL real**: migración `00006` con datos, baja lógica, `FOR UPDATE`, `CompraMasReciente` (orden con `DATE(DATE_SUB(...))`), movimientos `EDICION_COMPRA`/`COMPRA` y `precio_historial` reales, rollback real, estadísticas del período, lote del renglón quitado visible en el historial de inventario, FEFO con lotes en cero.
- **Concurrencia real**: dos PUT simultáneos, venta contra quitar.
- **UI en navegador**: botón "Agregar producto"/"Quitar"/"Deshacer", resumen de efectos, render del historial con QUITADO (que no muestre "undefined"), foco y accesibilidad (AXE). Sólo specs unitarios y lectura de plantilla; no existe suite Playwright.
- **API manual** con backend levantado (`cmd/api` sólo corre con MySQL).
- Rollback de la migración con renglones quitados ("reviven", advertido en la propia migración).

## Condiciones para dar OK de QA
1. Correr `go test -count=1 ./...` con `MAGUI_TEST_MYSQL_DSN` hacia un MySQL 8.4 de desarrollo; ningún `*MySQL` en SKIP ni en rojo.
2. Pasada manual de UI y API: agregar, quitar, deshacer, quitar con consumo (409), fecha, historial con QUITADO.
3. Decidir OBS-3 y BUG-001 (puede ir a backlog).
