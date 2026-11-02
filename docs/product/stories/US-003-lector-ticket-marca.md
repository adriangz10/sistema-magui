# US-003 Lector de códigos, ticket de venta, nombre y logo del negocio

**Como** dueña del negocio
**quiero** que el sistema reconozca el lector de códigos de barras, poder emitir un ticket por cada venta y
personalizar el nombre y el logo
**para** cobrar más rápido, entregar un comprobante cuando tenga ticketera y que el sistema muestre mi marca.

## Decisiones (2026-11-02)
- D-1. **Lector:** funciona como teclado (USB o Bluetooth), sin instalar nada. Se reconoce por la velocidad: una
  ráfaga de ≥ 4 caracteres con ≤ 50 ms entre teclas terminada en Enter o Tab. En Vender, una lectura se agrega al
  carrito aunque el foco no esté en el buscador, y el Enter del lector no "aprieta" el botón que tenga el foco.
  Configuración tiene un recuadro para probarlo.
- D-2. **Ticket:** se imprime con la impresora del sistema (ticketera térmica instalada en Windows, 58 u 80 mm, o
  cualquier otra) usando el diálogo de impresión del navegador o del escritorio. Opciones: habilitar, imprimir apenas se
  cobra (si no, botón "Imprimir ticket"), ancho, logo, datos bajo el nombre (≤ 6 líneas) y mensaje al pie. Se puede
  reimprimir desde Ventas. Dice "Comprobante no válido como factura" (no es factura electrónica ARCA).
- D-3. **Nombre y logo:** se ven en el menú, la pantalla de ingreso, la pestaña del navegador y el ticket. Logo PNG, JPG
  o WEBP de hasta 512 KB (sin SVG por seguridad), validado por contenido. Nombre y logo son públicos (se ven antes de
  ingresar); la configuración del ticket requiere sesión. Se guarda una copia local para que se vean e impriman sin
  conexión.

## Criterios de aceptación
```gherkin
Escenario: Leer un código con el foco en un botón
  Dado que estoy en Vender y el foco quedó en un botón de pago
  Cuando escaneo el código de un producto activo
  Entonces el producto se agrega al carrito y el botón no se activa

Escenario: Persona escribiendo no es un lector
  Cuando escribo "1234" y Enter a velocidad normal fuera de un campo
  Entonces no se busca ningún producto

Escenario: Ticket al cobrar
  Dado que el ticket está habilitado con "Imprimir apenas se cobra"
  Cuando cobro una venta
  Entonces se abre la impresión con el ticket de esa venta (nombre, datos, renglones, total, pagos, vuelto y pie)

Escenario: Cambiar el logo
  Cuando subo un PNG de 100 KB
  Entonces el logo nuevo se ve en el menú, en el ingreso y en el ticket
  Cuando subo un SVG
  Entonces veo "Usá una imagen PNG, JPG o WEBP"
```

## API
| Método | Ruta | Sesión | Uso |
|---|---|---|---|
| GET | `/api/negocio/marca` | No | `{nombre, tieneLogo, logoVersion}` |
| GET | `/api/negocio/logo?v=` | No | La imagen |
| GET | `/api/negocio` | Sí | Marca + `ticket` |
| PUT | `/api/negocio` | Sí | `{nombre, ticket: {habilitado, imprimirAlCobrar, anchoPapel, mostrarLogo, datos, pie}}` |
| PUT | `/api/negocio/logo` | Sí | Cuerpo = imagen |
| DELETE | `/api/negocio/logo` | Sí | Quita el logo |

## Datos
Migración `00008_negocio`: tabla `negocio` de una sola fila (`id = 1`) con nombre, datos del ticket y logo (BLOB).

## Preguntas abiertas
- Impresión **sin diálogo** (directa a la ticketera por ESC/POS, con corte de papel y apertura del cajón): requiere un
  componente en la app de escritorio. Se puede sumar cuando se elija la ticketera (marca/modelo).
- ¿Se quiere la factura electrónica ARCA en lugar del ticket no fiscal?
