---
name: implement-feature
description: Flujo estándar para implementar una historia de usuario o tarea técnica: leer contexto, crear rama, TDD, commits convencionales, verificación local y apertura de PR con plantilla. Usar por cualquier agente desarrollador (backend, frontend, desktop, integraciones, IA) al tomar una tarea.
---

# Implementar una feature

## 1. Contexto (no saltear)
- Historia `docs/product/stories/US-NNN*.md` y sus criterios Gherkin.
- Contrato `docs/api/openapi.yaml`, diseño `docs/ux/`, ADRs relacionados, modelo `docs/data/`.
- Código existente parecido: copiá el patrón del repo, no inventes uno nuevo.

## 2. Rama
```bash
git checkout main && git pull
git checkout -b feature/US-NNN-slug-corto
```

## 3. TDD por criterio de aceptación
1. Escribí el test que expresa el criterio (falla).
2. Implementación mínima para que pase.
3. Refactor con tests en verde.
Repetí por cada escenario, incluyendo errores y permisos.

## 4. Commits (Conventional Commits, en inglés)
```
feat(sales): add endpoint to create sales order

Refs: US-004
```
Tipos: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`, `ci`, `build`. Cambio incompatible: `BREAKING CHANGE:` en el cuerpo.

## 5. Verificación local
```bash
npm run lint && npm test && npm run build   # o los scripts equivalentes del repo
```
Si hay migraciones: correrlas desde cero y también `down` + `up`.
Revisá tu propio diff (`git diff main...HEAD`) buscando: console.log, código muerto, secretos, TODOs sin issue.

## 6. Pull Request
Título: `feat(sales): create sales order (US-004)`. Cuerpo:

```markdown
## Qué
## Por qué (US-NNN, ADR-NNNN)
## Cómo probarlo
1. …
## Criterios de aceptación
- [x] Escenario 1 — test: `sales.e2e-spec.ts › crea venta`
## Migraciones / variables de entorno nuevas
## Capturas (UI)
## Riesgos / notas para el reviewer
```

## 7. Entrega a la sesión principal
Devolvé: rama, resumen, archivos, cómo probar, pendientes. La sesión principal llama a code-reviewer, security-engineer y qa-engineer.
