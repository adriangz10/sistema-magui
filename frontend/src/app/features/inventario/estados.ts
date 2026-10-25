import { EstadoLote } from '../../core/api/inventario.api';

/** Clase de badge según el estado de vencimiento. */
export function claseEstadoLote(e: EstadoLote): string {
  return { VENCIDO: 'badge-danger', POR_VENCER: 'badge-warn', VIGENTE: 'badge-ok', SIN_FECHA: 'badge-neutro' }[e];
}

/** "Vencido hace 3 días", "Vence en 10 días", "Vence hoy", "Sin fecha". */
export function textoEstadoLote(e: EstadoLote, dias: number | null): string {
  if (e === 'SIN_FECHA' || dias === null) {
    return 'Sin fecha';
  }
  if (dias < 0) {
    return dias === -1 ? 'Venció ayer' : `Venció hace ${-dias} días`;
  }
  if (dias === 0) {
    return 'Vence hoy';
  }
  return dias === 1 ? 'Vence mañana' : `Vence en ${dias} días`;
}
