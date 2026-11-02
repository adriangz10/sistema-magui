# US-004 Foto opcional de cada producto

**Como** dueña del negocio
**quiero** poder agregarle una foto a un producto (sin que sea obligatorio)
**para** reconocerlo rápido en el listado y al vender.

## Decisiones (2026-11-02)
- D-1. Una foto por producto, opcional. Se agrega, cambia o quita desde el formulario del producto (también al crearlo:
  se sube apenas se guarda el producto).
- D-2. La app achica la foto antes de subirla (lado mayor 800 px, WEBP o JPG, fondo blanco), así una foto de celular de
  varios MB sube rápido. El servidor acepta PNG, JPG o WEBP de hasta 1 MB y la valida por contenido (sin SVG).
- D-3. Se ve en el listado de Productos y en las sugerencias del buscador de Vender. Sin foto (o sin conexión) se
  muestra un ícono.
- D-4. La foto se descarga sin sesión (`<img>` no manda el token); los IDs de producto son UUID y no se pueden adivinar.
- D-5. `producto.imagen_version` solo aumenta: la URL `?v=` se guarda en caché para siempre y nunca muestra una foto vieja.

## Criterios de aceptación
```gherkin
Escenario: Agregar foto al crear un producto
  Cuando creo un producto y elijo una foto de 4 MB del celular
  Entonces el producto se guarda y su foto aparece en el listado

Escenario: Producto sin foto
  Dado un producto sin foto
  Entonces en el listado y en Vender se ve un ícono en lugar de la foto

Escenario: Quitar la foto
  Cuando quito la foto y guardo
  Entonces el producto queda sin foto

Escenario: Archivo no válido
  Cuando subo un SVG o un archivo de más de 1 MB
  Entonces veo un mensaje y el producto no cambia
```

## API
| Método | Ruta | Sesión | Uso |
|---|---|---|---|
| PUT | `/api/productos/{id}/imagen` | Sí | Cuerpo = la foto → `{imagenVersion}` |
| DELETE | `/api/productos/{id}/imagen` | Sí | Quita la foto |
| GET | `/api/imagenes/productos/{id}?v=` | No | La foto |

`ProductoResumen`/`ProductoDetalle` suman `imagenVersion` (0 = sin foto).

## Datos
Migración `00009_producto_imagen`: columna `producto.imagen_version` y tabla `producto_imagen` (foto en BLOB).

## Preguntas abiertas
- ¿Más de una foto por producto (galería), por ejemplo para una tienda online?
- La app de escritorio sin conexión muestra el ícono (las fotos no se copian a la PC). ¿Hace falta verlas offline?
