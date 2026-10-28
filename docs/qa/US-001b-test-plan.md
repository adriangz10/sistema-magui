# Plan de pruebas — US-001b Editar los renglones existentes de una compra

Rama: `feature/edit-purchase` (sin commitear) · Fecha: 2026-10-27 · Autor: QA
Historia: `docs/product/stories/US-001b-editar-renglones-compra.md` · Reglas: `US-001-editar-compra.md` · Contrato: `docs/API.md` "Editar una compra" · Diseño: `docs/architecture/editar-compra.md`

## Alcance y entorno
- Backend Go: dominio/aplicación de compras (`PlanificarEdicion`, `VerificarConsumo`, `EditarCompra`), inventario (lectura bloqueante, `AjustarLoteDeCompra`, movimiento `EDICION_COMPRA`), migración `00005`, frontend Angular (`edicion-compra.ts`, formulario).
- Entorno de esta corrida: Go y Node locales. **Sin MySQL de test ni Docker** (sin DSN). Los tests `*MySQL` hacen SKIP. Sin e2e Playwright en el repo.
- Tipos: unit (dominio/aplicación con repositorios en memoria), HTTP (router chi real + fakes), integración MySQL (`MAGUI_TEST_MYSQL_DSN`), frontend unit (Vitest), manual UI.
- Archivos de QA nuevos de esta pasada: `backend/internal/compras/application/edicion_renglones_qa_test.go` y `backend/internal/compras/adapters/http/editar_renglones_qa_http_test.go` (prefijo `TestQA`).

Estados: PASA, NO EJEC. (con motivo). Los casos "unit" con fakes validan la lógica, no el SQL.

