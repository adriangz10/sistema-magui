# 0005. Auditar cada edición de compra en `compra_edicion` con un documento JSON de cambios (antes/después por campo)

- Estado: Propuesto
- Fecha: 2026-10-27
- Decisores: Software Architect + Database Engineer; D-5 y D-11 (cliente)
- Historias relacionadas: US-001a (cabecera), US-001b (renglones modificados), US-001c (agregados/quitados)

## Contexto
Cada edición efectiva guarda fecha/hora, origen, motivo opcional (≤ 255) y los valores antes/después (RN-EC-24). El
detalle muestra "Historial de ediciones" con solo los campos que cambiaron (RN-A-03), por ejemplo
"fecha 20/10/2026 → 19/10/2026", "Kaiak Clásico: cantidad 4 → 6", "Agregado: Chronos Crema x2". El contenido crece por
parte (a: cabecera; b: renglones modificados; c: agregados/quitados) y no debe obligar a rehacer la tabla. Sin "quién".
Se lee siempre junto con la compra; no hay consultas analíticas sobre los cambios.

## Alternativas consideradas
### A. Dos columnas JSON `antes` y `despues` con fotos (completas o parciales) de la compra
Pros: simple de escribir. Contras: para mostrar "solo lo que cambió" hay que comparar en cada lectura y emparejar
renglones; fotos completas de compras de 300 renglones repiten todo en cada edición.
### B. Una columna JSON `cambios` con la lista de diferencias, cada una con `antes` y `despues`
Pros: guarda exactamente lo que se muestra; tamaño proporcional al cambio; uniforme para cabecera y renglones
(agregado = antes null, quitado = después null); el estado completo de cualquier momento se reconstruye desde el estado
actual hacia atrás. Contras: no se puede indexar por campo (no se necesita).
### C. Columnas fijas (`fecha_antes`, `fecha_despues`, …) + tabla hija `compra_edicion_renglon`
Pros: tipado fuerte en la base. Contras: dos tablas y una migración por cada campo editable nuevo; para la parte a ya
habría que crear la tabla hija vacía.

## Decisión
Elegimos **B**.

Tabla `compra_edicion (id, compra_id, fecha, origen, motivo, version, cambios)`; MySQL `JSON`, SQLite
`TEXT CHECK (json_valid(cambios))`; `UNIQUE (compra_id, version)` (`version` = versión de la compra que dejó la
edición; además ordena el historial y evita registrar dos veces la misma edición). Detalle en `docs/data/editar-compra.md`.

Formato del documento (versión 1, se escribe completo desde la parte a; `renglones` es `[]` en a):
```json
{
  "v": 1,
  "cabecera": [
    { "campo": "fecha",   "antes": "2026-10-20T15:00:00.000Z", "despues": "2026-10-19T15:00:00.000Z" },
    { "campo": "empresa", "antes": "Natura", "despues": "Avon", "antesId": "…", "despuesId": "…" },
    { "campo": "notas",   "antes": null, "despues": "Pedido ciclo 15" }
  ],
  "renglones": [
    { "accion": "MODIFICADO", "itemId": "…", "productoId": "…", "producto": "Kaiak Clásico",
      "cambios": [ { "campo": "cantidad", "antes": 4, "despues": 6 },
                   { "campo": "costoUnitario", "antes": "35520.00", "despues": "36000.00" } ] },
    { "accion": "AGREGADO", "itemId": "…", "productoId": "…", "producto": "Chronos Crema",
      "cambios": [ { "campo": "cantidad", "antes": null, "despues": 2 },
                   { "campo": "costoUnitario", "antes": null, "despues": "15000.00" } ] },
    { "accion": "QUITADO", "itemId": "…", "productoId": "…", "producto": "Ekos Jabón",
      "cambios": [ { "campo": "cantidad", "antes": 10, "despues": null } ] }
  ]
}
```
Reglas del formato:
- `campo` de cabecera: `fecha` (instante ISO UTC), `empresa` (nombre, con `antesId`/`despuesId`), `notas`.
- `campo` de renglón: `cantidad` (número), `costoUnitario` y `precioVentaNuevo` (texto decimal "0.00"),
  `vencimiento` ("AAAA-MM-DD"). AGREGADO lista todos los campos informados; QUITADO lista `cantidad` y `costoUnitario`.
- Nombres de producto y empresa se guardan como estaban **al editar** (el historial no cambia si se renombran después).
- Orden: cabecera en el orden fijo de arriba; renglones en orden de `compra_item.id`.
- `v` permite cambiar el formato sin migrar filas viejas: el lector acepta `v = 1` y falla en voz alta con otra.
- Una edición sin cambios no crea fila (RN-EC-04). `origen`/`motivo` no cuentan como cambio (RN-A-02).
- La API devuelve el documento tal cual (sin `v`) dentro de `ediciones[]` del detalle (ver `docs/API.md`).

## Consecuencias
- Positivas: la parte a ya escribe el formato final; b y c solo agregan entradas a `renglones`. El frontend dibuja el
  historial sin lógica de comparación.
- Negativas: los valores van como texto/número JSON (sin tipos de base); una migración de formato futura debe
  convivir con `v = 1`.
- Datos personales: no hay (sin "quién"). El motivo es texto libre: la pantalla debe aclarar que no se escriban datos
  de clientes.
- Qué hay que hacer ahora: migración 00004 (parte a), `domain.Edicion` y su serialización, repositorio
  `GuardarEdicion`/`Ediciones`.
- Cuándo revisarla: si se piden reportes sobre ediciones (ej. "todas las bajas de cantidad del mes"), pasar a la opción C
  o a columnas generadas.
