# US-001b — Editar los renglones existentes de una compra
Épica: EP-01 Compras de mercadería · Prioridad: Should · Estimación: 8 puntos · Estado: Borrador
Padre: [US-001](US-001-editar-compra.md) · Depende de: [US-001a](US-001a-editar-cabecera-compra.md)

## Historia
Como dueña del negocio, quiero corregir cantidad, costo, vencimiento y precio de reventa de los renglones de una compra, para
que el stock, los lotes, el último costo y el precio de venta reflejen lo realmente comprado, sin poder dejar el stock inconsistente.

## Reglas de negocio
RN-EC-01 (producto fijo), RN-EC-02, RN-EC-03, RN-EC-04, RN-EC-05 a RN-EC-08, RN-EC-11 a RN-EC-18, RN-EC-20 a RN-EC-25
(de US-001). En esta rebanada no se agregan ni se quitan renglones (US-001c); la auditoría de US-001a se amplía con los
renglones antes/después.
- RN-B-01: En la pantalla, cada renglón muestra cuántas unidades ya salieron del lote; la cantidad mínima es ese número.
- RN-B-02: Si varios renglones se rechazan, el mensaje 409 los lista todos.
- RN-B-03: Los movimientos de cantidad se hacen solo por la diferencia (cantidad sin cambios = sin movimiento).

