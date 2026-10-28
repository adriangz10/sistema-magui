---
name: test-strategy
description: Definir la estrategia y el plan de pruebas de un proyecto o release (pirámide de tests, matriz historia-casos, datos y entornos) y escribir tests automatizados unitarios, de integración/API y e2e con Playwright. Usar al planificar QA, al cerrar una historia o antes de un release.
---

# Estrategia de pruebas

## Pirámide
- **Unit (muchos):** lógica de dominio, servicios, validadores, utilidades de cálculo (montos, impuestos, fechas). Jest/Vitest. Rápidos, sin I/O.
- **Integración / API (bastantes):** endpoints reales con base de test en Docker (Supertest). Cubren criterios de aceptación y permisos.
- **Contrato:** respuestas de terceros grabadas (ARCA/AFIP, MercadoPago) para los adaptadores.
- **E2E (pocos):** Playwright, sólo caminos críticos de negocio (login, venta, cobro, factura, reporte).
- **No funcionales según PRD:** carga (k6), accesibilidad (axe con Playwright), seguridad (security-engineer).

## Plan (`docs/qa/test-plan.md`)

```markdown
# Plan de pruebas — <proyecto/release>
## Alcance y fuera de alcance
## Entornos y datos (seeds, usuarios de prueba por rol)
## Matriz de trazabilidad
| Historia | Criterio | Caso | Tipo | Automatizado | Archivo |
## Casos borde transversales
## Criterios de entrada / salida
## Riesgos de calidad
```

## Casos borde que siempre se prueban
Vacíos y nulos · límites (0, 1, máximo, máximo+1) · decimales y redondeo de dinero · fechas (fin de mes, cambio de año, zona horaria AR) ·
caracteres especiales y ñ/acentos · duplicados / doble click / doble envío · concurrencia (dos usuarios editan lo mismo) ·
permisos (otro rol, otro tenant, recurso ajeno) · red caída / timeout de tercero · sesión vencida.

## Playwright — convenciones
- `tests/e2e/<feature>.spec.ts`, fixtures para login por rol, datos creados por API en `beforeEach`.
- Selectores `getByRole`, `getByLabel`, `getByTestId` (nunca CSS frágil); sin `waitForTimeout`.
- Trace y video `on-first-retry`; correr en CI con la app levantada vía docker-compose.

## Salida
Reporte en `docs/qa/reporte-<id>.md`: resumen, casos ejecutados/pasados/fallados, bugs (formato BUG-NNN), veredicto OK de QA / NO OK.
