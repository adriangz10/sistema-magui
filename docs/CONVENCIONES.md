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
- Un módulo **no** importa el `domain` de otro. Se comunica mediante puertos de `application` (ej. ventas usa un puerto `Inventario`).
- El cableado (qué adaptador va con qué caso de uso) vive solo en `cmd/*`.
- `internal/platform` es infraestructura compartida (config, logger, http, db), no lógica de negocio.

### Reglas
- Dinero: `decimal` (nunca `float64`). En MySQL `DECIMAL(12,2)`.
- IDs: UUID v7 generados por la aplicación (permite crear registros offline).
- Fechas: se guardan en UTC y se muestran en la hora de Argentina.
- Errores de dominio tipados (`ErrStockInsuficiente`, …). El adaptador HTTP los traduce a códigos de estado.
- Toda operación que toca varias tablas va en una transacción.
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
- Tablas siempre paginadas, máximo 10 filas.

## Git
- Rama `main` estable; una rama por fase o funcionalidad (`fase-1-esquema`, …).
- Commits en español, en imperativo: `Agrega caso de uso de registrar venta`.
