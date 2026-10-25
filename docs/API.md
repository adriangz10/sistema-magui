# API del Sistema Magui

Base: `/api`. Todo en JSON (`camelCase`). Montos como texto con 2 decimales: `"59200.00"`.

**Sesión:** todas las rutas salvo `/health` y `/auth/login` requieren `Authorization: Bearer <token>`.

**Listas paginadas:** `?page=1&size=10` (máximo 10). Respuesta:
`{ "items": [...], "page": 1, "size": 10, "total": 353, "totalPages": 36 }`

**Errores:** `{ "error": { "code": "sku_duplicado", "message": "…", "fields": { "campo": "…" } } }`

| HTTP | Cuándo |
|---|---|
| 401 | Sin sesión, sesión vencida o contraseña incorrecta |
| 404 | No existe |
| 409 | Duplicado, en uso, o `version_desactualizada` (otro dispositivo lo modificó: recargar) |
| 422 | Datos inválidos (`fields` dice qué campo) |
| 429 | Demasiados intentos de login |

**Control de versión:** al editar se envía la `version` que se leyó. Si alguien modificó el registro
mientras tanto, la API responde 409 `version_desactualizada` en lugar de pisar el cambio.

---

## Acceso

| Método | Ruta | Cuerpo | Respuesta |
|---|---|---|---|
| GET | `/health` | | `{status, version, checks}` |
| POST | `/auth/login` | `{clave}` | `{token, expiraEn}` |
| GET | `/auth/sesion` | | `{activa: true}` |
| POST | `/auth/cambiar-clave` | `{actual, nueva}` | `{token, expiraEn}` (las demás sesiones se cierran) |

## Empresas

| Método | Ruta | Detalle |
|---|---|---|
| GET | `/empresas?q=&activas=true` | Paginado |
| POST | `/empresas` | `{nombre}` |
| PUT | `/empresas/{id}` | `{nombre, activo, version}` |

## Categorías

| Método | Ruta | Detalle |
|---|---|---|
| GET | `/categorias` | Árbol completo: `[{id, nombre, padreId, version, productos, hijos: [...]}]` |
| POST | `/categorias` | `{nombre, padreId?}` |
| PUT | `/categorias/{id}` | `{nombre, padreId?, version}` (mover o renombrar) |
| DELETE | `/categorias/{id}` | Solo si no tiene subcategorías ni productos |
| POST | `/categorias/{id}/precios/vista-previa` | Cambio masivo sin guardar |
| POST | `/categorias/{id}/precios` | Aplica el cambio masivo (todo o nada) |

Cuerpo del cambio masivo:
```json
{ "tipo": "PORCENTAJE", "valor": "10.5", "redondeo": "100", "incluirSubcategorias": true, "motivo": "Aumento Natura" }
```
`tipo`: `PORCENTAJE` (valor en %) o `MONTO` (valor en $, negativo para bajar). Solo afecta productos activos.
Respuesta: `{cantidad, aplicado, productos: [{productoId, sku, nombre, precioAnterior, precioNuevo}]}`.

## Productos

| Método | Ruta | Detalle |
|---|---|---|
| GET | `/productos` | Paginado. Filtros: `q` (nombre/SKU/código, sin importar tildes), `categoriaId` (incluye subcategorías), `empresaId`, `tipo` (SIMPLE/COMBO), `estado` (activos/inactivos/todos), `stockBajo=true`, `orden` (nombre/recientes/precio/stock) |
| GET | `/productos/codigo/{codigo}` | Búsqueda exacta por SKU o código de barras (lector del punto de venta) |
| GET | `/productos/{id}` | Ficha completa con categorías y componentes |
| POST | `/productos` | Crear simple o combo |
| PUT | `/productos/{id}` | Editar (si cambia el precio queda en el historial) |
| PATCH | `/productos/{id}/estado` | `{activo}` (baja lógica) |
| GET | `/productos/{id}/precios` | Historial de precios, paginado |

Cuerpo de producto:
```json
{
  "tipo": "SIMPLE", "sku": "P341355", "codigoBarras": "7790000000011",
  "nombre": "Kaiak Clásico femenino", "descripcion": "", "empresaId": "…",
  "precioVenta": "59200", "costo": "30000", "stockMinimo": 2, "mostrarEnTienda": true,
  "categoriaIds": ["…"], "version": 1
}
```
- `sku` vacío: se genera uno (`MG-XXXXXXXX`, o `CB-XXXXXXXX` para combos).
- SKU y código de barras son únicos entre **todos** los productos (un código nunca puede apuntar a dos productos).
- Combo: `"tipo": "COMBO"` e `"items": [{"productoId": "…", "cantidad": 2}]`. Componentes simples y activos,
  al menos 2 unidades en total. Su `stock` es cuántos se pueden armar con el stock de los componentes.
