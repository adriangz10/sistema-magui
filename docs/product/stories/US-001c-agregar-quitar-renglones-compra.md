# US-001c — Agregar y quitar renglones de una compra
Épica: EP-01 Compras de mercadería · Prioridad: Should · Estimación: 5 puntos · Estado: Borrador
Padre: [US-001](US-001-editar-compra.md) · Depende de: [US-001b](US-001b-editar-renglones-compra.md)

## Historia
Como dueña del negocio, quiero agregar productos que olvidé cargar y quitar renglones cargados por error, para que la compra,
el stock y el último costo queden como la mercadería realmente recibida.

## Reglas de negocio
RN-EC-09, RN-EC-10, RN-EC-13, RN-EC-14, RN-EC-15 a RN-EC-20, RN-EC-23, RN-EC-24 (de US-001), más lo ya vigente de US-001a/b.
- RN-C-01: La compra siempre conserva al menos 1 renglón.
- RN-C-02: Quitar exige que el lote del renglón no tenga unidades consumidas; si las tiene, 409 `compra_consumida` con el
  mensaje por renglón ("Ekos Jabón: ya salió 1 de las 10 unidades; no se puede quitar. Podés bajar la cantidad hasta 1").
- RN-C-03: Un renglón agregado y quitado en la misma edición no genera lote ni movimientos.
- RN-C-04: El recálculo del último costo (RN-EC-19) corre también cuando la edición solo cambia la fecha.

## Criterios de aceptación
```gherkin
Antecedentes:
  Dado la compra C1 del 2026-10-20, versión 4, con Kaiak Clásico (4 x 35.520,00) y Ekos Jabón (10 x 2.100,00), total 163.080,00

Escenario: Agregar un renglón
  Cuando agrego "Chronos Crema", cantidad 2, costo 15.000,00, vencimiento 2028-01-31, precioVentaNuevo 24.000,00
  Entonces se crea un lote nuevo con un movimiento COMPRA de +2
  Y el total pasa a 193.080,00
  Y si C1 es la más reciente de Chronos, su último costo es 15.000,00 y su precio de venta 24.000,00 (historial COMPRA)

Escenario: Agregar el mismo producto en otro renglón
  Cuando agrego otro renglón de Kaiak Clásico, 2 unidades, vencimiento 2028-03-31
  Entonces se crea un lote independiente y la compra tiene dos renglones de Kaiak

Escenario: Quitar un renglón sin consumo
  Cuando quito Ekos Jabón (stock del lote 10 de 10)
  Entonces se crea un EDICION_COMPRA de -10 en su lote y el stock de Ekos baja 10
  Y la compra queda con 1 renglón y total 142.080,00
  Y el lote y sus movimientos siguen en el historial de inventario

Escenario: Quitar un renglón ya consumido
  Dado que se vendió 1 unidad de Ekos Jabón de ese lote
  Cuando quito el renglón
  Entonces 409 "compra_consumida" que indica que ya salió 1 de las 10 unidades y no cambia nada

Escenario: No se puede dejar la compra vacía
  Cuando quito todos los renglones
  Entonces 422 "Agregá al menos un producto"

Escenario: Máximo de renglones
  Dado una compra con 300 renglones
  Cuando agrego uno más
  Entonces 422 "Máximo 300 renglones por compra"

Escenario: Agregar un combo
  Cuando agrego el combo "Pack Kaiak + Ekos"
  Entonces 422 "producto_no_comprable: No se puede comprar un combo"

Escenario: Producto inexistente en un renglón nuevo
  Cuando agrego un renglón con producto inexistente junto con otros cambios válidos
  Entonces 404 "producto_no_encontrado" y no se aplica ninguno

Escenario: Quitar la compra más reciente de un producto recalcula el último costo
  Dado que C1 es la más reciente de Kaiak (costo 35.520,00) y existe C0 del 2026-10-10 con Kaiak a 33.000,00
  Y C1 tiene un único renglón de Kaiak
  Cuando quito el renglón de Kaiak de C1 (sin consumo) y dejo Ekos
  Entonces el último costo de Kaiak pasa a 33.000,00
  Y el precio de venta de Kaiak no cambia

Escenario: Quitar y no queda otra compra del producto
  Dado que ninguna otra compra incluye a Ekos Jabón
  Cuando quito su renglón
  Entonces el último costo de Ekos se mantiene (no se borra)

Escenario: Cambiar la fecha cambia cuál es la más reciente
  Dado que C1 (2026-10-20) es la más reciente de Kaiak y C0 (2026-10-10) tiene Kaiak a 33.000,00
  Cuando cambio la fecha de C1 a 2026-10-05
  Entonces C0 pasa a ser la más reciente y el último costo de Kaiak pasa a 33.000,00
  Y el precio de venta no cambia

Escenario: Cambiar la fecha a la más reciente
  Dado que C0 (2026-10-10) es la más reciente de Kaiak y C1 (2026-10-05) tiene Kaiak a 35.520,00
  Cuando cambio la fecha de C1 a 2026-10-20
  Entonces C1 pasa a ser la más reciente y el último costo de Kaiak pasa a 35.520,00

Escenario: Agregar y quitar en la misma edición
  Cuando agrego un renglón y lo quito antes de guardar
  Entonces no se crea lote ni movimiento

Escenario: Combinación de cambios todo o nada
  Dado que agrego un renglón válido y quito otro con consumo
  Cuando guardo
  Entonces 409 y no se agrega el renglón nuevo ni se cambia nada

Escenario: Dos personas editan a la vez
  Dado que A y B abrieron C1 en la versión 4
  Cuando A agrega un renglón y B quita otro con la versión 4
  Entonces a B 409 "version_desactualizada" y se le pide recargar

Escenario: Auditoría
  Cuando guardo agregando Chronos y quitando Ekos
  Entonces el historial de ediciones muestra "Agregado: Chronos Crema x2" y "Quitado: Ekos Jabón x10"

Escenario: Producto inactivo
  Dado que Ekos Jabón está inactivo y sin consumo
  Cuando quito su renglón
  Entonces se acepta; y agregar un producto inactivo no está ofrecido en el buscador de la pantalla
```

## Datos de ejemplo
| campo | valor válido | valor inválido |
|---|---|---|
| producto del renglón nuevo | "Chronos Crema" (simple) | combo; id inexistente |
| cantidad | 2 | 0 |
| costoUnitario | "15000.00" | "-1" |
| vencimiento | 2028-01-31 o vacío | texto no fecha |
| renglones tras editar | 1 a 300 | 0 o 301 |

## Notas de UX
Botón "Agregar producto" (reutiliza `app-buscador-producto`) y "Quitar" por renglón; "Quitar" deshabilitado con explicación
si hay unidades consumidas. Resumen de efectos antes de guardar ("se quitarán 10 unidades de Ekos Jabón del stock").
Pendiente en `docs/ux/`.

## Dependencias
US-001a y US-001b. Decisión técnica del arquitecto sobre la baja de renglón (baja lógica vs `compra_item_id` NULL).

## Fuera de alcance
Cambiar el producto de un renglón (se quita y se agrega); anular la compra completa.
