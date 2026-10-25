import { MetodoPago } from '../../core/api/ventas.api';
import { aCentavos } from '../../shared/formato/dinero';

/** Un pago cargado en pantalla. En efectivo, el monto es lo que entregó el cliente. */
export interface PagoCargado {
  metodo: MetodoPago;
  monto: string | null; // formato API ("5000.00") o null si está vacío/inválido
}

export interface EstadoCobro {
  total: number; // centavos
  pagado: number; // aplicado al total
  falta: number;
  vuelto: number;
  /** Mensaje si los pagos no son válidos (ej. tarjeta de más). */
  problema: string | null;
  /** Se puede cobrar: hay total, no falta nada y no hay problemas. */
  listo: boolean;
}

/**
 * Mismo criterio que el servidor (backend/internal/ventas/domain/pagos.go):
 * transferencia, QR y tarjeta no pueden pasarse del total; el efectivo cubre lo que falta
 * y el excedente es vuelto.
 */
export function calcularCobro(totalCentavos: number, pagos: readonly PagoCargado[]): EstadoCobro {
  let noEfectivo = 0;
  let efectivo = 0;
  let hayEfectivo = false;
  for (const p of pagos) {
    const c = aCentavos(p.monto);
    if (p.metodo === 'EFECTIVO') {
      efectivo += c;
      hayEfectivo = true;
    } else {
      noEfectivo += c;
    }
  }
  const restaSinEfectivo = totalCentavos - noEfectivo;
  const aplicadoEfectivo = Math.max(0, Math.min(efectivo, restaSinEfectivo));
  const pagado = Math.min(noEfectivo, totalCentavos) + aplicadoEfectivo;
  const falta = Math.max(0, totalCentavos - noEfectivo - efectivo);
  const vuelto = Math.max(0, efectivo - Math.max(restaSinEfectivo, 0));

  let problema: string | null = null;
  if (noEfectivo > totalCentavos) {
    problema = 'Los pagos con transferencia, QR o tarjeta superan el total (no dan vuelto)';
  } else if (hayEfectivo && efectivo > 0 && restaSinEfectivo === 0) {
    problema = 'El total ya está cubierto: quitá el pago en efectivo';
  } else if (pagos.some((p) => p.monto !== null && aCentavos(p.monto) <= 0)) {
    problema = 'Los montos deben ser mayores a cero';
  }

  return {
    total: totalCentavos,
    pagado,
    falta,
    vuelto,
    problema,
    listo: totalCentavos > 0 && falta === 0 && problema === null && pagos.some((p) => aCentavos(p.monto) > 0),
  };
}
