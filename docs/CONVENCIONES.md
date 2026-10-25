# Convenciones

## Idioma
- **Negocio en español** (nombres de entidades, tablas, rutas, mensajes): `producto`, `venta`, `/api/ventas`.
- Términos técnicos en inglés cuando son estándar en Go/Angular: `Repository`, `Handler`, `Service`.

## Backend (Go)

### Capas por módulo
```
internal/<modulo>/
  domain/        entidades, value objects, errores de negocio. Sin imports externos (solo stdlib).
  application/   casos de uso + puertos (interfaces). Solo importa domain.
  adapters/
    http/        handlers HTTP → llaman casos de uso
    mysql/       implementación de los puertos para la nube
    sqlite/      implementación de los puertos para el escritorio
```
- Las dependencias apuntan **hacia adentro**: adapters → application → domain.
- Un módulo **no** importa el `domain` de otro. Se comunica mediante puertos de `application` (ej. compras define
  un puerto `Inventario`). La implementación vive en `adapters/enlaces` (o similar) del módulo que consume y
  llama al servicio del otro módulo en el mismo proceso; la transacción se comparte por el contexto.
- El cableado (qué adaptador va con qué caso de uso) vive solo en `cmd/*`.
- `internal/platform` es infraestructura compartida (config, logger, http, db), no lógica de negocio.
- `internal/kernel` es el núcleo compartido que pueden usar todos los dominios: `dinero`, `errores`,
  `paginacion`, `id`, `reloj` y `tx` (puerto de transacciones). No depende de nada más.
- Cada módulo expone un `modulo.go` en su raíz que arma servicio + adaptadores; `cmd/*` solo los junta.

### Reglas
- Dinero: nunca `float64`. En el dominio es un tipo `Dinero` (int64 en **centavos**).
  MySQL lo guarda como `DECIMAL(12,2)` y SQLite como `INTEGER` en centavos; cada adaptador convierte.
- Esquema: las migraciones viven en `backend/migrations/{mysql,sqlite}` y se mantienen **en paralelo**
  (mismas tablas y columnas). Cada cambio de esquema es una migración nueva en ambas carpetas.
- IDs: UUID v7 generados por la aplicación (permite crear registros offline).
- Fechas: se guardan en UTC y se muestran en la hora de Argentina.
- Errores de dominio tipados (`ErrStockInsuficiente`, …). El adaptador HTTP los traduce a códigos de estado.
- Toda operación que toca varias tablas va en una transacción: el caso de uso recibe un
  `tx.Transactor` y los repositorios usan `db.Conn(ctx, conn)`, que toma la transacción del contexto.
- Errores: los casos de uso devuelven `errores.Invalido/NoEncontrado/Conflicto/...` con un código estable
  (`stock_insuficiente`). `httpx.Error` los traduce a 422/404/409/401/429; cualquier otro error es 500
  y se registra en el log sin mostrar detalles al usuario.
- Horas: `reloj.Reloj` se inyecta en los servicios (en tests, `reloj.Fijo`). Para agrupar "el día" se
  usa `reloj.ZonaArgentina`.
- Tests: casos de uso con repositorios en memoria; adaptadores con base real.

### API HTTP
- Prefijo `/api`, recursos en plural: `/api/productos`, `/api/ventas/{id}`.
- Paginación: `?page=1&size=10` (máximo 10). Respuesta:
  ```json
  { "items": [], "page": 1, "size": 10, "total": 0, "totalPages": 0 }
  ```
- Errores:
  ```json
  { "error": { "code": "stock_insuficiente", "message": "…", "fields": { "cantidad": "…" } } }
  ```
- JSON en `camelCase`; montos como string decimal (`"59200.00"`) para no perder precisión.

## Frontend (Angular)
- Componentes standalone, `OnPush`, signals, `input()`/`output()`, control flow nativo (`@if`, `@for`).
- Estructura: `core/` (servicios globales, API), `shared/` (UI reutilizable), `features/<modulo>/` con carga diferida.
- Archivos: `*.page.ts` (pantallas de ruta), `*.component.ts` (piezas), `*.service.ts`.
- Estilos solo con Tailwind y los tokens de `styles.css` (`brand-*`, `ink-*`, `ok`, `warn`, `danger`).
- Accesibilidad WCAG AA: contraste, foco visible, etiquetas en formularios.
- Tablas siempre paginadas, máximo 10 filas: usar `<app-tabla>` (incluye paginador, carga y estado vacío).

### Piezas compartidas del frontend
| Pieza | Para qué |
|---|---|
| `core/auth` | `AuthService` (sesión), `authInterceptor` (token y cierre por 401), guards de ruta |
| `core/api` | `ApiError` (errores de la API ya interpretados), `Pagina<T>`, `parametros()` |
| `core/ui` | `ToastService` (avisos), `ConfirmacionService` (`await preguntar(...)`), `TemaService` (claro/oscuro) |
| `core/red` | `ConexionService` (en línea / sin conexión) |
| `shared/formato` | Pipes `dinero` y `fecha`; `interpretarDinero()` acepta "59.200,50" y devuelve "59200.50" |
| `shared/ui` | `app-tabla`, `app-paginador`, `app-modal`, `app-campo`, `app-input-dinero` (usa `idCampo`), `app-encabezado`, `app-buscador-producto`, `app-motivo-modal` |
| `shared/graficos` | `app-grafico-columnas` (SVG, tooltip, tabla), `app-barras-horizontales`, `app-tarjeta-dato`; adaptadores de datos de estadísticas |
| `shared/catalogo` | `aplanar()` y utilidades del árbol de categorías |
| `styles.css` | Clases base: `card`, `btn-primario`, `btn-secundario`, `btn-fantasma`, `btn-peligro`, `btn-icono`, `input`, `etiqueta`, `badge-*`, `tabla`, `titulo-pagina` |

- Montos: la API usa texto `"59200.50"`; para sumar en pantalla convertir a centavos (`aCentavos`) y nunca usar `number` con decimales.
- Fechas y horas siempre en hora de Argentina y formato de 24 h (`fecha` pipe).
- Gráficos: una sola serie con el rosa de marca validado para contraste (`brand-600` en claro, `brand-500` en oscuro), sin leyenda, eje Y redondeado, máximo rotulado, tooltip con mouse y teclado y siempre "Ver como tabla".
- Íconos: `lucide-angular` (`<lucide-icon [img]="Icono" />`), empaquetados en la app (funciona sin internet).

## Git
- Rama `main` estable; una rama por fase o funcionalidad (`fase-1-esquema`, …).
- Commits en español, en imperativo: `Agrega caso de uso de registrar venta`.
