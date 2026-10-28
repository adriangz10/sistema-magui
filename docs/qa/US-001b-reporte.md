# Reporte de QA — US-001b Editar los renglones existentes de una compra

Rama `feature/edit-purchase` (cambios sin commitear) · 2026-10-27 · Plan: `docs/qa/US-001b-test-plan.md`

## Veredicto: CONDICIONADO a la prueba con MySQL real
No hay bugs críticos ni altos abiertos y toda la regresión automatizada está en verde, pero esta historia depende de
bloqueos `FOR UPDATE` / `FOR SHARE`, de SQL nuevo (lotes por renglón, compra más reciente, `ActualizarLote`) y de
concurrencia, y nada de eso se ejecutó: no hay DSN de MySQL ni Docker. Los tests MySQL de la historia
(`TestEditarRenglonesMySQL`, `TestEditarRenglonNoEsLaMasRecienteMySQL`, `TestEditarTodoONadaMySQL`,
`TestVentaSimultaneaContraBajaMySQL`, `TestVentasSobreCompraEditadaMySQL`) existen pero hacen SKIP. Sin OK de QA hasta correrlos.

## Resultados de ejecución
| Suite | Comando | Resultado |
|---|---|---|
| Backend completo | `go test -count=1 ./...` (en `backend/`) | Verde (los `*MySQL` hacen SKIP) |
| Backend vet | `go vet ./...` | Sin hallazgos |
| Backend, rerun con los tests QA nuevos | `go test -count=1 ./...` | Verde |
| Frontend unit | `npm test` | 12 archivos, 63 tests, todos pasan |
| Frontend build | `npm run build` | OK |

Tests agregados por QA (archivos nuevos, prefijo `TestQA`):
- `backend/internal/compras/application/edicion_renglones_qa_test.go`: venta anulada (4 -> 1, -3), mensaje de consumo (salieron N / hasta N), lote consumido que sí permite subir y cambiar costo, total/versión/auditoría de cantidad+costo en una compra en versión 3, sólo precio sin movimiento de stock, dos personas editan a la vez, cabecera+renglones en una sola edición, tope de cantidad/total (BUG-001).
- `backend/internal/compras/adapters/http/editar_renglones_qa_http_test.go`: bordes de consumo por HTTP (justo lo consumido 200, consumido-1 409 con campo y mensaje, 0 y negativos 422), valores inválidos (costo -1, precio 0 y negativo, producto ajeno, sin id), renglón repetido y ajeno.

Cobertura de los casos borde pedidos (detalle en el plan):
bajar justo a `consumido` (ok) y a consumido-1 (409 con todos los renglones rechazados), subir cantidad, costo/vencimiento sin movimiento,
`vencimiento: null` -> SIN_FECHA, precio sólo si es la más reciente por fecha, edición de compra vieja sin pisar costo, alta con fecha vieja sin pisar costo,
edición sin cambios (sin escrituras), `precioVentaActual`, total recalculado y auditoría MODIFICADO: todos PASAN con fakes.

## Bugs

BUG-001 — Sin tope de cantidad ni de total al editar (ni al dar de alta) un renglón
Severidad: baja
Historia: US-001b (también aplica al alta, US-001, comportamiento previo)
Pasos:
1. `PUT /api/compras/{id}` con un renglón `cantidad: 3000000000`, `costoUnitario: "1000"`, `version` vigente.
2. (Reproducible sin MySQL) `go test -v -run TestQASinTopeDeCantidadNiDeTotal ./internal/compras/application/`.
Esperado: 422 sobre `items[i].cantidad` (o el campo que corresponda) porque `compra_item.cantidad` es `INT` y `compra.total` es `DECIMAL(12,2)` (máx. 9.999.999.999,99).
Obtenido: la validación de dominio lo acepta (total calculado 3.000.000.021.000,00); contra MySQL fallaría recién al escribir (error de rango, 500) y, en el cálculo en `int64`, valores aún mayores podrían desbordar el total. En SQLite no hay error.
Evidencia: log del test `BUG-001 vigente: cantidad 3.000.000.000 x $1.000 (total 3000000021000.00) pasa la validación`.
Riesgo real bajo (un usuario de un solo local, la pantalla no ofrece esos valores). Sugerencia: tope razonable de cantidad (p. ej. 1.000.000) y de total en `validarItems`.

