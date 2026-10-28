---
name: write-prd
description: Redactar un PRD (Product Requirements Document) a partir de una propuesta aprobada o una idea de producto. Usar cuando hay que definir objetivos medibles, personas, alcance MVP, fuera de alcance, épicas, métricas y riesgos en docs/product/prd.md.
---

# PRD

## Plantilla (`docs/product/prd.md`)

```markdown
# PRD — <Producto> v<0.1> (<fecha>)

## 1. Resumen
Una frase: para <usuario> que <necesidad>, <producto> es <categoría> que <beneficio clave>.

## 2. Problema
Situación actual, dolor, evidencia (datos, citas del cliente).

## 3. Objetivos y métricas
| Objetivo | Métrica | Línea base | Meta | Cómo se mide |
|---|---|---|---|---|

## 4. Usuarios / personas
Por persona: rol, contexto de uso, dispositivo, objetivos, frustraciones.

## 5. Alcance
### MVP (Must)
Épicas con una línea de descripción: `EP-01 Gestión de clientes — ...`
### Should / Could
### Fuera de alcance (Won't, por ahora)

## 6. Requisitos no funcionales
Rendimiento, disponibilidad, offline, seguridad, privacidad (Ley 25.326), accesibilidad, navegadores/dispositivos, volumen de datos.

## 7. Dependencias e integraciones

## 8. Supuestos y restricciones

## 9. Riesgos

## 10. Hitos

## 11. Preguntas abiertas
```

## Reglas
- Objetivos SMART; si no hay línea base, poné "a medir en semana 1".
- El MVP debe poder usarse de punta a punta para al menos un proceso real.
- Cada épica tiene que poder trazarse a un objetivo.
- Al terminar, generá `docs/product/backlog.md` con las épicas priorizadas (MoSCoW) para que el business-analyst las baje a historias.
