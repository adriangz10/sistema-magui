---
name: project-kickoff
description: Relevamiento inicial de un proyecto o pedido de cliente. Usar al arrancar un proyecto nuevo, ante un pedido vago o antes de estimar, para obtener problema, usuarios, procesos, integraciones, restricciones y criterios de éxito, y dejarlo en docs/proposal/brief.md.
---

# Project kickoff (relevamiento)

Objetivo: salir con un brief que permita estimar sin adivinar.

## 1. Preguntas de relevamiento
Agrupá las preguntas y hacé sólo las que el pedido no responde.

**Negocio**
- ¿Qué problema resuelve? ¿Cómo lo hacen hoy (planilla, papel, otro sistema)? ¿Cuánto les cuesta hoy?
- ¿Qué tiene que pasar para que el proyecto se considere un éxito? (métrica concreta)
- ¿Hay fecha límite real (temporada, inicio de año fiscal, evento)?

**Usuarios**
- ¿Quiénes lo usan, cuántos, con qué dispositivos y qué conectividad? ¿Hay roles con permisos distintos?
- ¿Nivel de manejo tecnológico?

**Funcionalidad**
- Los 3–5 procesos principales, paso a paso.
- Reportes o salidas que necesitan (tickets, PDFs, Excel, facturas).
- Datos existentes a migrar (formato, volumen, calidad).

**Integraciones**
- Facturación electrónica (ARCA/AFIP: tipo de contribuyente, puntos de venta), cobros (MercadoPago), mensajería (WhatsApp), Google Workspace, otros sistemas.
- ¿Tienen credenciales/cuentas de sandbox?

**Restricciones**
- Presupuesto aproximado, infraestructura existente, preferencia de hosting, requisitos legales (datos personales, menores).
- ¿Quién mantiene el sistema después? ¿Quiere soporte mensual?

## 2. Brief (`docs/proposal/brief.md`)

```markdown
# Brief — <Proyecto> (<fecha>)
## Cliente y contexto
## Problema actual
## Objetivo y métricas de éxito
## Usuarios y roles
## Procesos principales (actual → propuesto)
## Integraciones
## Datos y migración
## Restricciones (plazo, presupuesto, infra, legales)
## Propuesta de MVP (borrador)
## Riesgos detectados
## Preguntas abiertas
```

## 3. Cierre
Si quedan preguntas abiertas que cambian el alcance en más de un 20%, no estimes: devolvé las preguntas.
