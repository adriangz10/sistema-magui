import { aCentavos, deCentavos, formatearDinero, interpretarDinero } from './dinero';
import { formatearFecha, hoyISO } from './fecha';

describe('dinero', () => {
  it('interpreta lo que escribe la usuaria', () => {
    const casos: Record<string, string | null> = {
      '59200': '59200.00',
      '59.200': '59200.00',
      '59.200,5': '59200.50',
      '59200,50': '59200.50',
      '59200.5': '59200.50',
      '1.234.567': '1234567.00',
      '$ 5.500': '5500.00',
      '0,05': '0.05',
      '007': '7.00',
      '-200': '-200.00',
      '': null,
      abc: null,
      '1,234': null,
      '1.2.3,4,5': null,
    };
    for (const [texto, esperado] of Object.entries(casos)) {
      expect(interpretarDinero(texto), texto).toBe(esperado);
    }
  });

  it('formatea en pesos argentinos', () => {
    expect(formatearDinero('59200.50').replace(/\s/g, ' ')).toBe('$ 59.200,50');
    expect(formatearDinero(null)).toBe('—');
  });

  it('convierte a centavos y vuelve sin perder precisión', () => {
    expect(aCentavos('0.10') + aCentavos('0.20')).toBe(30);
    expect(deCentavos(aCentavos('59200.05'))).toBe('59200.05');
    expect(deCentavos(-1025)).toBe('-10.25');
  });
});

describe('fecha', () => {
  it('muestra la hora de Argentina', () => {
    expect(formatearFecha('2026-10-25T18:30:00Z')).toBe('25/10/2026, 15:30');
    expect(formatearFecha('2026-10-25T18:30:00Z', 'hora')).toBe('15:30');
    expect(formatearFecha('2026-09-30')).toBe('30/09/2026');
    expect(formatearFecha(null)).toBe('—');
  });

  it('calcula hoy en Argentina', () => {
    // 01:00 UTC del 26 todavía es 25 en Argentina
    expect(hoyISO(new Date('2026-10-26T01:00:00Z'))).toBe('2026-10-25');
  });
});
