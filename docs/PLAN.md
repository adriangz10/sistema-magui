# Sistema Magui — Estrategia y plan

Punto de venta + control de stock + estadísticas para una revendedora de cosmética (Natura, Avon, etc.).
Funciona en **web**, **móvil** y **escritorio**; el escritorio sigue cobrando **sin internet** y se sincroniza al volver la conexión.

---

## 1. Requisitos (lo que pidió la clienta)

| Módulo | Qué hace |
|---|---|
| **Catálogo** | Productos con SKU **y** código de barras propio, categorías jerárquicas, empresa (Natura, Avon… opcional y ampliable), precio de venta manual, costo. |
| **Precios en cascada** | Elegir una categoría y subir/bajar precios (% o monto) de todos sus productos y subcategorías. Queda historial. |
| **Combos** | Producto compuesto (ej. "Día de la Madre") con varios productos y cantidades; al venderlo descuenta stock de cada componente. Precio propio. |
| **Stock y vencimientos** | Stock por lote con fecha de vencimiento; alertas "por vencer" y "vencido". Cada cambio de stock queda como movimiento. |
| **Compras de mercadería** | Registrar llegada de mercadería: empresa, fecha, productos, cantidad, costo, vencimiento, y el nuevo precio de reventa (manual). Total gastado. |
| **Ventas (POS)** | Venta rápida por escáner/búsqueda, solo con fecha. Pago combinado: efectivo, transferencia, QR, tarjeta. El front muestra **cuánto falta** y el **vuelto** en efectivo. Cada venta guarda sus pagos. |
| **Gastos / retiros de caja** | Sacar plata de caja para gastos que no son mercadería (motivo, monto, fecha, método). |
| **Estadísticas** | Ventas del día/semana/mes, más vendidos, gasto en compras, gastos de caja, margen de ganancia %, ganancia neta, gráficos. |
| **UI** | Angular + Tailwind, estilizado, paginación de **máx. 10 filas** por tabla. |

**Fuera de alcance (a propósito):** usuarios de empleados, apertura/cierre de caja, tickets/facturación, clientes, proveedores formales.

> ⚠️ Aunque no haya empleados, la versión **web** está en internet: necesita **un único acceso de la dueña** (contraseña/PIN) para que nadie más vea o modifique datos. No es "gestión de usuarios", es solo un candado.

**Mejoras que agrego:** vuelto en efectivo, anular venta (devuelve stock), alertas de stock bajo y vencimientos en el inicio, historial de precios, importación del Excel actual, exportar reportes a Excel/CSV, modo oscuro.

---

## 2. Arquitectura general

```
                ┌──────────────────────────── NUBE (VPS) ────────────────────────────┐
  Web (navegador) ─┐                                                                │
                   ├──HTTPS──►  API Go (cmd/api)  ──►  MySQL (fuente de verdad)      │
  Móvil (PWA) ─────┘                ▲                                               │
                └───────────────────┼───────────────────────────────────────────────┘
                                    │ sincronización (push ventas / pull catálogo)
                ┌───────────────────┴──────────── PC DE LA TIENDA ───────────────────┐
                │  App escritorio (Wails) = mismo Angular + mismo núcleo Go          │
                │                           + SQLite local + servicio de sync        │
                └────────────────────────────────────────────────────────────────────┘
```

- **Un solo frontend Angular** para las tres versiones. Solo cambia la URL de la API.
- **Un solo núcleo Go** (dominio + casos de uso). Cambian los adaptadores: MySQL en la nube, SQLite en el escritorio.
- **Web:** Angular servido por la API Go.
- **Móvil:** el mismo Angular como **PWA** instalable (icono en el celular, pantalla completa). Más adelante, si hace falta tienda de apps, se envuelve con Capacitor sin reescribir.
- **Escritorio:** **Wails v2** (ya instalado). Genera un `.exe` liviano con Angular adentro y el backend Go corriendo local contra SQLite.

### Arquitectura hexagonal (puertos y adaptadores)

```
           Adaptadores de entrada            Núcleo (sin dependencias externas)         Adaptadores de salida
   ┌───────────────────────────┐      ┌─────────────────────────────────────┐      ┌─────────────────────────┐
   │ HTTP REST (chi)           │ ───► │  application: casos de uso          │ ───► │ MySQL repos             │
   │ Wails (escritorio)        │      │     usan PUERTOS (interfaces)       │      │ SQLite repos            │
   │ Worker de sincronización  │      │  domain: entidades, reglas, errores │      │ Cliente HTTP de sync    │
   └───────────────────────────┘      └─────────────────────────────────────┘      │ Reloj, generador de IDs │
                                                                                   └─────────────────────────┘
```

