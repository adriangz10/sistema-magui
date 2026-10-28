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

## Inventario

El stock nunca se edita directo: cada cambio es un **movimiento** (`INICIAL`, `COMPRA`, `VENTA`, `ANULACION`, `AJUSTE`,
`EDICION_COMPRA`) y el stock es la suma. `EDICION_COMPRA` es la diferencia de cantidad al editar o quitar un renglón de una
compra (positivo o negativo, en el lote de ese renglón, `referenciaId` = compra) — disponible desde US-001b. Cada compra crea un **lote** con su vencimiento y costo. Las salidas se descuentan primero de los lotes
vigentes que vencen antes (FEFO), después los sin fecha y por último los ya vencidos (lo vencido sigue en las
alertas hasta que se descarte con un ajuste).

| Método | Ruta | Detalle |
|---|---|---|
| GET | `/inventario/productos/{id}/stock` | `{productoId, sku, nombre, stock, lotes: [{id, vencimiento, costo, stock, estado, diasRestantes}]}` |
| GET | `/inventario/movimientos` | Paginado. Filtros: `productoId`, `tipo`, `desde`, `hasta` (AAAA-MM-DD, inclusive) |
| POST | `/inventario/ajustes` | `{productoId, cantidad, motivo, loteId?, vencimiento?, costo?}` — cantidad positiva suma, negativa resta (nunca deja stock negativo) |
| GET | `/inventario/vencimientos?estado=` | `vencidos`, `por_vencer` o `todos`. Lotes con stock, ordenados por fecha |
| GET | `/inventario/alertas` | `{vencidos, porVencer, sinStock, stockBajo, diasAlerta}` |
| GET / PUT | `/inventario/configuracion` | `{diasAlertaVencimiento}` (1 a 365, por defecto 90) |

Estados de un lote: `VENCIDO`, `POR_VENCER` (dentro del plazo de aviso), `VIGENTE`, `SIN_FECHA`.

## Compras

| Método | Ruta | Detalle |
|---|---|---|
| GET | `/compras` | Paginado, de la más reciente. Filtros: `desde`, `hasta`, `empresaId`. Cada compra trae `version` y `editadaEn` |
| GET | `/compras/{id}` | Detalle con renglones, `version`, `editadaEn` e historial de `ediciones` (ver "Editar una compra") |
| POST | `/compras` | Registra la compra (ver abajo) |
| PUT | `/compras/{id}` | Edita la compra (ver "Editar una compra") |

```json
{
  "fecha": "2026-10-24", "empresaId": "…", "notas": "Pedido ciclo 15",
  "items": [
    { "productoId": "…", "cantidad": 4, "costoUnitario": "35520", "precioVentaNuevo": "62000", "vencimiento": "2027-09-30" }
  ]
}
```
Al registrarla, en una sola operación (todo o nada):
- se guarda la compra con su total (suma de cantidad × costo),
- cada renglón suma stock en un lote nuevo con su vencimiento y costo,
- el producto queda con ese costo como "último costo" y, si se indicó `precioVentaNuevo`, con el nuevo precio
  de reventa (queda en el historial con origen `COMPRA`) — **solo si esta es la compra más reciente de ese producto**
  (por `fecha`; a igual fecha, la cargada después). Registrar una compra con fecha anterior a otra ya cargada no pisa el
  último costo ni el precio. Si el producto está en varios renglones, rige el último.

Los combos no se compran (se compran sus productos). `fecha` vacía = ahora.

### Editar una compra

Se entrega por partes (a, b y c ya están disponibles): **US-001a** cabecera (fecha, empresa, notas), **US-001b** renglones existentes, **US-001c**
agregar y quitar renglones. El contrato es el mismo en las tres. Diseño: `docs/architecture/editar-compra.md` y ADRs 0001–0005.

```json
// PUT /compras/{id}
{
  "fecha": "2026-10-19", "empresaId": "…", "notas": "Pedido ciclo 15",
  "version": 3, "origen": "WEB", "motivo": "Error de carga",
  "items": [
    { "id": "…", "productoId": "…", "cantidad": 6, "costoUnitario": "36000", "precioVentaNuevo": "65000", "vencimiento": "2027-12-31" },
    { "productoId": "…", "cantidad": 2, "costoUnitario": "15000", "precioVentaNuevo": null, "vencimiento": null }
  ]
}
```
- `version` (obligatoria): la que se leyó. Si la compra cambió mientras tanto → 409 `version_desactualizada`
  ("Otra persona modificó esta compra, recargá") y no se aplica nada.
