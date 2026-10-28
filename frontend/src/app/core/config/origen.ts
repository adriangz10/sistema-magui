import { ES_ESCRITORIO } from './plataforma';

export type OrigenOperacion = 'WEB' | 'MOVIL' | 'DESKTOP';

/** En pantallas táctiles (celular) la operación queda marcada como MOVIL; en el escritorio, DESKTOP. */
export function origenDispositivo(): OrigenOperacion {
  if (ES_ESCRITORIO) {
    return 'DESKTOP';
  }
  return globalThis.matchMedia?.('(pointer: coarse)').matches ? 'MOVIL' : 'WEB';
}