Regla: `domain` no importa nada; `application` solo importa `domain` y define interfaces; los `adapters` implementan esas interfaces. El cableado se hace en `cmd/*`.

### Estructura de carpetas (monorepo)

```
sistema-magui/
├── backend/
│   ├── cmd/
│   │   ├── api/                 # servidor nube (MySQL)
│   │   └── migrate/             # migraciones + importación del Excel
│   ├── internal/
│   │   ├── catalogo/            # productos, categorías, empresas, combos, precios
│   │   │   ├── domain/
│   │   │   ├── application/     # casos de uso + puertos (interfaces)
│   │   │   └── adapters/        # http/, mysql/, sqlite/
│   │   ├── inventario/          # lotes, movimientos, vencimientos
│   │   ├── compras/
│   │   ├── ventas/              # ventas + pagos
│   │   ├── caja/                # gastos / retiros
│   │   ├── estadisticas/        # solo lectura
│   │   ├── sync/                # outbox, push/pull
│   │   ├── auth/                # acceso único de la dueña
│   │   └── platform/            # config, logger, db, http server, errores, paginación
│   ├── migrations/{mysql,sqlite}/
│   └── go.mod
├── frontend/                    # Angular 21 + Tailwind (standalone, signals)
│   └── src/app/
│       ├── core/                # api client, interceptores, config de entorno
│       ├── shared/              # tabla paginada, modales, inputs de dinero, toasts
│       └── features/            # pos, productos, categorias, combos, compras, gastos, dashboard
├── desktop/                     # proyecto Wails: usa backend/internal + frontend/dist
├── deploy/                      # docker-compose (MySQL + API), Caddy/HTTPS
└── docs/
```

Stack Go: `net/http` + `chi` (router), `database/sql` + `sqlx`, `golang-migrate`, `modernc.org/sqlite` (sin CGO, ideal para Wails), `shopspring/decimal` para dinero, IDs **UUID v7**.

---

## 3. Estrategia offline y sincronización (escritorio)

El principio: **en la tienda nunca se deja de cobrar**.

1. **IDs generados en el cliente (UUID v7).** Una venta hecha offline ya tiene su ID definitivo, así no hay choques al subirla.
2. **El stock no se guarda como número, sino como movimientos** (`+10 compra`, `-1 venta`, `+1 anulación`). El stock es la suma. Dos dispositivos que venden offline nunca se pisan: el servidor solo suma los movimientos.
3. **Outbox local.** Cada operación del escritorio (venta, gasto, compra, anulación) se guarda en SQLite y en una tabla `sync_outbox`, en la misma transacción.
4. **Push.** Cuando hay internet, un worker envía el outbox a `POST /sync/push`. El servidor aplica cada operación una sola vez (idempotente por ID). Si se corta a la mitad, reintenta sin duplicar.
5. **Pull.** El escritorio pide `GET /sync/pull?desde=<cursor>`: productos, precios, categorías, combos y movimientos nuevos hechos desde la web o el celular.
6. **Conflictos:**
   - Ventas, gastos, compras y movimientos solo se agregan, nunca se editan, así que no hay conflicto.
   - Catálogo (precio, nombre): gana el último cambio (`updated_at` + `version`).
   - Stock negativo por ventas offline simultáneas: se permite y se marca en alertas para revisar.
7. **Indicador en pantalla:** "🟢 En línea / 🟠 Sin conexión — 3 ventas pendientes de subir".

Web y móvil trabajan online contra la API. Opcional en una fase posterior: la PWA puede cachear el catálogo para consultar precios sin señal.

---

## 4. Modelo de datos (v2)

Reemplaza el esquema actual, que era solo una copia limpia del Excel. Los datos ya cargados se migran.

