---
name: technical-docs
description: Escribir y mantener documentación técnica y de usuario (README, onboarding de desarrolladores, manual de usuario, manual de administrador, runbooks, referencia de API) verificada contra el código real. Usar al cerrar un release, al entregar un proyecto o cuando la documentación quedó desactualizada.
---

# Documentación

## README.md (plantilla)
```markdown
# <Proyecto>
Una línea: qué es y para quién.

## Requisitos
Node X, Docker, …

## Arranque rápido
cp .env.example .env
docker compose up -d db
npm ci
npm run migration:run && npm run seed
npm run start:dev

## Scripts
| Script | Qué hace |

## Variables de entorno
| Variable | Obligatoria | Ejemplo | Descripción |

## Estructura del proyecto
## Tests
## Despliegue → docs/ops/runbooks/deploy.md
## Documentación → docs/
```

## Manual de usuario — `docs/manuals/manual-usuario.md`
- Organizado por **tareas** ("Registrar una venta", "Cerrar la caja"), no por pantallas.
- Paso a paso numerado, segunda persona, voseo, botones en *cursiva*.
- Capturas o `![captura: descripción]` como placeholder.
- Sección "Si algo sale mal": mensajes de error comunes → qué hacer.
- Preguntas frecuentes.

## Manual de administrador — `docs/manuals/manual-admin.md`
Usuarios y roles, configuración, integraciones (dónde se cargan credenciales, cómo renovar certificados), backups, a quién contactar.

## Onboarding dev — `docs/onboarding.md`
Arquitectura en 1 página (link a C4), convenciones, flujo de ramas/PR, cómo agregar un módulo, dónde están los ADRs.

## Reglas
- Ejecutá o leé cada comando antes de documentarlo; tabla de variables generada desde `.env.example`.
- Fecha y versión del sistema al pie de cada documento.
- Si encontrás documentación desactualizada en otro archivo, corregila o listala.
