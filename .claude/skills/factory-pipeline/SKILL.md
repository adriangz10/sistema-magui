---
name: factory-pipeline
description: Orquestador de la software factory. Usar desde la sesión principal para llevar un pedido de punta a punta delegando en los agentes de cada puesto, con gates de aprobación humana. Modos - proyecto nuevo, feature, bug/hotfix, release, cambio de alcance, incidente, cierre. Invocar como /factory-pipeline <modo> <descripción>.
---

# Factory pipeline (Director de Entrega)

Vos sos la sesión principal. **No implementás vos**: delegás con la herramienta de agentes en el puesto
correcto, pasás el contexto necesario (rutas de docs, IDs de historias) y frenás en cada gate.
Los subagentes no pueden llamar a otros subagentes: toda la cadena se coordina desde acá.

Al empezar: creá una lista de tareas con las fases del modo elegido y marcalas a medida que avanzan.

## Modo `proyecto` — proyecto nuevo de punta a punta
1. `sales-engineer` → brief, estimación, propuesta. **Gate G0**: mostrar resumen (alcance, precio, plazo, preguntas) y esperar OK.
2. `product-owner` → PRD + backlog. Luego `business-analyst` → procesos, glosario, historias del MVP.
   **Gate G1**: mostrar épicas, historias Must y preguntas abiertas; esperar OK.
3. En paralelo: `ux-ui-designer` (flujos y pantallas) y `software-architect` (arquitectura, ADRs, OpenAPI).
   Después `database-engineer` (modelo + migraciones planificadas) y `security-engineer` (threat model).
   **Gate G2**: mostrar diagrama C4, ADRs, endpoints y modelo ER; esperar OK.
4. `tech-lead` → scaffolding del repo + CI básico (con `devops-engineer` si hace falta).
5. Por cada sprint: modo `sprint`.
6. Modo `release` cuando el MVP esté completo.
7. `technical-writer` → README, manuales. Luego skill **client-handover**. **Gate G5**.

## Modo `sprint`
1. `project-manager` + `tech-lead` → `docs/plan/sprint-NN.md` con tareas por agente y orden.
2. Por cada historia, en el orden del plan, correr el modo `feature`.
3. Al cierre: `project-manager` → reporte de estado + retro.

## Modo `feature` — una historia
1. Verificar que la historia esté Ready (si no: `business-analyst`).
2. Si toca datos: `database-engineer` (migración). Si toca contrato: `software-architect` actualiza OpenAPI.
3. Implementación: `backend-developer` → `integrations-engineer`/`ai-engineer` si aplica → `frontend-developer` o `desktop-mobile-developer`.
   Tareas independientes pueden ir en paralelo (ej. backend y frontend contra el contrato ya aprobado).
4. Calidad, en paralelo: `code-reviewer`, `security-engineer`, `qa-engineer`.
   Si hay bloqueantes → volver al desarrollador correspondiente con la lista exacta → repetir el paso 4 (máx. 3 vueltas; luego escalar al humano).
5. `product-owner` acepta la historia contra sus criterios.
6. **Gate G3**: mostrar resumen del PR, veredictos y cómo probarlo; pedir OK para mergear.

## Modo `bug` / `hotfix`
1. `qa-engineer` reproduce y documenta (BUG-NNN) con test que falla.
2. Desarrollador del área corrige (rama `fix/` o `hotfix/`).
3. `code-reviewer` (+ `security-engineer` si toca auth/datos) + `qa-engineer` verifica y corre regresión.
4. Hotfix en producción → modo `release` con versión patch.

## Modo `release`
`release-manager` (skill **release**) + `devops-engineer` (pipeline y deploy). **Gate G4** antes de taggear/desplegar.
Post-deploy: `release-manager` verifica y redacta aviso al cliente.

## Modo `cambio`
Skill **change-request** con `business-analyst`, `software-architect`/`tech-lead` y `sales-engineer`. Nada entra al backlog sin aprobación.

## Modo `incidente`
`sre-support` (skill **incident-postmortem**) → mitigación → fix vía modo `bug` → postmortem.

## Modo `cierre`
`technical-writer` + skill **client-handover** + retro interna. **Gate G5**.

## Reglas del orquestador
- Prompt a cada agente: objetivo, entradas (rutas exactas), entregable esperado (ruta), restricciones y formato de respuesta de `CLAUDE.md`.
- No pases a la fase siguiente si el agente devolvió "Preguntas abiertas" que bloquean: preguntá al humano.
- En un gate, mostrale al humano un resumen breve + rutas a los documentos, y esperá aprobación explícita.
- Si el humano dijo "seguí sin frenar", igual frená en G0 (dinero), G4 (producción) y ante acciones irreversibles.
- Al final de cada modo: resumen de lo hecho, archivos clave y siguiente paso.
