import { claseEstadoLote, textoEstadoLote } from './estados';

describe('estados de lote', () => {
  it('describe los días de forma natural', () => {
    expect(textoEstadoLote('VENCIDO', -3)).toBe('Venció hace 3 días');
    expect(textoEstadoLote('VENCIDO', -1)).toBe('Venció ayer');
    expect(textoEstadoLote('POR_VENCER', 0)).toBe('Vence hoy');
    expect(textoEstadoLote('POR_VENCER', 1)).toBe('Vence mañana');
    expect(textoEstadoLote('VIGENTE', 120)).toBe('Vence en 120 días');
    expect(textoEstadoLote('SIN_FECHA', null)).toBe('Sin fecha');
  });

  it('usa colores por estado', () => {
    expect(claseEstadoLote('VENCIDO')).toBe('badge-danger');
    expect(claseEstadoLote('POR_VENCER')).toBe('badge-warn');
  });
});
