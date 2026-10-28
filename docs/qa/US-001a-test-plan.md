# Plan de pruebas — US-001a Editar la cabecera de una compra

Rama: `feature/edit-purchase` (sin commitear) · Fecha: 2026-10-27 · Autor: QA
Historia: `docs/product/stories/US-001a-editar-cabecera-compra.md` · Contrato: `docs/API.md` "Editar una compra"

## Alcance y entorno
- Backend Go (`backend/`), frontend Angular (`frontend/`), migración `00004_editar_compra` (MySQL y SQLite).
- Entorno disponible en esta corrida: Go y Node locales. **Sin MySQL de test** (no hay credenciales en `.env.example` ni en
  la documentación, y Docker no está disponible: el engine no responde). Por eso los casos "MySQL" y "API manual" quedan
  NO EJEC. No existe suite Playwright en el repo.
- Tipos: unit (dominio/aplicación con repositorio en memoria), integración MySQL (`MAGUI_TEST_MYSQL_DSN`), API (HTTP real),
  frontend unit (Vitest), e2e/manual (UI).

## Matriz criterio -> caso -> tipo -> estado
Estados: PASA, FALLA, NO EJEC. (no se pudo ejecutar, con motivo).

| # | Criterio Gherkin | Caso | Tipo | Estado |
|---|---|---|---|---|
| 1 | Corregir fecha, empresa y notas | `TestEditarCompraFeliz`, tests de dominio en `edicion_test.go` | unit | PASA |
| 1b | idem, persistido; renglones/total sin cambio; versión 2 | `TestEditarCabeceraCompraMySQL` (feliz) | integración MySQL | NO EJEC. (sin DSN, SKIP) |
| 1c | idem, vía `PUT /compras/{id}` | PUT con fecha, empresa y notas | API | PASA el enrutado (BUG-001 corregido); ejecución completa NO EJEC. (sin MySQL) |
| 1d | "Compra actualizada" | toast en `compras.page.ts` | manual UI | NO EJEC. (sin backend); sólo revisado en código |
| 2 | Formulario con renglones de solo lectura | revisión de `compra-form.component.ts` | manual UI | NO EJEC. |
| 3 | Historial: fecha/hora, origen WEB, motivo, cambios antes->después | `TestEditarCompraFeliz`; MySQL (`ed.Cabecera`) | unit / integración | PASA (unit) / NO EJEC. (MySQL) |
| 4 | Compra sin ediciones: "Sin ediciones", `ediciones` = `[]` | MySQL (`Ediciones != nil && len == 0`); `historial-ediciones.component.ts` | integración / UI | NO EJEC. |
| 5 | Quitar la empresa | tramo "Quitar la empresa" en MySQL; unit de `PlanificarEdicion` | unit / integración | PASA (unit) / NO EJEC. (MySQL) |
| 6 | Mover a otro período en estadísticas | edición de fecha + consulta de estadísticas | integración / API | NO EJEC. (no hay caso automatizado; ver Gaps) |
| 7 | Guardar sin cambios: misma versión, "No había cambios" | `TestEditarCompraSinCambiosNoEscribe`; `seAplicoEdicion` (spec frontend) | unit / frontend | PASA |
| 8 | Fecha futura -> 422 `fecha` | `TestEditarCompraErrores` | unit | PASA (HTTP: NO EJEC.) |
| 9 | Notas > 500 -> 422 `notas` | `TestEditarCompraErrores` | unit | PASA (HTTP: NO EJEC.) |
| 10 | Empresa inexistente -> 404 | `TestEditarCompraErrores`; MySQL | unit / integración | PASA (unit) / NO EJEC. (MySQL) |
| 11 | Compra inexistente -> 404 `compra_no_encontrada` | `TestEditarCompraErrores` | unit | PASA |
| 12 | Concurrencia: 409 `version_desactualizada` | `TestEditarCompraErrores`; MySQL (versión vieja + `ActualizarCompra`) | unit / integración | PASA (unit) / NO EJEC. (MySQL) |
| 12b | Concurrencia real (dos PUT simultáneos, `FOR UPDATE`) | dos clientes en paralelo | API / MySQL | NO EJEC. |
| 13 | Todo o nada si falla la auditoría | `TestEditarCompraFallaLaAuditoria`; MySQL con transacción real | unit / integración | PASA (unit) / NO EJEC. (MySQL) |

## Casos borde adicionales
| Caso | Esperado | Estado |
|---|---|---|
| `items` presente (incl. `[]`) | 422 `renglones_no_editables` | PASA (unit) / HTTP NO EJEC. |
| `items: null` o ausente | renglones sin cambios | revisión de código OK; HTTP NO EJEC. |
| `version` 0 / ausente | 422 `version` | PASA (unit) |
| `origen` inválido | 422 `origen`; vacío = WEB | PASA (unit) |
| `motivo` > 255 | 422 `motivo` | PASA (unit) |
| Instante ISO (fecha y hora) | conserva el instante, tolerancia +1 h | PASA (`TestEditarCompraInstanteISO`) |
| Mismo día que el original | conserva la hora, no cuenta como cambio | PASA (unit dominio) |
| Fecha "hoy" | queda la hora actual | PASA (unit dominio) |
| Cuerpo vacío / JSON inválido / campo desconocido | 422 `cuerpo_vacio` / `json_invalido` | NO EJEC. (HTTP) |
| Zona horaria (compra 23:00 ART, día UTC distinto) | `diaArgentino` correcto | PASA (spec frontend) |
| Compras previas a la migración (versión 1, `updated_at = created_at`) | editables, `editadaEn` null | NO EJEC. (requiere MySQL con datos previos) |
| Alta POST sigue funcionando (`version` 1, `ediciones` `[]`) | 201 | PASA (unit `compras_test`) / HTTP-MySQL NO EJEC. |
| Migración 00004 sube/baja en SQLite | OK | PASA (`migrate_sqlite_test.go`) |
| Migración 00004 sobre MySQL con datos | OK | NO EJEC. |
| Reenvío con la versión vieja (doble clic) | 409 | cubierto por caso 12 |
| Notas con emojis / 500 caracteres exactos | contados en runas | revisión de código OK (`RuneCountInString`) |

## Tests agregados por QA
- `backend/internal/compras/adapters/http/handler_test.go` — `TestRutaPutComprasRegistrada`: verifica que `PUT /compras/{id}`
  esté registrado en `Routes`. Pasa tras corregir BUG-001.

## Gaps de automatización recomendados
1. Test de integración HTTP (`httptest` + módulo real sobre MySQL temporal) para PUT/GET.
2. Caso de estadísticas por período tras cambiar la fecha (criterio 6).
3. Caso de concurrencia real con dos goroutines contra MySQL.
4. e2e Playwright de editar desde la lista (no hay infraestructura e2e en el repo).
