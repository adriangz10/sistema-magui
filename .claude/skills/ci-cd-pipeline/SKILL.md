---
name: ci-cd-pipeline
description: Armar o revisar la contenedorización y los pipelines de integración y despliegue continuo (Docker multi-stage, docker-compose, GitHub Actions, entornos staging/producción, migraciones, rollback, backups y monitoreo básico en VPS). Usar al iniciar un proyecto o preparar su primer despliegue.
---

# CI/CD

## Dockerfile (Node/NestJS) — base
```dockerfile
FROM node:22-alpine AS deps
WORKDIR /app
COPY package*.json ./
RUN npm ci

FROM deps AS build
COPY . .
RUN npm run build && npm prune --omit=dev

FROM node:22-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
RUN addgroup -S app && adduser -S app -G app
COPY --from=build --chown=app:app /app/node_modules ./node_modules
COPY --from=build --chown=app:app /app/dist ./dist
COPY --chown=app:app package.json ./
USER app
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s CMD wget -qO- http://localhost:3000/health || exit 1
CMD ["node", "dist/main.js"]
```
Verificá la versión LTS de Node vigente antes de fijarla.

## CI — `.github/workflows/ci.yml`
```yaml
name: CI
on: { pull_request: {}, push: { branches: [main] } }
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env: { POSTGRES_PASSWORD: postgres, POSTGRES_DB: app_test }
        ports: ["5432:5432"]
        options: >-
          --health-cmd pg_isready --health-interval 5s --health-timeout 5s --health-retries 10
    env:
      DATABASE_URL: postgres://postgres:postgres@localhost:5432/app_test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 22, cache: npm }
      - run: npm ci
      - run: npm run lint
      - run: npm run migration:run --if-present
      - run: npm test -- --ci
      - run: npm run build
      - run: npm audit --omit=dev --audit-level=high
```

## Deploy — `.github/workflows/deploy.yml` (resumen)
1. `on: push tags v*` (y `workflow_dispatch`).
2. Build de imagen y push a GHCR con tags `vX.Y.Z` y `sha`.
3. Job `staging`: SSH al servidor → `docker compose pull` → correr migraciones → `docker compose up -d` → smoke test `/health`.
4. Job `production`: `environment: production` (aprobación manual requerida), mismos pasos.
Secretos: `SSH_HOST`, `SSH_USER`, `SSH_KEY`, variables de app en el servidor (`.env` con permisos 600) o en GitHub Environments.

## Runbooks obligatorios
- `docs/ops/runbooks/deploy.md` — pasos manuales equivalentes al pipeline.
- `docs/ops/runbooks/rollback.md` — volver al tag anterior; qué hacer con migraciones.
- `docs/ops/runbooks/backup-restore.md` — `pg_dump -Fc` diario + copia externa + **restauración probada**.

## Checklist de primer deploy
- [ ] Dominio y DNS, TLS automático (Caddy/Nginx + certbot)
- [ ] Firewall, usuario no-root, SSH por clave
- [ ] Health check y uptime monitor externo
- [ ] Logs con rotación
- [ ] Backups programados + restauración probada
- [ ] Variables de entorno de prod cargadas, nada por defecto
