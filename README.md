# Sistema Magui

Punto de venta, control de stock con vencimientos, compras de mercadería, gastos de caja y estadísticas
para una revendedora de cosmética (Natura, Avon, …). Web, móvil (PWA) y escritorio con modo offline.

- Estrategia y plan por fases: [docs/PLAN.md](docs/PLAN.md)
- Convenciones de código: [docs/CONVENCIONES.md](docs/CONVENCIONES.md)
- Referencia de la API: [docs/API.md](docs/API.md)
- Despliegue en un VPS: [docs/DESPLIEGUE.md](docs/DESPLIEGUE.md)

## Estructura

```
backend/    API Go (arquitectura hexagonal, un paquete por módulo en internal/)
frontend/   Angular 21 + Tailwind 4 (web, PWA y UI del escritorio)
desktop/    App de escritorio Wails (usa el núcleo backend/escritorio; vende sin internet)
deploy/     Docker: MySQL de desarrollo y sistema completo para el VPS (API + MySQL + Caddy)
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

### 4. Versión web e instalable (como en producción)

Compilar el frontend (desde `frontend/`):

```bash
npm run build
```

Con `FRONTEND_DIR=../frontend/dist/frontend/browser` en `backend/.env`, la API sirve también la app en
http://localhost:8080. El service worker (app instalable y apertura sin señal) solo funciona en esta versión
compilada, no con `npm start`.

**Instalar en el celular:** abrir la dirección del sistema en Chrome (Android) y tocar *Instalar app*, o desde
*Configuración → App en este dispositivo*. En iPhone: Safari → *Compartir* → *Agregar a inicio*.
Requiere HTTPS en producción (Fase 11).

### 5. Escritorio (Windows)

App de escritorio con Wails: la misma pantalla que la web, pero **Vender cobra aunque no haya internet** y las
ventas se suben solas cuando vuelve la conexión (arriba se ve "N ventas pendientes de subir").

Compilar `desktop/build/bin/Magui.exe` (compila también el frontend con `npm run build:escritorio`), desde `desktop/`:

```bash
wails build -clean -skipbindings
```

**Instalador para la clienta** (`desktop/build/bin/Magui-amd64-installer.exe`), con la dirección del servidor de
producción y la versión. Necesita NSIS (`winget install NSIS.NSIS`). Desde `desktop/`, en PowerShell:

```powershell
.\instalador.ps1 -Servidor https://magui.ejemplo.com -Version 1.0.0
```

Se instala para el usuario actual, sin pedir permisos de administrador: el programa queda en
`%LOCALAPPDATA%\Programs\Magui`, con accesos en el menú Inicio y el escritorio. Si Magui está abierto, el
instalador lo cierra. Para actualizar se corre el instalador nuevo encima. Al desinstalar **no** se borran los
datos (ventas pendientes de subir, sesión, configuración). Windows puede mostrar el aviso de SmartScreen
("editor desconocido") porque el instalador no está firmado: *Más información → Ejecutar de todas formas*.

Los datos quedan en `%APPDATA%\Magui`: `config.json` (dirección del servidor, se puede editar), `magui.db`
(copia del catálogo y ventas por subir) y `magui.log`. La primera vez hay que ingresar con conexión.

Para probar el núcleo en el navegador (sin Wails), con la API corriendo y desde `backend/`:

```bash
go run ./cmd/escritorio -servidor http://localhost:8080 -frontend ../desktop/web/browser
```

Abre http://localhost:4300; cortar la API simula quedarse sin internet. `wails build -tags pruebas` arma una
versión que anota cada pedido de la ventana en `magui.log` (y `MAGUI_DATOS` cambia la carpeta de datos).

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
| 8 | Pantallas: vender, ventas, productos, categorías, inventario, compras, caja, inicio, estadísticas, configuración | ✅ |
| 9 | App instalable en el celular (PWA) | ✅ |
| 10 | Escritorio offline (Wails, vende sin internet y sincroniza) | ✅ |
| 11 | Despliegue (Docker + HTTPS, backups, instalador del escritorio) | ✅ |
