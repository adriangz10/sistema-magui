# Sistema Magui

Punto de venta, control de stock con vencimientos, compras de mercadería, gastos de caja y estadísticas
para una revendedora de cosmética (Natura, Avon, …). Web, móvil (PWA) y escritorio con modo offline.

- Estrategia y plan por fases: [docs/PLAN.md](docs/PLAN.md)
- Convenciones de código: [docs/CONVENCIONES.md](docs/CONVENCIONES.md)
- Referencia de la API: [docs/API.md](docs/API.md)

## Estructura

```
backend/    API Go (arquitectura hexagonal, un paquete por módulo en internal/)
frontend/   Angular 21 + Tailwind 4 (web, PWA y UI del escritorio)
desktop/    App de escritorio Wails con SQLite y sincronización (Fase 10)
deploy/     docker-compose e infraestructura
docs/       Plan, convenciones y decisiones
legacy/     Esquema y script de la primera importación del Excel
```

## Requisitos

Go 1.27+, Node 22+, Angular CLI 21, MySQL 8+ (local o con Docker), Wails 2 (solo para escritorio).

## Desarrollo

### 1. Base de datos

Opción A — MySQL local: crear la base `magui` y un usuario con permisos sobre ella.

Opción B — Docker (puerto 3307):

```bash
cp deploy/.env.example deploy/.env
```

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

### 2. Backend (http://localhost:8080)

```bash
cp backend/.env.example backend/.env
```

Completar `backend/.env`. Luego, desde `backend/`, crear el esquema:

```bash
go run ./cmd/migrate up
```

Importar el Excel de la clienta (solo la primera vez; si se repite, omite los SKU existentes):

```bash
go run ./cmd/migrate -empresa Natura importar-excel "ruta/STOCK VENCIMIENTOS.xlsx"
```

Otros comandos: `status`, `down`, `reset`. Con `-driver sqlite -sqlite magui.db` se usa SQLite.

Configurar la contraseña de la dueña (la pide sin mostrarla):

```bash
go run ./cmd/admin establecer-clave
```

Levantar la API (aplica las migraciones pendientes al arrancar):

```bash
go run ./cmd/api
```

Todas las rutas de `/api` salvo `/api/health` y `/api/auth/login` requieren el encabezado
`Authorization: Bearer <token>` que devuelve el login.

Tests (los de MySQL se activan con `MAGUI_TEST_MYSQL_DSN`, ej. `usuario:clave@tcp(localhost:3306)/`;
crean y borran una base temporal):

```bash
go test ./...
```

### 3. Frontend (http://localhost:4200)

Desde `frontend/` (las llamadas a `/api` se redirigen al backend con `proxy.conf.json`):

```bash
npm start
```

Tests:

```bash
npm test -- --watch=false
```

## Estado

| Fase | Descripción | Estado |
|---|---|---|
| 0 | Base del proyecto | ✅ |
| 1 | Esquema v2 + migraciones + importación | ✅ |
| 2 | Plataforma backend + acceso | ✅ |
| 3 | Catálogo (productos, categorías, empresas, combos, precios) | ✅ |
| 4 | Inventario (lotes, vencimientos, ajustes) y compras | ✅ |
| 5 | Ventas con pagos combinados y anulación | ✅ |
| 6 | Caja (gastos) y estadísticas | ✅ |
| 7 | Frontend base (ingreso, menú, componentes, modo oscuro) | ✅ |
| 8 | Pantallas de cada módulo | ⏳ |
| 9 | PWA móvil | ⏳ |
| 10 | Escritorio offline | ⏳ |
| 11 | Despliegue | ⏳ |
