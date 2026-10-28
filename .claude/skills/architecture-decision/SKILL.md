---
name: architecture-decision
description: Registrar una decisión de arquitectura como ADR (Architecture Decision Record) con contexto, alternativas, decisión y consecuencias. Usar cada vez que se elige tecnología, patrón, proveedor, estrategia de datos/despliegue o se cambia una decisión anterior.
---

# ADR

## Cuándo escribir uno
Elección de framework/ORM/base, monolito vs servicios, auth, multi-tenant, offline/sync, proveedor externo
(facturación, pagos, LLM), estrategia de despliegue, cualquier cosa cara de revertir o que alguien va a preguntar "¿por qué hicimos esto?".

## Archivo
`docs/architecture/adr/NNNN-titulo-en-kebab.md` (numeración correlativa de 4 dígitos). Mantené `docs/architecture/adr/README.md` con el índice.

## Plantilla

```markdown
# NNNN. <Título en forma de decisión: "Usar PostgreSQL con pgvector para RAG">

- Estado: Propuesto | Aceptado | Reemplazado por NNNN | Obsoleto
- Fecha: AAAA-MM-DD
- Decisores: <roles>
- Historias relacionadas: US-...

## Contexto
Problema, fuerzas en juego, requisitos de calidad relevantes, restricciones (equipo, costo, plazo, cliente).

## Alternativas consideradas
### A. <opción>
Pros / contras / costo / riesgo
### B. <opción>
...

## Decisión
Elegimos <X> porque <razones ligadas al contexto>.

## Consecuencias
- Positivas:
- Negativas / deuda aceptada:
- Qué hay que hacer ahora (tareas):
- Cuándo revisarla (disparadores):
```

## Reglas
- Mínimo dos alternativas reales (no hombres de paja).
- Nunca editar un ADR aceptado para cambiar la decisión: crear uno nuevo y marcar el viejo como "Reemplazado por".
- Ver precios, límites y versiones vigentes en documentación oficial antes de decidir sobre un proveedor.
