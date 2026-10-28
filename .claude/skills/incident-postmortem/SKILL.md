---
name: incident-postmortem
description: Gestionar un incidente de producción (triage, mitigación, comunicación) y escribir el postmortem sin culpables con línea de tiempo, causa raíz (5 porqués), impacto y acciones correctivas. Usar ante caídas, errores masivos, pérdida o corrupción de datos o fallas de integraciones.
---

# Incidentes y postmortem

## Durante el incidente
1. **Declarar:** severidad (SEV1 servicio caído o pérdida de datos · SEV2 función crítica degradada · SEV3 menor con workaround).
2. **Bitácora** con hora de cada observación y acción (sirve para el postmortem).
3. **Mitigar primero:** rollback al tag anterior, desactivar feature, reiniciar, escalar, workaround manual para el cliente.
4. **Comunicar al cliente** (SEV1/2): qué está pasando, qué funciona y qué no, próxima actualización en X min.
5. **Verificar recuperación** con el mismo síntoma que se detectó.

## Diagnóstico
`git log` desde el último deploy · logs alrededor de la hora de inicio · métricas (CPU, memoria, disco, conexiones a DB) ·
estado de terceros (ARCA/AFIP, MercadoPago, Meta) · cambios de configuración/certificados vencidos.

## Postmortem — `docs/ops/postmortems/AAAA-MM-DD-titulo.md`
```markdown
# Postmortem: <título>
Fecha · Severidad · Duración · Autor

## Resumen (3 líneas)
## Impacto
Usuarios/clientes afectados, funciones, datos, dinero.
## Línea de tiempo (hora local AR)
| Hora | Evento |
## Causa raíz
5 porqués hasta llegar a una causa sistémica (proceso/herramienta), no a una persona.
## Qué funcionó / qué no
## Acciones
| Acción | Tipo (prevenir/detectar/mitigar) | Dueño | Fecha | Estado |
## Lecciones
```

## Reglas
- Sin culpables: "el deploy no tenía verificación de migración", no "X se olvidó".
- Toda acción tiene dueño y fecha; las de código se convierten en historias/bugs.
- Si el bug no tenía test, la primera acción es el test que lo reproduce.
