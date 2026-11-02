import { MetodoPago } from '../../core/api/ventas.api';

/**
 * Atajos del punto de venta. F2 busca, F3–F6 eligen la forma de pago y Enter (en el monto) cobra.
 * Son teclas que el navegador deja usar a la página; F5 además evita recargar sin querer en medio de una venta.
 */
export const TECLA_BUSCAR = 'F2';
export const TECLA_COBRAR = 'F9';

export const ATAJOS_PAGO: Readonly<Record<string, MetodoPago>> = {
  F3: 'EFECTIVO',
  F4: 'TRANSFERENCIA',
  F5: 'QR',
  F6: 'TARJETA',
};

/** Tecla de cada forma de pago, para mostrarla en el botón. */
export function teclaDe(m: MetodoPago): string {
  return Object.entries(ATAJOS_PAGO).find(([, metodo]) => metodo === m)?.[0] ?? '';
}
