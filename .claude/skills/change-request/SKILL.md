---
name: change-request
description: Gestionar un pedido de cambio del cliente durante un proyecto en curso: análisis de impacto funcional y técnico, re-estimación, efecto en plazos y presupuesto, y aprobación antes de incorporarlo al backlog. Usar cuando el cliente pide algo que no estaba en el alcance aprobado.
---

# Pedido de cambio (change request)

## Flujo
1. **Registrar** en `docs/proposal/cambios/CR-NNN-slug.md` el pedido textual del cliente y la fecha.
2. **¿Está en alcance?** Comparar con `docs/proposal/propuesta.md` y PRD. Si es aclaración de algo incluido → historia normal, sin CR.
3. **Impacto funcional** (business-analyst): historias nuevas/afectadas, reglas de negocio, datos.
4. **Impacto técnico** (software-architect / tech-lead): módulos, contratos de API, migraciones, ADRs, riesgos.
5. **Estimación** (sales-engineer con skill estimate-and-quote): horas PERT, costo, corrimiento de fechas.
6. **Opciones al cliente:** (a) agregar con costo y nueva fecha, (b) cambiar por algo de igual tamaño que sale del alcance, (c) dejar para fase 2.
7. **Aprobación escrita** → recién ahí se agrega al backlog y al plan.

## Plantilla
```markdown
# CR-NNN — <título>
Fecha · Solicitado por · Estado: Analizando | Propuesto | Aprobado | Rechazado | Diferido
## Pedido (textual)
## Análisis de alcance
## Impacto funcional
## Impacto técnico
## Estimación (h, costo, días de corrimiento)
## Opciones
## Decisión del cliente (fecha y medio)
```