- `fecha` (obligatoria): `AAAA-MM-DD` o fecha y hora ISO. Si es el mismo día que ya tenía, se conserva la hora original
  (no cuenta como cambio); si es hoy, queda la hora actual. No puede ser futura (tolerancia 1 hora).
- `empresaId` vacío = sin empresa (se valida que exista, no que esté activa). `notas` hasta 500 caracteres.
- `origen`: `WEB` (por defecto), `MOVIL` o `DESKTOP`. `motivo`: opcional, hasta 255 caracteres.
- `items` ausente o `null` = los renglones no cambian. Si viene (desde US-001b), es la lista **completa** de renglones
  (reemplazo, no parche); los índices `items[i]` de los errores corresponden a la posición en la lista enviada:
  - con `id`: renglón existente; el `productoId` debe ser el mismo (para cambiar de producto se quita y se agrega otro).
  - sin `id`: renglón nuevo (US-001c): crea un lote nuevo con un movimiento `COMPRA`; el producto debe existir (404 `producto_no_encontrado`) y no ser combo (422 `producto_no_comprable`). Un renglón quitado antes no se puede reenviar (su `id` da 422 `items[i].id`).
  - renglones existentes que no vienen = quitados (US-001c): solo si su lote no tiene unidades consumidas (si no, 409 `compra_consumida` con `fields["quitados[<id>]"]`); se descuenta solo la cantidad que trajo el renglón (lo sumado por ajustes manuales queda en el lote). La compra conserva al menos 1 renglón (422 `items`).
  - Mismas validaciones que al registrar: 1 a 300 renglones, cantidad entre 1 y 1.000.000, costo y precio hasta 9.999.999.999,99, total de la compra hasta 9.999.999.999,99 (422 `items` si lo supera), cantidad > 0, costo ≥ 0, precio > 0 si se informa,
    producto existente y no combo.
  - **`vencimiento: null` (o ausente) deja el lote SIN vencimiento** (estado `SIN_FECHA`); no significa "sin cambio".
    Lo mismo vale para `precioVentaNuevo: null`: el renglón queda sin precio de reventa (el precio del producto no se revierte).
    Para no cambiar un campo hay que reenviar su valor actual.

Al editar, en una sola operación (todo o nada):
- se recalcula el `total` (suma de cantidad × costo de los renglones que quedan);
- si cambió la cantidad de un renglón, se registra un movimiento `EDICION_COMPRA` por la diferencia **en el mismo lote**
  ("Edición de compra del dd/mm/aaaa"); nunca se modifican movimientos existentes;
- si cambió el costo o el vencimiento, el lote se actualiza en el lugar (afecta solo al stock restante y a ventas futuras;
  las ventas ya hechas no se recalculan);
- un renglón nuevo crea su lote con un movimiento `COMPRA`; uno quitado descuenta de su lote la cantidad que trajo (`EDICION_COMPRA` negativo; el lote queda con lo sumado a mano, si hubo) y se
  conserva como historia (no aparece más en el detalle);
