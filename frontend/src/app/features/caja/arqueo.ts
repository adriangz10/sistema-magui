import { Arqueo, CambioCaja, ContadoPorMedio } from '../../core/api/caja.api';
import { Monto } from '../../core/api/modelos';
import { MetodoPago } from '../../core/api/ventas.api';
import { aCentavos, deCentavos, formatearDinero } from '../../shared/formato/dinero';

/** Medios que se concilian, en el orden de la API. */
export const MEDIOS: readonly MetodoPago[] = ['EFECTIVO', 'TRANSFERENCIA', 'QR', 'TARJETA'];

export const NOMBRE_MEDIO: Record<MetodoPago, string> = {
  EFECTIVO: 'Efectivo',
  TRANSFERENCIA: 'Transferencia',
  QR: 'QR',
  TARJETA: 'Tarjeta',
};

/** Para frases como "Contado en efectivo" / "Contado en QR". */
const MEDIO_EN_FRASE: Record<MetodoPago, string> = {
  EFECTIVO: 'efectivo',
  TRANSFERENCIA: 'transferencia',
  QR: 'QR',
  TARJETA: 'tarjeta',
};

export type EstadoDiferencia = 'cuadra' | 'sobrante' | 'faltante';

/** Diferencia = contado − esperado: positiva sobra, negativa falta. */
export function estadoDiferencia(diferencia: Monto | null | undefined): EstadoDiferencia | null {
  if (diferencia === null || diferencia === undefined) {
    return null;
  }
  const c = aCentavos(diferencia);
  return c === 0 ? 'cuadra' : c > 0 ? 'sobrante' : 'faltante';
}

/** "Cuadra", "Sobrante $ 100,00" o "Faltante $ 500,00". */
export function textoDiferencia(diferencia: Monto | null | undefined): string {
  const estado = estadoDiferencia(diferencia);
  if (!estado) {
    return '—';
  }
  if (estado === 'cuadra') {
    return 'Cuadra';
  }
  const monto = formatearDinero(deCentavos(Math.abs(aCentavos(diferencia))));
  return `${estado === 'sobrante' ? 'Sobrante' : 'Faltante'} ${monto}`;
}

/** Badge según la diferencia: verde si cuadra, ámbar si sobra, rojo si falta. */
export function claseDiferencia(diferencia: Monto | null | undefined): string {
  switch (estadoDiferencia(diferencia)) {
    case 'cuadra':
      return 'badge-ok';
    case 'sobrante':
      return 'badge-warn';
    case 'faltante':
      return 'badge-danger';
    default:
      return 'badge-neutro';
  }
}

/** Diferencia mientras se completa el formulario (null si todavía no se ingresó el monto). */
export function diferenciaEnVivo(esperado: Monto, contado: Monto | null | undefined): Monto | null {
  if (contado === null || contado === undefined) {
    return null;
  }
  return deCentavos(aCentavos(contado) - aCentavos(esperado));
}

/** Total de los cuatro medios del formulario y su diferencia contra el esperado. */
export function totalesFormulario(arqueo: readonly Arqueo[], contado: Partial<Record<MetodoPago, Monto | null>>) {
  let esperado = 0;
  let contadoTotal = 0;
  let completo = true;
  for (const a of arqueo) {
    esperado += aCentavos(a.esperado);
    const c = contado[a.metodo];
    if (c === null || c === undefined) {
      completo = false;
    } else {
      contadoTotal += aCentavos(c);
    }
  }
  return {
    esperado: deCentavos(esperado),
    contado: completo ? deCentavos(contadoTotal) : null,
    diferencia: completo ? deCentavos(contadoTotal - esperado) : null,
  };
}

/** Devuelve el contado listo para la API, o los medios que faltan completar (o son negativos). */
export function validarContado(
  contado: Partial<Record<MetodoPago, Monto | null>>,
): { contado: ContadoPorMedio; faltan?: undefined } | { faltan: MetodoPago[]; contado?: undefined } {
  const faltan = MEDIOS.filter((m) => {
    const v = contado[m];
    return v === null || v === undefined || v.startsWith('-');
  });
  if (faltan.length) {
    return { faltan };
  }
  return { contado: { ...(contado as ContadoPorMedio) } };
}

/** Nombre legible del campo de una edición: "contado.EFECTIVO" -> "Contado en efectivo". */
export function etiquetaCampo(campo: string): string {
  const [base, medio] = campo.split('.');
  const enFrase = medio ? (MEDIO_EN_FRASE[medio as MetodoPago] ?? medio) : '';
  switch (base) {
    case 'montoInicial':
      return 'Monto inicial';
    case 'notas':
      return 'Notas';
    case 'contado':
      return `Contado en ${enFrase}`;
    case 'esperado':
      return `Esperado en ${enFrase} (recalculado)`;
    default:
      return campo;
  }
}

/** Valor de un cambio para mostrar: montos formateados, notas tal cual, vacío como "—". */
export function valorCambio(c: CambioCaja, v: string | null): string {
  if (v === null) {
    return '—';
  }
  return c.campo === 'notas' ? v : formatearDinero(v);
}
