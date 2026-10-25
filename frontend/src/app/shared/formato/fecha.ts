import { Pipe, PipeTransform } from '@angular/core';

const ZONA = 'America/Argentina/Buenos_Aires';

const fechaHora = new Intl.DateTimeFormat('es-AR', {
  timeZone: ZONA, day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', hourCycle: 'h23',
});
const soloFecha = new Intl.DateTimeFormat('es-AR', { timeZone: ZONA, day: '2-digit', month: '2-digit', year: 'numeric' });
const soloHora = new Intl.DateTimeFormat('es-AR', { timeZone: ZONA, hour: '2-digit', minute: '2-digit', hourCycle: 'h23' });

export type FormatoFecha = 'fecha-hora' | 'fecha' | 'hora';

/**
 * Formatea fechas en hora argentina.
 * Acepta instantes ISO ("2026-10-25T18:00:00Z") o días sin hora ("2026-10-25").
 */
export function formatearFecha(valor: string | null | undefined, formato: FormatoFecha = 'fecha-hora'): string {
  if (!valor) {
    return '—';
  }
  if (/^\d{4}-\d{2}-\d{2}$/.test(valor)) {
    const [a, m, d] = valor.split('-');
    return `${d}/${m}/${a}`;
  }
  const f = new Date(valor);
  if (Number.isNaN(f.getTime())) {
    return valor;
  }
  return (formato === 'fecha' ? soloFecha : formato === 'hora' ? soloHora : fechaHora).format(f);
}

/** Fecha de hoy en Argentina como "2026-10-25". */
export function hoyISO(ahora = new Date()): string {
  const partes = new Intl.DateTimeFormat('en-CA', { timeZone: ZONA }).format(ahora);
  return partes; // en-CA ya da AAAA-MM-DD
}

/** Pipe de plantilla: {{ venta.fecha | fecha }} o {{ lote.vencimiento | fecha: 'fecha' }}. */
@Pipe({ name: 'fecha' })
export class FechaPipe implements PipeTransform {
  transform(valor: string | null | undefined, formato: FormatoFecha = 'fecha-hora'): string {
    return formatearFecha(valor, formato);
  }
}