No se encontraron bugs de severidad media, alta ni crítica en lo ejecutable.

## Observaciones (no son bugs confirmados)
- OBS-1 — Desempate "misma fecha". Lo implementado compara el instante completo de `compra.fecha` y recién después `created_at`. La historia/RN-EC-15 habla de "igual fecha". En el uso normal coinciden (una compra creada después tiene un instante posterior), pero si se edita la fecha de una compra a un día (queda a las 12:00 hora argentina) y otra compra de ese mismo día se cargó por la tarde, la más reciente será esta última aunque se haya creado antes. Conviene que PO confirme si "fecha" es día o instante; con el contrato actual el comportamiento es consistente con US-001a.
- OBS-2 — El código de error del JSON mal formado (campo desconocido, cantidad decimal, vencimiento no fecha, costo con 3 decimales) es 422 `json_invalido`, no 400. Es la convención del backend (`httpx.LeerJSON`); sólo hay que tenerlo presente si la API se documenta como 400. Lo cubre `TestQAPutRenglonValoresInvalidos`.
- OBS-3 — El mensaje 409 real ("Kaiak Clásico: de las 4 unidades de esta compra ya salieron 3 (ventas o ajustes). Solo podés bajar hasta 3 unidades.") sigue RN-EC-07 y es más largo que el del Gherkin; coherente con la regla.
- OBS-4 — El SQL nuevo con `FOR SHARE` y `GROUP BY ... HAVING` (en `LotesConStock` y `LotesPorCompraItem`) es válido en MySQL 8.4 (imagen de `deploy/`) pero no en MariaDB; sólo se verificó por lectura.
- La revisión de `edicion.go` (aplicación), `domain/edicion.go`, `inventario/application/servicio.go`, `inventario/adapters/mysql/repositorio.go`, el handler y `edicion-compra.ts` no mostró otras divergencias con el Gherkin: orden de bloqueo por id, verificación de consumo antes de escribir, precio sólo si cambió el del renglón, costo sólo si es la compra más reciente evaluada después de escribir los renglones, vaciar precio sin revertir, auditoría `MODIFICADO` sólo para renglones con cambios.

## No probado (y por qué)
- **MySQL real**: sin DSN y sin Docker (el engine no responde). No se leyó `backend/.env` ni se probaron credenciales. Sin cubrir: SQL de `LotesPorCompraItem`, `ActualizarLote`, `CompraMasReciente` (orden), movimientos `EDICION_COMPRA` y migración `00005` sobre MySQL con datos, filas reales de `precio_historial`, rollback real, estadísticas del período con la compra editada, ventas pasadas intactas y FEFO con el nuevo vencimiento, producto inactivo que sigue inactivo.
- **Concurrencia real**: venta simultánea contra una baja y doble PUT. Sólo se probó que las rutas usen lectura bloqueante y la traducción de deadlock/timeout (`TestTraducirConcurrencia`); no hay evidencia de que los bloqueos serialicen de verdad ni de que no haya deadlocks.
- **UI en navegador**: botón Editar, "ya salieron N de M", cantidad mínima, resumen de efectos, mensaje de bloqueo junto al renglón, toast. Sólo specs unitarios (`edicion-compra.spec.ts`, `compra-form.component.spec.ts`) y lectura de código. No existe suite Playwright en el repo.
- **API manual** con el backend levantado (`cmd/api` sólo corre contra MySQL).

## Condiciones para dar OK de QA
1. Correr `go test -count=1 ./...` con `MAGUI_TEST_MYSQL_DSN` apuntando a un MySQL 8.4 de desarrollo y revisar que ninguno de los `*MySQL` quede en SKIP ni falle (en especial `TestVentaSimultaneaContraBajaMySQL`).
2. Una pasada manual de API (PUT con subir, bajar, bloquear, varios renglones, versión vieja) y de UI sobre una base con datos.
3. Decidir sobre BUG-001 (puede ir a backlog) y confirmar OBS-1 con el PO.