```
empresa(id, nombre UNIQUE, activo)
categoria(id, nombre, padre_id → categoria)
producto(id UUID, sku UNIQUE, codigo_barras UNIQUE NULL, nombre, descripcion,
         empresa_id NULL, tipo ENUM('SIMPLE','COMBO'), precio_venta, costo_ultimo,
         stock_minimo, activo, peso/medidas, mostrar_en_tienda, created_at, updated_at, version)
producto_categoria(producto_id, categoria_id)
combo_item(combo_id → producto, producto_id → producto, cantidad)
precio_historial(id, producto_id, precio_anterior, precio_nuevo, motivo, fecha)

lote(id UUID, producto_id, fecha_vencimiento NULL, costo_unitario, compra_item_id NULL)
movimiento_stock(id UUID, producto_id, lote_id NULL, cantidad ±, tipo ENUM('COMPRA','VENTA',
                 'ANULACION','AJUSTE','INICIAL'), referencia_id, fecha)

compra(id UUID, fecha, empresa_id NULL, total, notas)
compra_item(id, compra_id, producto_id, cantidad, costo_unitario, precio_venta_nuevo, fecha_vencimiento)

venta(id UUID, fecha, total, costo_total, estado ENUM('CONFIRMADA','ANULADA'), origen ENUM('WEB','MOVIL','DESKTOP'))
venta_item(id, venta_id, producto_id, cantidad, precio_unitario, costo_unitario, subtotal)
venta_pago(id, venta_id, metodo ENUM('EFECTIVO','TRANSFERENCIA','QR','TARJETA'), monto, monto_recibido NULL, vuelto NULL)

gasto_caja(id UUID, fecha, monto, motivo, categoria_gasto, metodo)

sync_outbox / sync_cursor   (solo escritorio)
```

Claves:
- `venta_item.costo_unitario` guarda el costo **al momento de vender**. Así el margen histórico no cambia si después cambia el costo.
- Al vender un combo, `venta_item` registra el combo y se generan movimientos de stock negativos de cada componente. Los lotes se consumen primero por fecha de vencimiento más próxima (FEFO).
- Precio en cascada: una consulta recursiva toma la categoría y todas sus hijas, y el cambio se aplica en una transacción con historial.
- Margen % = `(ventas − costo de lo vendido) / ventas`. Ganancia neta = `ventas − costo de lo vendido − gastos de caja`.

---

## 5. Plan por fases (paso a paso, con tu aprobación en cada una)

| # | Fase | Entregable | Cómo se verifica |
|---|---|---|---|
| **0** | Base del proyecto | Monorepo, `git init`, `go.mod`, Angular + Tailwind creados, `docker-compose` con MySQL, README, convenciones | `go build`, `ng serve` y la DB levantan |
| **1** | Esquema v2 + migraciones | Migraciones MySQL (y SQLite equivalente) + importación del Excel/DB actual | 353 productos y stock 1057 en el nuevo esquema |
| **2** | Plataforma backend | Config, logger, conexión DB, router, manejo de errores, paginación estándar (`page`, `size≤10`), acceso de la dueña | Endpoint `/health` y login con tests |
| **3** | Catálogo | CRUD de productos, categorías, empresas; búsqueda por SKU/código de barras; combos; precios en cascada | Tests de casos de uso + pruebas HTTP |
| **4** | Inventario + compras | Lotes, movimientos, vencimientos, alertas; registrar compra con costo y nuevo precio | Stock correcto tras compra |
| **5** | Ventas + pagos | Crear venta con pagos combinados, validación de total, vuelto, anulación, FEFO, combos | Tests de la suma de pagos y del stock |
| **6** | Caja + estadísticas | Gastos/retiros; endpoints de estadísticas (día/semana/mes, top, margen, compras) | Números cuadran con datos de prueba |
| **7** | Frontend base | Layout, tema, navegación, componentes compartidos (tabla paginada, modal, montos, toasts), login | Navegable en desktop y celular |
| **8** | Frontend módulos | POS (escáner, carrito, pagos combinados con "falta $X"), productos, categorías/cascada, combos, compras, gastos, dashboard con gráficos | Recorrido completo de una venta |
| **9** | PWA móvil | Manifest, instalable, diseño responsive para el celular | Instalada en un celular |
| **10** | Escritorio offline | Proyecto Wails, adaptadores SQLite, outbox, worker de sync, indicador online/offline | Vender sin internet, reconectar, ver la venta en la web |
| **11** | Despliegue | Docker para VPS (API + MySQL + HTTPS), instalador `.exe`, backups automáticos de MySQL | Sistema accesible por dominio |

Cada fase termina con: código compilando, tests pasando, un commit y un resumen antes de seguir con la siguiente.

---

## 6. Decisiones a confirmar

1. **Escritorio con Wails** (Go nativo, ya instalado) — recomendado frente a Electron/Tauri por compartir el mismo código Go.
2. **Hosting de la versión web:** VPS con Docker (ej. ~USD 5/mes) o un servidor que ya tengas.
3. **Acceso de la dueña:** una contraseña única para web/móvil (el escritorio puede abrir directo).
4. **Precio de oferta del Excel (precio × 0.6):** ¿se sigue usando o se descarta? Propuesta: descartarlo y usar solo precio de venta (más simple para el POS).
5. **Moneda:** pesos argentinos, 2 decimales.
