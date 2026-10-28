# Plan de pruebas — US-001c Agregar y quitar renglones de una compra

Rama: `feature/edit-purchase` (sin commitear) · Fecha: 2026-10-27 · Autor: QA
Historia: `docs/product/stories/US-001c-agregar-quitar-renglones-compra.md` · Reglas: `US-001-editar-compra.md` · Contrato: `docs/API.md` "Editar una compra" · Diseño: `docs/architecture/editar-compra.md`

## Alcance y entorno
- Backend Go: dominio (`PlanificarEdicion`, `VerificarConsumo`, `ProductosAfectados`), aplicación (`EditarCompra`, `actualizarCostos`), repositorio MySQL (`QuitarItems`, `AgregarItems`, `CompraMasReciente`, filtros `quitado_en IS NULL`), migración `00006`, frontend Angular (`edicion-compra.ts`, `compra-form.component.ts`, `historial-ediciones.component.ts`).
- Entorno: Go y Node locales. **Sin MySQL de test ni Docker** (sin DSN). Los `*MySQL` hacen SKIP. Sin Playwright en el repo.
- Tipos: unit (dominio/aplicación con repositorios en memoria), HTTP (router chi + fakes), integración MySQL, frontend unit (Vitest), revisión de código.
- Tests de QA nuevos (archivos nuevos, prefijo `TestQA`/`QA`):
  - `backend/internal/compras/application/agregar_quitar_qa_test.go`
  - `frontend/src/app/features/compras/edicion-compra.qa.spec.ts`

Estados: PASA, NO EJEC. (con motivo). Con fakes se valida la lógica, no el SQL.