- **último costo y precio:** solo si esta compra es la más reciente del producto (por **día** argentino de `fecha`; el mismo día, la cargada
  después), el producto toma el costo del último renglón de ese producto y, si el `precioVentaNuevo` del renglón
  **cambió** y difiere del precio actual, el nuevo precio (historial con origen `COMPRA`, motivo "Edición de compra del
  dd/mm/aaaa"). Vaciar `precioVentaNuevo` no revierte el precio. Si al quitar un renglón o cambiar la fecha la compra deja
  de ser la más reciente, el último costo pasa al de la nueva más reciente (el precio no se toca);
- se guarda una entrada en el historial de ediciones y `version` sube 1.

Una edición **sin cambios** (aunque traiga `origen` o `motivo`) responde 200 con el detalle y la **misma** `version`:
no registra movimientos, historial ni auditoría. La pantalla lo detecta comparando `version` ("No había cambios").

Respuesta 200: el detalle de la compra (igual que `GET /compras/{id}`).

Errores:

| HTTP | `code` | Cuándo |
|---|---|---|
| 404 | `compra_no_encontrada` | La compra no existe |
| 404 | `empresa_no_encontrada` | `empresaId` no existe |
| 404 | `producto_no_encontrado` | Un renglón nuevo trae un producto inexistente |
| 409 | `version_desactualizada` | Otra persona editó la compra después de que se leyó |
| 409 | `compra_consumida` | Bajar o quitar unidades que ya salieron del lote (ventas, ajustes o combos). `fields` trae **todos** los renglones rechazados |
| 409 | `operacion_concurrente` | Otra operación se cruzó con esta (ya se reintentó 3 veces): volver a intentar |
| 422 | `datos_invalidos` | `fields`: `fecha`, `notas`, `motivo`, `origen`, `version`, `items`, `items[i].id`, `items[i].productoId`, `items[i].cantidad`, `items[i].costoUnitario`, `items[i].precioVentaNuevo`, `items[i].vencimiento` |
| 422 | `producto_no_comprable` | Un renglón nuevo es un combo |

```json
// 409 compra_consumida
{ "error": { "code": "compra_consumida", "message": "Hay unidades de esta compra que ya salieron del stock",
  "fields": {
    "items[0].cantidad": "Kaiak Clásico: de las 4 unidades de esta compra ya salieron 3 (ventas o ajustes). Solo podés bajar hasta 3 unidades.",
    "quitados[0192f3a0-…]": "Ekos Jabón: ya salió 1 de las 10 unidades; no se puede quitar. Podés bajar la cantidad hasta 1."
  } } }
```
`items[i]` es la posición en el `items` enviado; `quitados[<id>]` es el `id` del renglón que se quiso quitar.

**Detalle (`GET /compras/{id}`)** — campos nuevos:
```json
{
  "id": "…", "fecha": "2026-10-19T15:00:00Z", "empresaId": "…", "empresa": "Avon", "total": "234120.00",
  "productos": 2, "unidades": 16, "notas": "Pedido ciclo 15",
  "version": 3, "editadaEn": "2026-10-27T13:05:12.345Z",
  "items": [
    { "id": "…", "productoId": "…", "sku": "P1", "producto": "Kaiak Clásico", "cantidad": 6,
      "costoUnitario": "35520.00", "subtotal": "213120.00", "precioVentaNuevo": "62000.00",
      "vencimiento": "2027-09-30", "consumido": 1, "precioVentaActual": "62000.00" }
  ],
  "ediciones": [
    { "id": "…", "fecha": "2026-10-27T13:05:12.345Z", "origen": "WEB", "motivo": "Error de carga", "version": 3,
      "cabecera": [
        { "campo": "fecha", "antes": "2026-10-20T15:00:00.000Z", "despues": "2026-10-19T15:00:00.000Z" },
        { "campo": "empresa", "antes": "Natura", "despues": "Avon", "antesId": "…", "despuesId": "…" },
        { "campo": "notas", "antes": null, "despues": "Pedido ciclo 15" }
      ],
      "renglones": [
        { "accion": "MODIFICADO", "itemId": "…", "productoId": "…", "producto": "Kaiak Clásico",
          "cambios": [ { "campo": "cantidad", "antes": 4, "despues": 6 } ] }
      ] }
  ]
}
```
- `editadaEn`: `null` si nunca se editó. También viene en cada compra del listado (marca "Editada el …").
- `ediciones`: de la más reciente a la más vieja (vacío si nunca se editó). `renglones[].accion`: `MODIFICADO`,
  `AGREGADO` (todos los `antes` en `null`) o `QUITADO` (todos los `despues` en `null`). `campo` de renglón:
  `cantidad` (número), `costoUnitario` y `precioVentaNuevo` (texto "0.00"), `vencimiento` (`AAAA-MM-DD`). Los nombres
  son los que tenían al editar.
- `items[].precioVentaActual` (desde US-001b, solo lectura): precio de venta **vigente hoy** del producto en el catálogo
  (texto `"0.00"`; `null` si el catálogo no lo conoce). Se obtiene con una sola consulta al catálogo para todos los
  productos de la compra. Sirve para mostrar "el precio pasará de $A a $B"; el servidor decide si realmente cambia.
- `items[].consumido` (desde US-001b): unidades del lote de ese renglón que ya salieron (cantidad − stock del lote).
  La cantidad mínima editable es `max(consumido, 1)`; si `consumido > 0` el renglón no se puede quitar.

## Ventas

| Método | Ruta | Detalle |
|---|---|---|
| POST | `/ventas` | Cobrar (ver abajo) |
| GET | `/ventas` | Paginado, la más reciente primero. Filtros: `desde`, `hasta`, `estado` (CONFIRMADA/ANULADA), `metodo` |
| GET | `/ventas/{id}` | Detalle: renglones, pagos, costo, ganancia y vuelto |
| POST | `/ventas/{id}/anular` | `{motivo}` — devuelve el stock a los lotes de donde salió |

```json
{
  "items": [ { "productoId": "…", "cantidad": 1 }, { "productoId": "…", "cantidad": 2 } ],
  "pagos": [ { "metodo": "QR", "monto": "40000" }, { "metodo": "EFECTIVO", "monto": "50000" } ],
  "origen": "WEB",
  "permitirSinStock": false
}
```

**Precios:** se toman del catálogo en el momento del cobro (el frontend no los envía).

**Pagos combinados** (`EFECTIVO`, `TRANSFERENCIA`, `QR`, `TARJETA`):
- Transferencia, QR y tarjeta no pueden superar el total (no dan vuelto). Puede haber varios.
- Efectivo: un solo renglón con **lo que entregó el cliente**; cubre lo que falta y el resto es vuelto.
- La suma aplicada debe ser exactamente el total. Si falta: 409 `pagos_insuficientes` ("Faltan $500.00…").
- Cada pago queda guardado con su método y monto; el efectivo además con `montoRecibido` y `vuelto`.

Cálculo para la pantalla (mismo criterio que el servidor):
`falta = total − (suma de no-efectivo) − efectivo` (mínimo 0); `vuelto = efectivo − (total − suma de no-efectivo)` (mínimo 0).

**Stock:** si no alcanza responde 409 `stock_insuficiente` con el nombre del producto. Si la cajera confirma que
igual lo tiene, se reenvía con `"permitirSinStock": true` (el stock queda negativo hasta corregirlo).
Los combos descuentan el stock de sus componentes.

**Ganancia:** `costoTotal` sale del costo de los lotes vendidos (o del último costo del producto si el lote no
lo tiene). `ganancia = total − costoTotal`. Si algún producto no tenía costo, `costoCompleto` es `false` y la
ganancia es aproximada (mayor que la real).

## Sincronización (escritorio)

La usa la app de escritorio para vender sin conexión. Requiere sesión como el resto.

| Método | Ruta | Detalle |
|---|---|---|
| GET | `/sync/catalogo` | Copia completa del catálogo: todos los productos (activos e inactivos) con stock, y los componentes de los combos |
| POST | `/sync/ventas` | Sube ventas cobradas sin conexión (hasta 50 por envío) |

```json
// GET /sync/catalogo
{ "generadaEn": "…",
  "productos": [ { "id": "…", "sku": "P1", "codigoBarras": null, "nombre": "Kaiak", "tipo": "SIMPLE",
                   "empresa": "Natura", "precioVenta": "59200.00", "stock": 3, "activo": true, "version": 2 } ],
  "componentes": [ { "comboId": "…", "productoId": "…", "cantidad": 2 } ] }

// POST /sync/ventas
{ "ventas": [ { "id": "0192f3a0-…", "fecha": "2026-10-24T18:30:00Z",
                "items": [ { "productoId": "…", "cantidad": 1, "precio": "55000" } ],
                "pagos": [ { "metodo": "EFECTIVO", "monto": "60000" } ] } ] }
// -> 200, un resultado por venta y en el mismo orden
{ "resultados": [ { "id": "0192f3a0-…", "estado": "registrada" } ] }
```

- `id` es un UUID generado en el escritorio: **reenviar una venta no la duplica** (`estado: "ya_existia"`).
- Se respetan la fecha y los precios cobrados. Si la fecha está en el futuro (reloj de la PC adelantado) se usa la
  hora del servidor. El origen queda `DESKTOP`.
- Se registra aunque no haya stock (queda negativo) o el producto se haya desactivado después.
- `estado: "rechazada"` trae `error: {code, message}`: datos inválidos, reenviarla no sirve.
- Si el servidor falla (500) no se registra ninguna respuesta parcial: el escritorio reintenta el envío completo.

### Handler local del escritorio

La pantalla del escritorio habla con un handler local (`internal/escritorio`), no directo con el servidor:

| Ruta | Con conexión | Sin conexión |
|---|---|---|
| `POST /api/ventas` | Se cobra en local y se sube enseguida | Se cobra en local y queda en la cola |
| `GET /api/productos`, `/api/productos/codigo/{c}` | Del servidor | De la copia (`desdeCopia: true`); busca `q`, filtra `estado` y `tipo` |
| `POST /api/auth/login` | En el servidor (se recuerda la sesión) | Con la última contraseña usada en esta PC |
| `GET /api/escritorio/estado` | `{enLinea, pendientes, rechazadas, necesitaIngresar, ultimaSincronizacion, copiaDelCatalogo, ultimoError}` | ídem |
| Resto de `/api/*` | Se reenvía al servidor | 503 `sin_conexion` |

La venta local usa el mismo caso de uso que el servidor (precios del catálogo, combos, aviso de
`stock_insuficiente` y reglas de pago); el stock que muestra es el de la copia menos lo vendido en la PC que la
copia todavía no refleja.

## Caja (gastos y retiros)

Dinero que sale de la caja y que **no** es compra de mercadería (servicios, transporte, retiro personal…).
Los gastos no se borran: se anulan.

| Método | Ruta | Detalle |
|---|---|---|
| GET | `/caja/gastos` | Paginado. Filtros: `desde`, `hasta`, `categoria`, `incluirAnulados=true`. Incluye `totalMonto` (suma de los filtrados, sin anulados) |
| POST | `/caja/gastos` | `{fecha?, monto, motivo, categoria?, metodo?}` (método por defecto EFECTIVO) |
| POST | `/caja/gastos/{id}/anular` | `{motivo}` |
| GET | `/caja/categorias` | Sugeridas + las ya usadas |

## Estadísticas

Todas aceptan `?periodo=dia|semana|mes|anio|rango` (por defecto `dia`; la semana va de lunes a domingo) y, para
`rango`, `&desde=AAAA-MM-DD&hasta=AAAA-MM-DD` (máximo un año). Las fechas son de Argentina.
Solo cuentan las ventas confirmadas y los gastos no anulados.

| Método | Ruta | Detalle |
|---|---|---|
| GET | `/estadisticas/resumen` | Tablero (ver abajo) |
| GET | `/estadisticas/ventas-por-dia` | Una fila por día del período, con ceros: `[{fecha, cantidad, total, ganancia}]` (no admite `anio`) |
| GET | `/estadisticas/ventas-por-hora` | 24 filas: `[{hora, cantidad, total}]` |
| GET | `/estadisticas/mas-vendidos` | `&limite=10` (máx. 50) `&orden=unidades|importe`: `[{productoId, sku, nombre, tipo, unidades, total, ganancia}]` |

Resumen:
```json
{
  "periodo": { "tipo": "dia", "desde": "…", "hasta": "…" },
  "ventas": { "cantidad": 3, "total": "98900.00", "costo": "59340.00", "ganancia": "39560.00",
              "margen": 40, "ticketPromedio": "32966.67", "anuladas": 0, "renglonesSinCosto": 0 },
  "porMetodo": [ { "metodo": "EFECTIVO", "monto": "34700.00", "cantidad": 2, "porcentaje": 35.09 }, … ],
  "compras": { "cantidad": 1, "total": "112800.00" },
  "gastos": { "cantidad": 1, "total": "8000.00" },
  "gananciaNeta": "31560.00",
  "efectivoNeto": "26700.00",
  "anterior": { "total": "0.00", "ganancia": "0.00", "variacionTotal": null, "variacionGanancia": null }
}
```
- `margen` = ganancia / vendido (%). `gananciaNeta` = ganancia − gastos de caja.
- `efectivoNeto` = cobrado en efectivo − gastos pagados en efectivo (lo que debería haber en caja).
- `anterior` compara con el mismo tramo del período previo (ej. hoy hasta esta hora contra ayer hasta la misma hora).
  `variacion*` es `null` si el período anterior no tuvo ventas.
- Si `renglonesSinCosto > 0`, algunos productos vendidos no tienen costo cargado y la ganancia es aproximada.
- Las compras de mercadería se informan aparte y **no** se restan de la ganancia (la ganancia ya descuenta
  el costo de lo vendido).
