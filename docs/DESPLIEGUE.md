# Despliegue en un VPS

El sistema completo corre con Docker en un servidor propio: **MySQL + API (que también sirve la app web) +
Caddy**, que pide y renueva solo el certificado HTTPS de Let's Encrypt. Archivos en `deploy/`:

| Archivo | Para qué |
|---|---|
| `Dockerfile` | Imagen de la API: compila la app web y los binarios Go; imagen mínima sin shell, usuario sin privilegios |
| `docker-compose.prod.yml` | Los servicios (MySQL, API, Caddy y backup), redes y volúmenes |
| `Caddyfile` | HTTPS automático para `DOMINIO` |
| `backup.sh` | Copia diaria de la base (lo corre el servicio `backup`) |
| `restaurar.sh` | Vuelve la base a una copia |
| `.env.prod.example` | Variables a completar (se copia a `.env.prod`, que nunca se sube al repositorio) |

## Qué hace falta

- Un VPS Linux con 1 GB de RAM o más (ej. Ubuntu 24.04), con los puertos 80 y 443 abiertos.
- Un dominio o subdominio (ej. `magui.ejemplo.com`) con un **registro A** que apunte a la IP del VPS.
- Docker con el plugin compose (`curl -fsSL https://get.docker.com | sh`).

## Primera instalación

1. Copiar el repositorio al VPS (por ejemplo con `git clone`) y entrar a la carpeta.

2. Crear la configuración y completarla (dominio, correo y contraseñas largas):

   ```bash
   cp deploy/.env.prod.example deploy/.env.prod
   ```

   Para generar cada contraseña y el `AUTH_SECRET`:

   ```bash
   openssl rand -base64 48
   ```

   Proteger el archivo:

   ```bash
   chmod 600 deploy/.env.prod
   ```

3. Levantar todo (la primera vez compila la imagen; tarda unos minutos):

   ```bash
   docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod up -d --build
   ```

   La API crea el esquema de la base al arrancar. Caddy obtiene el certificado apenas el dominio responde.

4. Cargar el Excel de la clienta (solo la primera vez). Subirlo al VPS, por ejemplo a `~/datos/`, y correr:

   ```bash
   docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod run --rm --no-deps -v ~/datos:/datos:ro --entrypoint /app/migrate api -empresa Natura importar-excel "/datos/STOCK VENCIMIENTOS.xlsx"
   ```

5. Poner la contraseña de la dueña (la pide sin mostrarla):

   ```bash
   docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod exec api /app/admin establecer-clave
   ```

6. Abrir `https://DOMINIO`, ingresar y comprobar. `https://DOMINIO/api/health` debe responder `"status":"ok"`.

## Actualizar a una versión nueva

Bajar los cambios (`git pull`), cambiar `VERSION` en `deploy/.env.prod` y volver a levantar:

```bash
docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod up -d --build
```

Las migraciones nuevas se aplican solas al arrancar la API. Los datos quedan en los volúmenes de Docker
(`mysql_data`, y `caddy_data` para los certificados): **nunca usar `down -v`**, borra la base.

## Copias de seguridad

El servicio `backup` hace una copia de la base **todos los días a las 03:00** (hora de Argentina, se cambia con
`BACKUP_HORA`) y guarda las de los últimos **14 días** (`BACKUP_DIAS`). Al arrancar, si no hay ninguna copia de
las últimas 24 horas, hace una enseguida. Quedan en `deploy/backups/magui-AAAAMMDD-HHMMSS.sql.gz`.

- La copia se hace sin frenar el sistema (las ventas siguen) y se verifica antes de darla por buena: si se
  corta a la mitad, se descarta.
- `docker compose ... ps` muestra el servicio `backup` como **unhealthy** si no hay una copia de las últimas
  26 horas.

Hacer una copia ya (por ejemplo, antes de actualizar):

```bash
docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod exec backup bash /backup.sh ahora
```

**Guardar copias fuera del VPS.** Si el VPS se pierde, se pierden también sus copias. Bajar una a la PC cada
tanto (desde la PC):

```bash
scp usuario@IP_DEL_VPS:magui/deploy/backups/magui-*.sql.gz .
```

o automatizarlo con un servicio de almacenamiento (por ejemplo `rclone` a Google Drive con un `cron` diario).

### Restaurar una copia

**Reemplaza todos los datos actuales.** El script primero guarda una copia del estado actual (por si hay que
volver atrás), detiene la API, restaura y la vuelve a levantar. Pide escribir `RESTAURAR` para confirmar:

```bash
deploy/restaurar.sh deploy/backups/magui-20261026-030000.sql.gz
```

Sin argumentos muestra las copias disponibles. Todo lo que se cargó **después** de esa copia se pierde (ventas,
compras, cambios de precio), incluidas las ventas del escritorio que ya se habían subido. Las del escritorio
que todavía estaban pendientes se suben solas después.

## Comandos útiles

Estado y salud de los servicios:

```bash
docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod ps
```

Registro de la API (cada pedido con su IP, estado y duración):

```bash
docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod logs -f api
```

## Seguridad

- MySQL no publica puertos y está en una red interna sin salida a internet: solo la API llega a la base.
- La API no publica puertos: se entra solo por Caddy (HTTPS, con HSTS). HTTP redirige a HTTPS.
- La imagen de la API no tiene shell y corre con un usuario sin privilegios.
- Límite de intentos de ingreso por IP: detrás de Caddy, la IP real sale de `X-Forwarded-For`, pero solo se
  acepta si el pedido viene de Caddy (`TRUSTED_PROXIES`, IP fija `172.30.10.2`). Un cliente no puede esquivar el
  límite inventando ese encabezado.
- `AUTH_SECRET` es obligatorio en producción (la API no arranca sin él).

## App de escritorio

Con el servidor funcionando, generar el instalador apuntando al dominio (en la PC de desarrollo, desde `desktop/`):

```powershell
.\instalador.ps1 -Servidor https://magui.ejemplo.com -Version 1.0.0
```

Llevar `desktop/build/bin/Magui-amd64-installer.exe` a la PC de la tienda, instalarlo y abrir Magui. La primera
vez hay que ingresar con conexión. Si cambia la dirección del servidor, se edita
`%APPDATA%\Magui\config.json` y se vuelve a abrir la app (o se genera un instalador nuevo).

## Requisitos de MySQL para la concurrencia de stock (ADR 0002)
- **`binlog_format=ROW`** (es el valor por defecto en MySQL 8.4). Las transacciones de la API corren en `READ COMMITTED` y InnoDB
  no admite escrituras en ese nivel con `binlog_format=STATEMENT`. Verificar con `SHOW VARIABLES LIKE 'binlog_format';`.
- **`innodb_lock_wait_timeout`**: la API lo fija en **10 s por sesión** desde el DSN (`platform/config`), no hace falta tocar el
  `.env` ni `my.cnf`. No subirlo por encima de ~20 s: el escritorio corta a los 30 s y reenviaría la venta.
- Un deadlock se reintenta solo (hasta 3 veces); si persiste, la API responde 409 `operacion_concurrente` y la pantalla permite reintentar.
