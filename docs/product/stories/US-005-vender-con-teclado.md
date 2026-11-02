# US-005 Vender con teclado y lector, sin mouse

**Como** dueña, cobrando en el mostrador
**quiero** escanear, elegir la forma de pago y cobrar solo con el teclado
**para** atender más rápido.

## Flujo
1. Al entrar a Vender el cursor queda en el buscador: cada código leído se agrega al carrito (también si el foco quedó
   en otro lado, ver US-003).
2. Forma de pago con una tecla: **F3** Efectivo · **F4** Transferencia · **F5** QR · **F6** Tarjeta. El cursor va al
   monto, con el valor seleccionado: en efectivo se escribe lo que entrega el cliente y se ve el vuelto.
3. **Enter** en el monto cobra. Si todavía falta plata, avisa cuánto y no cobra.
4. Enter con el buscador vacío también cobra si ya está todo pagado (si no hay pago elegido, recuerda las teclas).
5. En "Venta registrada", Enter = "Nueva venta" y el cursor vuelve al buscador.
- F2 sigue llevando al buscador y F9 sigue cobrando. Las teclas F3–F6 no hacen nada con un diálogo abierto.
- F5 no recarga la página en Vender (evita perder la venta en curso por error).

## Inicio y Esc (2026-11-02)
- El sistema abre en **Vender** (también después de ingresar), con el cursor en el buscador.
- **Esc** desde cualquier pantalla lleva a Vender. Si hay un diálogo, el menú "Más" o la lista de sugerencias abiertos,
  el primer Esc solo los cierra (no se pierde un formulario a medio cargar). En Vender, Esc vuelve el cursor al buscador.
- Botones de forma de pago: ícono a la izquierda, nombre arriba y tecla abajo; el nombre completo entra desde 390 px
  hasta 1920 px de ancho.

## Verificación
Probado con Chrome real enviando teclas (ráfaga de lector, F3, monto, Enter): la venta llegó al servidor con el vuelto
correcto y el foco volvió al buscador.
