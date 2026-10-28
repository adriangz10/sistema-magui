---
name: user-stories
description: Escribir historias de usuario con criterios de aceptación en Gherkin, reglas de negocio y Definition of Ready, a partir de un PRD o épicas. Usar para armar o refinar el backlog en docs/product/stories/.
---

# Historias de usuario

## Plantilla (`docs/product/stories/US-NNN-slug.md`)

~~~~markdown
# US-NNN — <título en verbo>
Épica: EP-NN · Prioridad: Must|Should|Could · Estimación: <puntos> · Estado: Borrador|Ready|En curso|Hecha

## Historia
Como <rol>, quiero <acción>, para <beneficio>.

## Reglas de negocio
- RN-NNN: ...

## Criterios de aceptación
```gherkin
Escenario: <camino feliz>
  Dado ...
  Cuando ...
  Entonces ...

Escenario: <error / validación>
  ...

Escenario: <permiso insuficiente>
  ...
```

## Datos de ejemplo
| campo | valor válido | valor inválido |

## Notas de UX
Link a `docs/ux/pantallas/...`

## Dependencias
US-..., integraciones, datos

## Fuera de alcance de esta historia
~~~~

## Checklist INVEST / Definition of Ready
- [ ] Independiente (o dependencia explícita)
- [ ] Negociable: describe el qué, no el cómo técnico
- [ ] Valiosa para un usuario o el negocio
- [ ] Estimable
- [ ] Small: entra en ≤ 3 días de un dev; si no, partir (por flujo, por regla, por rol, por CRUD)
- [ ] Testeable: cada criterio es verificable
- [ ] Caminos de error y permisos cubiertos
- [ ] Datos de ejemplo concretos (montos, fechas, formatos AR: DNI, CUIT, $ 1.234,56)

## Reglas
- Una historia = un archivo. Numeración correlativa, nunca reutilizar IDs.
- Actualizá `docs/product/backlog.md` con cada historia nueva.
