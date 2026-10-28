---
name: estimate-and-quote
description: Estimación de esfuerzo y armado de propuesta comercial para un proyecto de software. Usar después del relevamiento para desglosar el trabajo por módulos y roles, estimar con PERT, calcular presupuesto, plazos, supuestos, exclusiones y riesgos, y redactar la propuesta para el cliente.
---

# Estimación y propuesta

## 1. Desglose (WBS)
Listá módulos → funcionalidades. Agregá siempre las tareas que se olvidan:
setup del proyecto, autenticación/roles, ABM básicos, reportes, migración de datos, integraciones,
QA, despliegue, documentación, capacitación, gestión del proyecto (10–15%), garantía post-entrega.

## 2. Estimación de 3 puntos (PERT)
Por ítem: optimista (O), más probable (M), pesimista (P) en horas.
`E = (O + 4M + P) / 6` · `σ = (P − O) / 6`.
Total con buffer de riesgo: `ΣE + 1.65 × √(Σσ²)` (≈ 95% de confianza). Riesgos altos (integración sin sandbox, datos sucios a migrar) suman buffer explícito.

Tabla en `docs/proposal/estimacion.md`:

| Módulo | Ítem | Rol | O | M | P | E |
|---|---|---|---|---|---|---|

## 3. Presupuesto
- Tarifa por hora por rol desde `docs/proposal/tarifas.md`; si no existe, usá `[TARIFA_HORA]` y preguntá.
- Mostrá: total por fase, forma de pago (ej. 40% inicio / 30% entrega MVP / 30% cierre), moneda.
- En ARS: cláusula de actualización (por índice acordado o tipo de cambio) y vigencia de la propuesta (ej. 15 días).
- Opcional: abono mensual de mantenimiento (horas incluidas, SLA de respuesta, hosting).

## 4. Propuesta (`docs/proposal/propuesta.md`)

```markdown
# Propuesta — <Proyecto>
## Entendimiento del problema
## Solución propuesta (en lenguaje del cliente)
## Alcance
### Fase 1 — MVP
### Fases siguientes (opcionales)
## Exclusiones (qué NO incluye)
## Supuestos (qué tiene que aportar el cliente)
## Cronograma (hitos)
## Inversión y forma de pago
## Garantía y soporte
## Riesgos y cómo los manejamos
## Próximos pasos
```

## Reglas
- Nunca presentes un único número sin rango ni supuestos.
- Los supuestos del cliente (entregar datos, accesos, validar a tiempo) condicionan las fechas: escribilo.
- Cambios de alcance posteriores se cotizan aparte.