## Matriz criterio Gherkin -> caso -> tipo -> estado
| # | Escenario | Caso | Tipo | Estado |
|---|---|---|---|---|
| 1 | Agregar un renglón (lote nuevo, COMPRA +2, total 193.080, costo y precio si es la más reciente, historial COMPRA) | `TestAgregarRenglonCreaLoteYMovimientoCompra`; `TestAgregarRenglon` (dominio) | unit | PASA. Lote/movimiento/`precio_historial` reales: `TestAgregarYQuitarRenglonesMySQL` NO EJEC. |
| 1b | Total con decimales | `TestQATotalConDecimalesAlAgregar` (3 x 1.234,56) | unit | PASA |
| 2 | Mismo producto en otro renglón (lote independiente) | `TestAgregarMismoProductoEnOtroRenglonEsLoteIndependiente`; `TestAgregarMismoProductoEnOtroRenglon` | unit | PASA |
| 2b | Dos nuevos del mismo producto: gana el último (costo y precio) | `TestQAAgregarDosRenglonesNuevosDelMismoProductoGanaElUltimo` | unit | PASA |
| 3 | Quitar sin consumo (EDICION_COMPRA -10, total 142.080, lote queda en historial) | `TestQuitarRenglonSinConsumo`; `TestQuitarRenglon` | unit | PASA. Baja lógica en SQL: NO EJEC. (MySQL) |
| 3b | Quitar con consumo 0 aunque el lote tenga ajuste manual positivo: baja sólo lo del renglón | `TestQuitarConAjustePositivoDescuentaSoloLaCantidadDelRenglon`; `TestVerificarConsumoQuitar` (stock 12 de 10) | unit | PASA |
| 4 | Quitar renglón consumido: 409 `compra_consumida`, `quitados[id]`, nada cambia | `TestQuitarRenglonConsumidoRechazaTodoJunto`; `TestPutAgregarYQuitarRenglones` ("quitar con consumo", HTTP) | unit / HTTP | PASA |
| 4b | 409 lista todos los quitados rechazados juntos y no agrega el nuevo | `TestQuitarRenglonConsumidoRechazaTodoJunto` (dos quitados + uno agregado válido) | unit | PASA |
| 4c | Consumo de una sola unidad: mensaje sin "bajar hasta" | `TestQAQuitarRenglonDeUnaUnidadConsumida` | unit | PASA |
| 5 | No dejar la compra vacía: 422 "Agregá al menos un producto" | `TestNoSePuedeDejarLaCompraVacia`; `TestNoQuedaCompraVaciaNiMasDe300`; HTTP "quitar todos" | unit / HTTP | PASA |
| 6 | Máximo 300 renglones | `TestNoQuedaCompraVaciaNiMasDe300` (301 nuevos); `TestQATope300RenglonesExistentes` (300 existentes + 1 = 422 sin escribir; quitar 1 + agregar 1 = 300 ok) | unit | PASA |
| 7 | Combo: 422 `producto_no_comprable` | `TestAgregarProductoInexistenteComboYNadaSeAplica` (tipo `TipoInvalido`); el mapeo tipo -> 422 es de `httpx` | unit | PASA (el status por HTTP real del combo no tiene test: el fake HTTP acepta cualquier producto) |
| 8 | Producto inexistente: 404 `producto_no_encontrado`, nada se aplica | ídem (con un cambio válido en otro renglón; `nadaEscrito`) | unit | PASA |
| 9 | Quitar la más reciente: el costo vuelve al de la otra compra, el precio no cambia | `TestQuitarRecalculaElUltimoCostoConLaNuevaMasReciente`; `TestQuitarNuncaRevierteElPrecio` | unit | PASA |
| 9b | Quitar 1 de 2 renglones del mismo producto: costo del que queda (catálogo con el costo del quitado) | `TestQAQuitarSegundoRenglonDelMismoProductoVuelveAlCostoDelQueQueda` (el del desarrollador no lo ejercitaba) | unit | PASA |
| 9c | Quitar en una compra que ya no es la más reciente: catálogo intacto | `TestQAQuitarEnCompraQueNoEsLaMasRecienteNoTocaElCosto` | unit | PASA |
| 10 | Quitar y no hay otra compra del producto: el costo se mantiene | `TestQuitarRenglonSinConsumo` (sin costos emitidos) | unit | PASA |
| 11 | Cambiar fecha a más vieja: recálculo a la otra compra, precio intacto | `TestCambiarFechaCambiaCualEsLaMasReciente`; `TestCambiarFechaRecalculaUltimoCostoMySQL` | unit / MySQL | PASA (unit) / NO EJEC. (MySQL) |
| 12 | Cambiar fecha a la más reciente | `TestCambiarFechaALaMasReciente` (catálogo con costo de C0 -> pasa a 35.520) | unit | PASA |
| 12b | Fecha sin otra compra: costo se mantiene; fecha + quitar en la misma edición | `TestCambiarFechaSinOtraCompraMantieneElCosto`; `TestQACambioDeFechaMasQuitarEnLaMismaEdicion` | unit | PASA |
| 13 | Agregar y quitar en la misma edición: no hay lote ni movimiento (RN-C-03) | Es de UI: un nuevo quitado se descarta y nunca viaja. `compra-form` filtra el nuevo; `QA: un renglón nuevo ... no deja cambios`; el backend no recibe el renglón | unit front | PASA (lógica); clic real en navegador NO EJEC. |
| 13b | Quitar un original y agregar el mismo producto | `TestQAQuitarYAgregarMismoProductoEnUnaEdicion` (-10 sólo cantidad, +3 nuevo, costo 2.500, total 149.580) | unit | PASA |
| 13c | Quitar todos los originales con un nuevo | `TestQAQuitarTodosLosOriginalesAgregandoUnoNuevo` | unit | PASA |
| 14 | Todo o nada (agregar válido + quitar consumido) | `TestQuitarRenglonConsumidoRechazaTodoJunto`; `TestQuitarFallaEnInventarioNoDejaRastroEnLaAuditoria`; rollback real: `TestAgregarYQuitarRenglonesMySQL` | unit / MySQL | PASA (unit) / NO EJEC. (rollback real) |
| 15 | Dos personas a la vez: 409 `version_desactualizada` | `TestQAVersionVencidaConAgregarYQuitarNoEscribe` | unit | PASA. Doble PUT real NO EJEC. |
| 16 | Auditoría AGREGADO/QUITADO | `TestAgregarRenglonCreaLote...`; `TestAgregarYQuitarCombinadosConCostoYPrecio`; `TestQAAuditoriaJSONdeAgregadoYQuitado` (JSON: sólo cantidad y costo, `despues` null) | unit | PASA |
| 17 | Producto inactivo: quitar se acepta; agregar no se ofrece | `TestQuitarProductoInactivoSeAcepta`; el buscador pide `estado: 'activos'` (revisión de `buscador-producto.component.ts`) | unit / revisión | PASA. El backend sí acepta agregar un inactivo por API (ver OBS-3). |
| 18 | Renglón quitado ausente en detalle, listado y "más reciente" | `TestQADetalleTrasQuitarNoMuestraElRenglon` (fake); SQL revisado: `quitado_en IS NULL` en el listado (resumen/unidades), detalle, `ObtenerParaEditar`, `ActualizarItems`, `CompraMasReciente`; no hay otro SQL sobre `compra_item` en el backend | unit / revisión | PASA (unit); SQL NO EJEC. |
| 19 | Reenviar id de un renglón quitado: 422 `items[i].id` | `TestQuitarRenglonYaQuitadoNoSePuedeReenviar`; `TestRenglonQuitadoNoSePuedeReenviar` | unit | PASA |
| 20 | Precio nunca se revierte | `TestQuitarNuncaRevierteElPrecio` | unit | PASA |

