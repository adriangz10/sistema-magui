---
name: code-review
description: Checklist y formato para revisar un pull request o un diff (corrección, diseño, tests, rendimiento, mantenibilidad y convenciones), clasificando hallazgos en bloqueantes, importantes y menores. Usar antes de mergear cualquier cambio.
---

# Code review

## Preparación
```bash
git fetch && git diff --stat main...HEAD
git diff main...HEAD
git log --oneline main..HEAD
```
Leé la historia y ADRs referenciados en el PR. Corré `lint`, `test`, `build`.

## Checklist
**Corrección**
- ¿Cumple todos los criterios de aceptación? ¿Hay test para cada uno?
- Casos borde: nulos, vacíos, límites, concurrencia, errores de terceros, zona horaria, redondeo de dinero.
- Transacciones donde hay escrituras múltiples; idempotencia en webhooks y reintentos.
- Manejo de errores: no se tragan excepciones; mensajes útiles; códigos HTTP correctos.

**Seguridad básica** (la profunda la hace security-engineer)
- Autorización por recurso, validación de entrada, sin secretos, sin datos sensibles en logs.

**Diseño**
- Respeta módulos/capas y ADRs. Sin lógica de negocio en controllers ni componentes de UI.
- Sin duplicación evitable; abstracciones justificadas (no prematuras).
- Nombres claros; funciones cortas con una responsabilidad.

**Datos**
- Migración incluida y reversible; índices para consultas nuevas; sin N+1.

**Tests**
- Prueban comportamiento, no implementación; deterministas; sin dependencias de orden ni de hora real.

**Frontend**
- Estados de carga/error/vacío; accesibilidad; sin suscripciones colgadas; tipos del contrato.

**Mantenibilidad**
- PR enfocado; sin código comentado, console.log ni TODO sin issue; docs/CHANGELOG actualizados si corresponde.

## Clasificación
- **Bloqueante:** bug, pérdida de datos, vulnerabilidad, criterio no cumplido, rompe build/tests, viola ADR.
- **Importante:** deuda clara, test faltante de caso relevante, performance evidente.
- **Menor:** legibilidad, nombres, sugerencias.

Cada bloqueante con: `archivo:línea`, escenario concreto que falla y sugerencia de corrección.