## Matriz criterio Gherkin -> caso -> tipo -> estado
| # | Escenario | Caso | Tipo | Estado |
|---|---|---|---|---|
| 1 | Subir la cantidad (+2, total 234.120) | `TestEditarRenglonSubirCantidad`; `TestRenglonesSubirCantidadYTotal` | unit | PASA |
| 1b | idem en MySQL (movimiento en el lote) | `TestEditarRenglonesMySQL` | integración MySQL | NO EJEC. (SKIP, sin DSN) |
| 2 | Bajar sin tocar lo consumido (-1) | `TestEditarRenglonBajarSinSuperarElConsumo` | unit | PASA |
| 3 | Bloqueo al bajar bajo lo consumido: 409, nada cambia | `TestEditarRenglonBloqueoPorConsumoNoEscribeNada`; `TestVerificarConsumo`; `TestQAPutRenglonConsumoBorde` (409 + campo + mensaje) | unit / HTTP | PASA |
| 4 | Bajar justo hasta lo consumido (stock 0) | `TestEditarRenglonBajarJustoHastaLoConsumido`; `TestQAPutRenglonConsumoBorde` ("justo lo consumido", 200) | unit / HTTP | PASA |
| 5 | Consumo por ajuste y combo | `TestQAMensajeDeConsumoIndicaCuantasSalieron` (mensaje "salieron 3, hasta 3"); el origen del consumo es indistinto porque se calcula por stock del lote | unit | PASA. El consumo real vía combo/ajuste: `TestVentasSobreCompraEditadaMySQL` NO EJEC. |
| 6 | Venta anulada devuelve las unidades (4 -> 1, -3) | `TestQAVentaAnuladaDevuelveLasUnidades` | unit | PASA |
| 7 | Varios renglones bloqueados: el 409 los lista todos (RN-B-02) | `TestEditarRenglonBloqueoPorConsumoNoEscribeNada` (items[0] e items[1]) | unit | PASA |
| 8 | Cambiar costo y vencimiento sin movimiento | `TestEditarRenglonCostoYVencimientoSinMovimiento`; `TestAjustarLoteDeCompraCostoYVencimientoSinMovimiento` | unit | PASA |
| 8b | Ventas pasadas conservan costo y ganancia; próximas usan el nuevo costo y FEFO | `TestVentasSobreCompraEditadaMySQL` | integración MySQL | NO EJEC. (SKIP) |
| 8c | `vencimiento: null` deja el lote sin vencimiento (SIN_FECHA) | `TestEditarRenglonVencimientoNullDejaElLoteSinVencimiento`; `TestRenglonesVencimientoNullVaciaYVaciarPrecio` | unit | PASA (el estado SIN_FECHA se deriva de `fecha_vencimiento` NULL: verificado por lectura) |
| 9 | Es la más reciente: actualiza costo y precio + historial | `TestEditarRenglonEsLaMasRecienteActualizaCostoYPrecio` (motivo "Edición de compra del 25/10/2026") | unit | PASA. Fila de `precio_historial` real: NO EJEC. (MySQL) |
| 10 | No es la más reciente: el producto no se toca | `TestEditarRenglonNoEsLaMasReciente`; `TestEditarRenglonNoEsLaMasRecienteMySQL` | unit / MySQL | PASA (unit) / NO EJEC. (MySQL) |
| 11 | Desempate por `created_at` | `TestEditarRenglonDesempatePorCreatedAt` (ambos sentidos) | unit | PASA. Orden SQL `fecha DESC, created_at DESC, id DESC, item DESC`: revisado, NO EJEC. |
| 12 | Precio sin cambios no pisa un precio manual posterior | `TestEditarRenglonPrecioSinCambioNoSeReaplica` | unit | PASA |
| 13 | Vaciar `precioVentaNuevo` no revierte | `TestEditarRenglonVaciarPrecioNoRevierte` | unit | PASA |
| 14 | Precio nuevo igual al actual: sin fila de historial | `TestEditarRenglonPrecioIgualAlActualNoEmite` | unit | PASA |
| 14b | Sólo cambia el precio: sin movimiento de stock | `TestQASoloPrecioNoMueveStock` | unit | PASA |
| 15 | Estadísticas: sube el total del período, ganancia y efectivo no cambian | total recalculado: `TestQATotalVersionYAuditoriaDeCantidadYCosto`; consulta de período | unit / integración | Total PASA. Estadísticas con la compra editada NO EJEC. (sin DSN) |
| 16 | Producto inactivo: se acepta y no se reactiva | `TestEditarRenglonProductoInactivoSeAcepta` (no consulta el estado) | unit | PASA. Que siga inactivo en base: NO EJEC. |
| 17 | Valores inválidos (cantidad 0, costo -1, precio 0): 422 con campo `items[i].x` | `TestEditarRenglonErrores`; `TestRenglonesValidaciones`; `TestPutCompra`; `TestQAPutRenglonValoresInvalidos`, `TestQAPutRenglonConsumoBorde` | unit / HTTP | PASA |
| 18 | Cambiar el producto de un renglón: 422 | `TestEditarRenglonErrores`; `TestQAPutRenglonValoresInvalidos` ("producto ajeno") | unit / HTTP | PASA |
| 19 | Venta simultánea contra una baja (serialización, nunca negativo) | `TestVentaSimultaneaContraBajaMySQL`; `TestTraducirConcurrencia`; `TestSalidasYAjustesUsanLecturaBloqueante` (usa lectura bloqueante) | MySQL / unit | Lógica de bloqueo PASA (unit). Concurrencia real NO EJEC. |
| 20 | Dos personas editan a la vez: 409 `version_desactualizada` | `TestQADosPersonasEditanALaVez`; `TestEditarRenglonErrores` | unit | PASA. Doble PUT real: NO EJEC. |
| 21 | Todo o nada | `TestEditarRenglonBloqueoPorConsumoNoEscribeNada`, `TestEditarRenglonPropagaFallasDeInventarioYAuditoria`; `TestEditarTodoONadaMySQL` | unit / MySQL | PASA (unit) / NO EJEC. (MySQL, el rollback real) |
| 22 | Auditoría de renglones MODIFICADO (cantidad 4 -> 6, costo 35.520 -> 36.000) | `TestRenglonesAuditoriaGolden`; `TestQATotalVersionYAuditoriaDeCantidadYCosto` | unit | PASA |

