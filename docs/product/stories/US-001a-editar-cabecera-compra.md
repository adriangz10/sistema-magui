# US-001a — Editar la cabecera de una compra (fecha, empresa, notas)
Épica: EP-01 Compras de mercadería · Prioridad: Should · Estimación: 5 puntos · Estado: Borrador
Padre: [US-001](US-001-editar-compra.md) (reglas RN-EC y decisiones D-n)

## Historia
Como dueña del negocio, quiero tocar "Editar" en una compra de la lista y corregir su fecha, empresa y notas, y ver el
historial de ediciones en el detalle, para arreglar errores de carga sabiendo qué se cambió y cuándo.

## Reglas de negocio
RN-EC-01 (solo cabecera), RN-EC-02 (fecha, notas, empresa existente), RN-EC-03, RN-EC-04, RN-EC-21, RN-EC-22,
RN-EC-23, RN-EC-24 (de US-001). En esta rebanada los renglones, lotes, stock, costos y precios **no se modifican**; el total
tampoco cambia.
- RN-A-01: La empresa es opcional; se puede quitar (la compra queda sin empresa) o cambiar por otra. Se valida que exista
  (igual que al registrar); no se exige que esté activa.
- RN-A-02: Una edición con solo `origen`/`motivo` y sin cambios es "sin cambios" (RN-EC-04).
- RN-A-03: El historial se muestra del más reciente al más viejo con fecha/hora (Argentina, 24 h), origen, motivo y los
  campos cambiados (antes -> después).

## Criterios de aceptación
```gherkin
Antecedentes:
  Dado la compra C1 del 2026-10-20, empresa "Natura", sin notas, versión 1, total 163.080,00, con 2 renglones

Escenario: Corregir fecha, empresa y notas
  Cuando toco "Editar" en C1, cambio fecha a 2026-10-19, empresa a "Avon" y notas a "Pedido ciclo 15" y guardo
  Entonces la lista y el detalle muestran los nuevos datos y la versión pasa a 2
  Y los renglones, el total, el stock, el costo y el precio de los productos no cambian
  Y se informa "Compra actualizada"

Escenario: El formulario muestra los renglones solo de lectura
  Cuando abro "Editar" en C1
  Entonces veo fecha, empresa y notas editables y los renglones con cantidad, costo, vencimiento y subtotal sin poder modificarse

Escenario: Quedan registrados los cambios
  Dado que edité C1 como en el primer escenario desde la web con motivo "Error de carga"
  Cuando abro el detalle de C1
  Entonces el "Historial de ediciones" muestra una entrada con fecha/hora, origen WEB, motivo "Error de carga"
  Y los cambios: fecha 20/10/2026 -> 19/10/2026, empresa Natura -> Avon, notas (vacío) -> "Pedido ciclo 15"

Escenario: Compra sin ediciones
  Cuando abro el detalle de una compra nunca editada
  Entonces el historial dice "Sin ediciones"

Escenario: Quitar la empresa
  Cuando dejo la empresa vacía y guardo
  Entonces la compra queda sin empresa y la edición se audita (Natura -> sin empresa)

Escenario: Mover la compra a otro período en estadísticas
  Dado que octubre muestra compras: 1 por 163.080,00
  Cuando cambio la fecha de C1 a 2026-09-28
  Entonces octubre deja de contarla y septiembre la suma por 163.080,00
  Y la ganancia y el efectivo neto no cambian

Escenario: Guardar sin cambios
  Cuando abro "Editar" y guardo sin modificar nada
  Entonces no se crea entrada de historial ni cambia la versión y se informa "No había cambios"

Escenario: Fecha futura
  Cuando pongo una fecha posterior a ahora + 1 hora
  Entonces 422 en el campo fecha: "La fecha no puede ser futura" y no cambia nada

Escenario: Notas demasiado largas
  Cuando escribo más de 500 caracteres en notas
  Entonces 422 en el campo notas: "Máximo 500 caracteres"

Escenario: Empresa inexistente
  Cuando se envía un empresaId que no existe
  Entonces 404 de empresa no encontrada y no cambia nada

Escenario: Compra inexistente
  Cuando edito un id que no existe
  Entonces 404 "compra_no_encontrada"

Escenario: Dos personas editan a la vez
  Dado que A y B abrieron C1 en la versión 1
  Cuando A guarda y después B guarda con la versión 1
  Entonces a B se le responde 409 "version_desactualizada" con el aviso "Otra persona modificó esta compra, recargá" y no se aplica su cambio
  Y el historial tiene solo la entrada de A

Escenario: Todo o nada
  Dado que falla la grabación de la auditoría
  Cuando guardo un cambio de cabecera
  Entonces no se aplica ningún cambio de la compra
```

## Datos de ejemplo
| campo | valor válido | valor inválido |
|---|---|---|
| fecha | 2026-10-19 | fecha futura (> ahora + 1 h) |
| empresa | "Avon" o vacía | id inexistente |
| notas | "Pedido ciclo 15" | más de 500 caracteres |
| motivo | "Error de carga" o vacío | más de 255 caracteres |
| version | 1 (la leída) | 1 con la compra en versión 2 -> 409; ausente o 0 -> 422 `version` |

## Notas de UX
Botón "Editar" junto a "Ver" en `compras.page.ts` (`aria-label` con la fecha). Reutiliza `compra-form.component.ts` con los
renglones en solo lectura y aviso de que cambiar renglones llega en una próxima versión. Detalle: bloque "Historial de
ediciones" y la marca "Editada el dd/mm/aaaa hh:mm" en la lista. Pendiente en `docs/ux/`.

## Dependencias
Compras registradas (existente). Catálogo para validar empresa. Control de versión optimista. Habilita US-001b.

## Fuera de alcance
Cualquier cambio de renglones, stock, costos o precios (US-001b/c); anular la compra.