## Frontend
| Caso | Test | Estado |
|---|---|---|
| Nuevos sin `id` al final, quitados omitidos | `manda los nuevos sin id al final y omite los quitados` | PASA |
| Mapeo de errores por índice enviado (con quitados y nuevos) y `quitados[id]` por id | `mapea items[i]...`; `un nuevo puesto antes...`; QA: índice fuera de rango; dos nuevos + un quitado | PASA |
| 409 pintado en el renglón correcto con quitados | `409 con items[i] y quitados[id] se pinta en el renglón correcto aunque haya quitados` (componente) | PASA |
| Deshacer quitar | `quitar marca "Se quitará", ... deshacer lo revierte` (componente); `detecta cambios por agregar, quitar y deshacer` | PASA |
| Quitar deshabilitado con consumo / último renglón | `Quitar está deshabilitado...`; `no permite quitar el último renglón activo`; QA | PASA |
| Tope 300 y vacío en pantalla (quitados no cuentan) | `no deja la compra sin renglones...`; QA 300 | PASA |
| Total sin quitados y con decimales; resumen de efectos | QA | PASA |
| Historial QUITADO/AGREGADO sin `precioVentaNuevo`/`vencimiento`: sin "undefined"/"null" | `describe renglones agregados y quitados del historial`; QA (3 casos) | PASA (lógica); renderizado del template NO EJEC. en navegador |
| Errores `producto_no_comprable` / `producto_no_encontrado` mapeados a mensaje | revisión de `compra-form.component.ts` | Revisado, sin spec propio |

## Regresión
| Área | Caso | Estado |
|---|---|---|
| Cabecera (US-001a) | `TestEditarCompra*`, `TestPutCompra`, `TestRutaPutComprasRegistrada` | PASA |
| Editar renglones y consumo (US-001b) | `TestEditarRenglon*`, `TestVerificarConsumo`, `TestQA*` de US-001b, `editar_renglones_qa_http_test.go` | PASA |
| Alta de compra | `TestRegistrarCompra*` | PASA |
| Ventas / inventario (lectura bloqueante, ajustes) | suites de `ventas` e `inventario` | PASA (fakes) |
| Migraciones | `migrate_sqlite_test.go` (incluye `00006`) | PASA en SQLite; MySQL NO EJEC. |
| Frontend | `npm test` (13 archivos, 87 tests), `npm run build` | PASA |