## Criterios de aceptación
```gherkin
Antecedentes:
  Dado la compra C1 del 2026-10-20, versión 3, con renglones:
    | producto      | cantidad | costo     | vencimiento | precioVentaNuevo |
    | Kaiak Clásico | 4        | 35.520,00 | 2027-09-30  | 62.000,00        |
    | Ekos Jabón    | 10       | 2.100,00  |             |                  |
  Y total 163.080,00, y C1 es la compra más reciente de ambos productos
  Y el precio de venta actual de Kaiak es 62.000,00

Escenario: Subir la cantidad
  Cuando cambio Kaiak de 4 a 6
  Entonces se crea un movimiento EDICION_COMPRA de +2 en el lote de ese renglón con referencia a C1
  Y el stock de Kaiak sube 2 y el total pasa a 234.120,00

Escenario: Bajar la cantidad sin haber consumido esas unidades
  Dado que se vendió 1 unidad del lote de Kaiak (stock del lote 3)
  Cuando cambio la cantidad de 4 a 3
  Entonces se crea un EDICION_COMPRA de -1 y el lote queda con stock 2

Escenario: Bloqueo al bajar por debajo de lo consumido
  Dado que se vendieron 3 unidades del lote (stock del lote 1)
  Cuando cambio la cantidad de 4 a 2
  Entonces 409 "compra_consumida" con "de las 4 unidades ya salieron 3; solo podés bajar hasta 3"
  Y no cambia nada (ni cabecera, ni otros renglones, ni stock, ni producto)

Escenario: Bajar justo hasta lo consumido
  Dado que se consumieron 3 de las 4 unidades
  Cuando cambio la cantidad de 4 a 3
  Entonces se acepta y el lote queda con stock 0

Escenario: Consumo por ajuste y por combo
  Dado que 2 unidades salieron por un ajuste "rotura" y 1 por la venta de un combo que incluye a Kaiak
  Cuando intento bajar de 4 a 2
  Entonces se rechaza indicando que salieron 3

Escenario: Venta anulada devuelve las unidades
  Dado que se vendieron 3 unidades y esa venta se anuló (stock del lote 4)
  Cuando cambio la cantidad de 4 a 1
  Entonces se acepta con un movimiento de -3

Escenario: Varios renglones bloqueados
  Dado que ambos renglones tienen más consumo que la cantidad pedida
  Cuando intento bajar los dos
  Entonces el 409 lista los dos renglones con su mínimo permitido

Escenario: Cambiar costo y vencimiento
  Cuando cambio el costo de Kaiak a 36.000,00 y el vencimiento a 2027-12-31
  Entonces el lote se actualiza sin movimiento de stock
  Y las ventas pasadas conservan su costo y su ganancia
  Y las próximas ventas usan 36.000,00 y el nuevo vencimiento en FEFO y alertas
  Y el total se recalcula

Escenario: Es la compra más reciente: actualiza último costo y precio
  Cuando cambio el costo de Kaiak a 36.000,00 y el precioVentaNuevo a 65.000,00
  Entonces el último costo de Kaiak pasa a 36.000,00 y su precio de venta a 65.000,00
  Y el historial de precios suma 62.000,00 -> 65.000,00, origen COMPRA, motivo "Edición de compra del 20/10/2026"

Escenario: No es la más reciente
  Dado que existe otra compra de Kaiak con fecha 2026-10-25
  Cuando cambio en C1 el costo a 36.000,00 y el precioVentaNuevo a 65.000,00
  Entonces el lote de C1 se actualiza pero el último costo, el precio de venta y el historial de Kaiak no cambian

Escenario: Desempate por created_at
  Dado otra compra de Kaiak con la misma fecha 2026-10-20 pero creada antes que C1
  Cuando edito el costo de Kaiak en C1
  Entonces C1 se considera la más reciente y actualiza el último costo

Escenario: Precio sin cambios no pisa un precio manual posterior
  Dado que alguien cambió a mano el precio de Kaiak a 70.000,00
  Cuando edito solo la cantidad de Kaiak (precioVentaNuevo sigue en 62.000,00)
  Entonces el precio de venta sigue en 70.000,00

Escenario: Vaciar el precioVentaNuevo
  Cuando dejo vacío el precioVentaNuevo de Kaiak
  Entonces el precio de venta del producto no cambia y el historial anterior queda intacto

Escenario: Precio nuevo igual al actual
  Cuando cambio el precioVentaNuevo a 62.000,00 sin cambiar el precio del producto
  Entonces no se agrega fila al historial

Escenario: Estadísticas
  Dado que octubre muestra compras: 1 por 163.080,00
  Cuando el total pasa a 234.120,00
  Entonces octubre muestra compras: 1 por 234.120,00 y la ganancia y el efectivo neto no cambian

Escenario: Producto inactivo
  Dado que Ekos Jabón fue dado de baja
  Cuando cambio su cantidad de 10 a 12
  Entonces se acepta y el producto sigue inactivo

Escenario: Valores inválidos
  Cuando pongo cantidad 0, costo -1 o precioVentaNuevo 0
  Entonces 422 con el campo "items[i].cantidad|costoUnitario|precioVentaNuevo" y no cambia nada

Escenario: Intentar cambiar el producto de un renglón
  Cuando envío otro productoId para un renglón existente
  Entonces 422 explicando que se debe quitar y agregar otro renglón

Escenario: Venta simultánea contra una baja
  Dado que la edición baja un lote de 4 a 2 y a la vez se vende 1 unidad de ese lote
  Entonces se serializan y el lote nunca queda en negativo; si la venta ganó, la edición se evalúa con el stock restante

Escenario: Dos personas editan a la vez
  Dado que A y B abrieron C1 en la versión 3
  Cuando A guarda y luego B guarda con la versión 3
  Entonces a B 409 "version_desactualizada" y no se aplica nada

Escenario: Todo o nada
  Dado que un renglón es válido y otro está bloqueado por consumo
  Cuando guardo
  Entonces no se aplica ninguno (ni stock, ni lote, ni producto, ni historial de precios)

Escenario: Auditoría de renglones
  Cuando guardo cambios de cantidad y costo
  Entonces el historial de ediciones de la compra muestra por renglón cantidad 4 -> 6 y costo 35.520,00 -> 36.000,00
```

## Datos de ejemplo
| campo | valor válido | valor inválido |
|---|---|---|
| cantidad | 6; o 3 si se consumieron 3 | 0; -2; 2 si se consumieron 3 |
| costoUnitario | "36000.00" | "-1" |
| precioVentaNuevo | "65000.00" o vacío | "0" |
| vencimiento | 2027-12-31 o vacío | texto no fecha |
| version | 3 (la leída) | 2 -> 409 |

## Notas de UX
En el formulario de edición los renglones son editables; por renglón se muestra "ya salieron N de M" y la cantidad no
baja de N. Antes de guardar, resumen de efectos ("el stock de Kaiak sube 2", "el precio de venta de Kaiak pasará de
$62.000 a $65.000"). Mensaje de bloqueo junto al renglón. Pendiente en `docs/ux/`.

## Dependencias
US-001a (versión, auditoría, botón y formulario). Inventario (tipo `EDICION_COMPRA`, ajuste de lote, lectura de consumo
con bloqueo). Catálogo (último costo, precio, historial, compra más reciente).

## Fuera de alcance
Agregar o quitar renglones y el recálculo al cambiar la fecha (US-001c). Anular la compra.
