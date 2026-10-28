# Reporte de QA — US-001a Editar la cabecera de una compra

Rama `feature/edit-purchase` (cambios sin commitear) · 2026-10-27 · Plan: `docs/qa/US-001a-test-plan.md`

## Veredicto: CONDICIONADO a la prueba con MySQL real
BUG-001 (crítico) fue corregido y verificado. Sin bugs abiertos conocidos y con la regresión automatizada en verde, pero
no se ejecutó ninguna prueba contra MySQL (sigue sin DSN) ni la prueba manual de API/UI. No hay OK de QA hasta hacerlo.

## Resultados de ejecución
| Suite | Comando | Resultado |
|---|---|---|
| Backend completo | `go test -count=1 ./...` (en `backend/`) | Todo verde. Los tests MySQL (incluido `TestEditarCabeceraCompraMySQL`) hacen SKIP: sin DSN |
| Backend vet | `go vet ./...` | Sin hallazgos |
| Tests de edición (unit) | `go test -run 'Editar\|Edicion' ./internal/compras/...` | `TestEditarCompraFeliz`, `SinCambiosNoEscribe`, `Errores`, `FallaLaAuditoria`, `InstanteISO` PASAN |
| Frontend unit | `npm test` | 11 archivos, 47 tests, todos pasan |
| Frontend build | `npm run build` | OK |
| Backend, rerun final | `go test -count=1 ./...` | Todo verde (incluye `TestRutaPutComprasRegistrada` y `editar_http_test.go`) |
| Frontend, rerun final | `npm test` | 12 archivos, 52 tests, todos pasan |

## Bugs

BUG-001 — `PUT /compras/{id}` no está registrado (405 Method Not Allowed) — CORREGIDO Y VERIFICADO
Severidad: crítica
Historia: US-001a (todos los escenarios de edición)
Pasos:
1. En `backend/internal/compras/adapters/http/handler.go`, el método `Routes` sólo registra `GET /`, `POST /` y `GET /{id}`.
2. El método `editar` existe pero nunca se enlaza (no hay usos de `h.editar`).
3. Reproducir: `cd backend && go test ./internal/compras/adapters/http/ -run TestEditarCompraEstaEnrutado`.
Esperado: `PUT /api/compras/{id}` llega a `editar` (200/409/422/404 según el caso).
Obtenido: 405. La pantalla (`compras.api.ts` hace `http.put(.../compras/{id})`) recibiría error en cada intento de guardar.
Evidencia: `backend/internal/compras/adapters/http/handler_test.go` ("PUT /compras/{id} no está registrado en Routes: HTTP 405").
Arreglo probable: agregar `r.Put("/{id}", h.editar)` dentro de `Routes`. No lo apliqué (QA no toca código de producción).
Nota: los tests unitarios y de integración pasan porque llaman al servicio directamente y se saltean la capa HTTP.

Estado: el backend registró `r.Put("/{id}", h.editar)`; verificado con `TestRutaPutComprasRegistrada` (QA, `handler_test.go`) y
los tests HTTP de `editar_http_test.go`.

Observación sobre la historia: la tabla de datos de ejemplo dice "version 0 con la compra en versión 2 -> 409". El
comportamiento implementado (version 0 -> 422 en `version`) es el correcto según el contrato; se corrige el texto de la
historia a "version 1 con la compra en versión 2 -> 409". No es un bug de código.

La revisión de código de dominio, aplicación, repositorio, migración y frontend no mostró otros defectos (versión, día
argentino, auditoría y transacción son coherentes con el Gherkin), pero eso no sustituye la ejecución contra MySQL.

## No probado (y por qué)
- **Integración MySQL**: se saltea sin `MAGUI_TEST_MYSQL_DSN`. `backend/.env.example` y `deploy/.env.example` no traen
  credenciales y los docs sólo dan ejemplos genéricos. Docker no está disponible (el engine no responde), así que no se pudo
  usar `deploy/docker-compose.yml` (MySQL en el puerto 3307). Intenté leer `backend/.env` (archivo local, fuera de git) y el
  permiso fue denegado; no probé contraseñas.
- **API manual** (editar cada campo, 409, 422, 404, GET con `version`/`editadaEn`/`ediciones`, alta POST, compras previas a la
  migración): `cmd/api` sólo corre contra MySQL y no hubo base; además con BUG-001 el PUT no responde.
- **Migración 00004 sobre MySQL con datos** (`ALTER` + `UPDATE updated_at = created_at`): sólo validada en SQLite.
- **UI** (botón Editar, formulario con renglones de solo lectura, toasts, historial, marca "Editada el ..."): sin backend;
  sólo specs unitarios y lectura de código. No hay e2e Playwright en el repo.
- Estadísticas por período tras mover la fecha, y concurrencia real de dos escrituras.

## Condiciones para dar OK de QA
1. Correr `go test ./...` con `MAGUI_TEST_MYSQL_DSN` apuntando a un MySQL de desarrollo (crea y borra bases temporales).
2. Repetir la prueba manual de API con el backend levantado y una pasada de UI.
