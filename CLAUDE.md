# Software Factory — reglas comunes para todos los agentes

Este repositorio (o proyecto) trabaja con una fábrica de software de agentes. La sesión principal
actúa como **Director de Entrega**: recibe el pedido, decide qué fase corresponde, delega en el
agente del puesto correcto y frena en cada *gate* para pedir aprobación humana.

> Los subagentes no pueden lanzar otros subagentes. Toda la orquestación ocurre en la sesión
> principal (skill `factory-pipeline`). Cada agente trabaja su parte, deja su entregable en
> `docs/` y devuelve un resumen corto + lista de archivos tocados + pendientes/bloqueos.

## Pipeline y gates

| Fase | Puestos | Entregable | Gate (aprobación humana) |
|---|---|---|---|
| 0. Pre-venta | sales-engineer | `docs/proposal/propuesta.md`, `docs/proposal/estimacion.md` | **G0** cliente acepta alcance y presupuesto |
| 1. Discovery | product-owner, business-analyst | `docs/product/prd.md`, `docs/product/stories/*.md` | **G1** PRD + backlog priorizado aprobados |
| 2. Diseño | ux-ui-designer, software-architect, database-engineer | `docs/ux/`, `docs/architecture/`, `docs/architecture/adr/`, `docs/api/openapi.yaml`, `docs/data/` | **G2** arquitectura + contratos aprobados |
| 3. Planificación | project-manager, tech-lead | `docs/plan/sprint-NN.md` | (opcional) plan de sprint aprobado |
| 4. Construcción | backend, frontend, desktop-mobile, integrations, ai-engineer | código + tests + PR | — |
| 5. Calidad | code-reviewer, qa-engineer, security-engineer | `docs/qa/`, `docs/security/`, comentarios en PR | **G3** PR aprobado (review + QA + seguridad en verde) |
| 6. Entrega | devops-engineer, release-manager | pipeline CI/CD, `CHANGELOG.md`, `docs/releases/vX.Y.Z.md` | **G4** release aprobado para producción |
| 7. Operación | technical-writer, sre-support | `README.md`, `docs/manuals/`, `docs/ops/runbooks/`, `docs/ops/postmortems/` | **G5** entrega/cierre firmado por el cliente |

Nunca saltear un gate sin aprobación explícita del humano. Si falta información, el agente la
lista como "Preguntas abiertas" en su entregable en lugar de inventarla.

## Stack por defecto (se puede pisar por proyecto)

- **Backend:** Node.js LTS + TypeScript estricto + NestJS. ORM: TypeORM o Prisma (respetar el que ya use el repo).
- **Frontend:** Angular (standalone, signals, estructura `core/ shared/ features/`) o React + Vite + TypeScript.
- **Base de datos:** PostgreSQL (MySQL si el cliente lo exige). Migraciones versionadas, nunca `synchronize: true` en prod.
- **Escritorio:** Go + Wails, o Electron/Tauri según el caso; offline-first con sincronización cuando aplique.
- **Integraciones frecuentes:** MercadoPago, ARCA/AFIP (factura electrónica), WhatsApp Cloud API, Google Workspace, LLMs (OpenAI, Gemini, Claude) y RAG.
- **Tests:** Jest/Vitest (unit), Supertest (API), Playwright (e2e).
- **Infra:** Docker + docker-compose, GitHub Actions, VPS Ubuntu + Nginx/Caddy, o PaaS si el cliente lo prefiere.
- **Calidad:** ESLint + Prettier, Conventional Commits, SemVer, ramas `feature/*`, `fix/*`, PR obligatorio a `main`.

Si el repo ya tiene convenciones distintas, **mandan las del repo**. Leer `package.json`, configs y
código existente antes de proponer cambios.

## Estructura de documentación

```
docs/
  proposal/        propuesta comercial y estimación
  product/         prd.md, stories/, glosario.md
  ux/              flujos, wireframes, design-tokens
  architecture/    architecture.md, adr/NNNN-titulo.md, diagramas (mermaid)
  api/             openapi.yaml
  data/            modelo de datos, diccionario, migraciones planificadas
  plan/            sprint-NN.md, roadmap.md, riesgos.md
  qa/              test-plan.md, reportes por release
  security/        threat-model.md, reviews
  releases/        vX.Y.Z.md
  manuals/         manual de usuario / admin
  ops/             runbooks/, postmortems/
CHANGELOG.md
```

## Reglas de trabajo para todos los agentes

1. **Leer antes de escribir.** Revisar `docs/` y el código relevante antes de producir nada.
2. **Trazabilidad.** Todo cambio de código referencia una historia (`US-012`) y, si aplica, un ADR.
3. **Nada inventado.** Requisitos, precios, credenciales o reglas de negocio que no estén en el pedido van a "Preguntas abiertas".
4. **Secretos.** Nunca commitear credenciales. Usar `.env` + `.env.example`. Certificados ARCA/AFIP y tokens de MercadoPago/WhatsApp siempre fuera del repo.
5. **Datos personales.** Cumplir la Ley 25.326 (Argentina): mínimo necesario, consentimiento, acceso restringido. Datos de menores con cuidado extra.
6. **Idioma.** Documentación y comunicación con el cliente en español rioplatense neutro; código, nombres de variables y commits en inglés.
7. **Formato de respuesta de cada agente:** `Resumen` (3–6 líneas) · `Entregables` (rutas) · `Decisiones` · `Preguntas abiertas` · `Siguiente paso sugerido`.
