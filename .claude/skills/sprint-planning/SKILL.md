---
name: sprint-planning
description: Planificar un sprint o iteración: objetivo, selección de historias por prioridad y capacidad, descomposición en tareas técnicas por rol, dependencias, orden de ejecución y riesgos. Usar al inicio de cada sprint o al replanificar, dejando docs/plan/sprint-NN.md.
---

# Planificación de sprint

## Entradas
`docs/product/backlog.md` (priorizado), historias en estado Ready, capacidad del equipo/agentes, velocidad previa (si existe), fechas comprometidas en `docs/plan/roadmap.md`.

## Pasos
1. **Objetivo del sprint:** una frase con valor de negocio ("El vendedor puede registrar una venta y emitir factura B").
2. **Capacidad:** horas disponibles × 0.8 (20% para imprevistos/bugs).
3. **Selección:** historias Ready en orden de prioridad hasta llenar capacidad; respetar dependencias.
4. **Descomposición** (con el tech-lead): por historia, tareas ≤ 1 día, cada una con rol/agente.
   Orden típico: migración/modelo → contrato API → backend → integración → frontend → tests e2e → docs.
5. **Riesgos y supuestos** del sprint.

## Plantilla (`docs/plan/sprint-NN.md`)

```markdown
# Sprint NN — <fechas>
Objetivo:
Capacidad: X h (Y h comprometidas)

## Historias
| ID | Título | Pts/h | Agentes | Estado |

## Tareas
| # | Historia | Tarea | Agente | Depende de | Est. | Estado |
|---|---|---|---|---|---|---|
| 1 | US-004 | Migración tabla sales | database-engineer | — | 2h | ☐ |

## Orden de ejecución (para la sesión principal)
1. ...
## Riesgos
## Definition of Done
- Código mergeado a main vía PR aprobado (code-reviewer + security-engineer)
- Tests unit + integración + e2e del camino feliz en verde
- Criterios de aceptación verificados por QA y aceptados por PO
- Docs actualizadas (API, README, CHANGELOG unreleased)

## Retro (al cierre)
Funcionó / No funcionó / Acción para el próximo sprint
```