## Reglas transversales (RN-B)
| Regla | Caso | Estado |
|---|---|---|
| RN-B-01 "ya salieron N de M", mínimo = N | `TestObtenerCompraCalculaConsumido`; `minimoCantidad`/`validarRenglones` en `edicion-compra.spec.ts` | PASA (unit); UI en navegador NO EJEC. |
| RN-B-02 el 409 lista todos los rechazados | caso 7 | PASA |
| RN-B-03 movimiento sólo por la diferencia; sin cambio, sin movimiento | `TestEditarRenglonSinCambiosNoEscribe`; `TestRenglonesSinCambios`; "items sin cambios" en `TestPutCompra` | PASA |
| `precioVentaActual` correcto, una sola consulta al catálogo | `TestObtenerCompraTraePrecioVentaActual` | PASA |
| `consumido` en el detalle | `TestObtenerCompraCalculaConsumido` | PASA |

## Casos borde adicionales
| Caso | Esperado | Estado |
|---|---|---|
| Lote con stock 0: subir cantidad y cambiar costo | se acepta (+2) | PASA (`TestQALoteConsumidoPermiteSubirYCambiarCosto`) |
| Lote con stock 0: bajar a 3 | 409 "hasta 4 unidades" | PASA |
| Cabecera + renglones juntos | una versión, una fila de auditoría | PASA (`TestQACabeceraYRenglonesEnUnaSolaEdicion`) |
| Compra en versión 3 | versión 4, auditoría con `version` 4 | PASA |
| Renglón repetido / de otra compra | 422 `items[i].id` | PASA (HTTP) |
| Renglón sin `id` (nuevo) / lista sin un renglón | 422 `renglones_nuevos_no_disponibles` / `quitar_renglones_no_disponible` | PASA |
| Campo desconocido en el renglón, costo con 3 decimales, vencimiento no fecha, cantidad decimal | 422 `json_invalido` | PASA (el contrato real usa 422, no 400) |
| Cantidad o total enormes | 422 | **FALLA**: se acepta (BUG-001, baja) |
| Misma fecha pero otra hora, y created_at | ver observación OBS-1 en el reporte | Observación |

## Regresión
| Área | Caso | Estado |
|---|---|---|
| Ventas: salida FEFO (ahora con lectura bloqueante) | `TestIngresoYSalidaFEFO`, `TestSalidaSinStock`, `TestSalidasYAjustesUsanLecturaBloqueante`, `TestRevertirSalidas`, suites de ventas (`TestVenta*`, `TestAnularVenta`) | PASA (fakes) |
| Ajustes de stock (ahora bloquean lotes del producto) | `TestAjustes` | PASA (fakes) |
| SQL real de `LotesConStock ... FOR SHARE` con `GROUP BY/HAVING`, ventas contra MySQL | `TestVentasMySQL`, `TestVentasSobreCompraEditadaMySQL` | NO EJEC. (SKIP) |
| Alta de compra: último costo/precio sólo si es la más reciente por fecha | `TestRegistrarCompraSoloActualizaSiEsLaMasReciente` (alta con fecha vieja no pisa el costo), `TestRegistrarCompraRenglonesRepetidosUsaElUltimo`, `TestRegistrarCompra` | PASA |
| Cabecera (US-001a) | `TestEditarCompra*`, `TestPutCompra`, `TestRutaPutComprasRegistrada` | PASA |
| Migración 00005 (tipo `EDICION_COMPRA`) | `migrate_sqlite_test.go` | PASA en SQLite; MySQL NO EJEC. |
| Frontend | `npm test` (63 tests), `npm run build` | PASA |
