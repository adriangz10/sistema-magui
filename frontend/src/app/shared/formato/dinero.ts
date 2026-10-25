import { Pipe, PipeTransform } from '@angular/core';

const formato = new Intl.NumberFormat('es-AR', { style: 'currency', currency: 'ARS', minimumFractionDigits: 2 });
const formatoNumero = new Intl.NumberFormat('es-AR', { minimumFractionDigits: 2, maximumFractionDigits: 2 });

/** Convierte "59200.50" (formato de la API) a centavos enteros. */
export function aCentavos(monto: string | null | undefined): number {
  if (!monto) {
    return 0;
  }
  const negativo = monto.startsWith('-');
  const [entero, frac = ''] = monto.replace('-', '').split('.');
  const c = Number(entero) * 100 + Number((frac + '00').slice(0, 2));
  return negativo ? -c : c;
}

/** Convierte centavos enteros a "59200.50". */
export function deCentavos(c: number): string {
  const negativo = c < 0;
  const abs = Math.abs(Math.round(c));
  const texto = `${Math.trunc(abs / 100)}.${String(abs % 100).padStart(2, '0')}`;
  return negativo ? `-${texto}` : texto;
}

/** "59200.5" -> "$ 59.200,50". */
export function formatearDinero(monto: string | number | null | undefined): string {
  if (monto === null || monto === undefined || monto === '') {
    return '—';
  }
  const centavos = typeof monto === 'number' ? Math.round(monto * 100) : aCentavos(monto);
  return formato.format(centavos / 100);
}

/** "59200.5" -> "59.200,50" (sin símbolo, para inputs). */
export function formatearNumero(monto: string): string {
  return formatoNumero.format(aCentavos(monto) / 100);
}

/**
 * Interpreta lo que escribe la usuaria y devuelve el formato de la API ("59200.50"),
 * o null si no es un monto válido. Acepta "59200", "59.200", "59200,5", "59.200,50", "$ 59.200".
 * Regla argentina: la coma es decimal y el punto separa miles; un único punto seguido
 * de 1 o 2 cifras al final se toma como decimal ("59200.5").
 */
export function interpretarDinero(texto: string): string | null {
  let s = texto.replace(/\$/g, '').replace(/\s/g, '');
  if (s === '') {
    return null;
  }
  const negativo = s.startsWith('-');
  s = s.replace(/^[-+]/, '');
  if (s.includes(',')) {
    s = s.replace(/\./g, '').replace(',', '.');
  } else {
    const puntos = (s.match(/\./g) ?? []).length;
    const decimalConPunto = puntos === 1 && /\.\d{1,2}$/.test(s);
    if (!decimalConPunto) {
      s = s.replace(/\./g, '');
    }
  }
  if (!/^\d+(\.\d{1,2})?$/.test(s)) {
    return null;
  }
  const [entero, frac = ''] = s.split('.');
  const valor = `${BigInt(entero).toString()}.${(frac + '00').slice(0, 2)}`;
  return negativo ? `-${valor}` : valor;
}

/** Pipe de plantilla: {{ producto.precioVenta | dinero }} -> "$ 59.200,00". */
@Pipe({ name: 'dinero' })
export class DineroPipe implements PipeTransform {
  transform(monto: string | number | null | undefined): string {
    return formatearDinero(monto);
  }
}
