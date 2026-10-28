---
name: api-contract
description: Diseñar contratos de API REST con enfoque API-first en OpenAPI 3.1 (recursos, rutas, DTOs, errores, paginación, auth, versionado) antes de implementar. Usar en fase de diseño o al agregar/modificar endpoints, dejando el contrato en docs/api/openapi.yaml.
---

# Contrato de API (OpenAPI first)

## Convenciones
- Recursos en plural y kebab-case: `/api/v1/sales-orders/{id}`. Acciones no-CRUD como subrecurso: `POST /invoices/{id}/authorize`.
- JSON en camelCase. Fechas ISO-8601 en UTC. Dinero como string decimal (`"1234.56"`) o entero en centavos — elegir uno y documentarlo.
- Métodos: GET (seguro), POST (crear/acción), PUT (reemplazo), PATCH (parcial), DELETE.
- Códigos: 200, 201 + `Location`, 204, 400 validación, 401, 403, 404, 409 conflicto, 422 regla de negocio, 429, 500.
- Errores en **RFC 9457 Problem Details**:
  ```json
  { "type": "https://api.ejemplo.com/errors/stock-insuficiente", "title": "Stock insuficiente",
    "status": 422, "detail": "El producto X tiene 2 unidades", "instance": "/api/v1/sales-orders",
    "errors": [{ "field": "items[0].quantity", "message": "Máximo 2" }] }
  ```
- Paginación: `?page=1&pageSize=20` → `{ data: [], meta: { page, pageSize, total } }` (o cursor si el volumen lo pide).
- Filtros y orden: `?status=active&sort=-createdAt`.
- Auth: `bearerAuth` (JWT). Documentar roles requeridos por operación en `description` o extensión `x-roles`.
- Idempotencia en POST críticos (pagos, facturas): header `Idempotency-Key`.
- Versionado en la URL (`/v1`); cambios incompatibles → `/v2` + ADR.

## Pasos
1. Por cada historia, listar operaciones necesarias.
2. Definir `components/schemas` (Create/Update/Response separados) con ejemplos reales.
3. Definir operaciones con `operationId`, tags por módulo, respuestas de error.
4. Validar: `npx @redocly/cli lint docs/api/openapi.yaml`.
5. Generar tipos para el frontend (`openapi-typescript`) y, en NestJS, mantener sincronía con `@nestjs/swagger` (test que compare el spec generado con el contrato).

## Checklist
- [ ] Cada criterio de aceptación tiene operación que lo soporta.
- [ ] Ningún endpoint devuelve entidades de la base directo (siempre DTO de respuesta).
- [ ] Listados paginados; límites máximos de `pageSize`.
- [ ] Errores documentados por operación.
- [ ] Ejemplos en request y response.
