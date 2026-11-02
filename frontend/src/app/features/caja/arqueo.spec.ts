import { Arqueo } from '../../core/api/caja.api';
import {
  claseDiferencia, diferenciaEnVivo, estadoDiferencia, etiquetaCampo, textoDiferencia, totalesFormulario, validarContado, valorCambio,
} from './arqueo';

const arqueo = (metodo: Arqueo['metodo'], esperado: string): Arqueo =>
  ({ metodo, ventas: '0.00', gastos: '0.00', esperado, contado: null, diferencia: null });

describe('arqueo de caja', () => {
  it('clasifica la diferencia en sobrante, faltante o cuadra', () => {
    expect(estadoDiferencia('0.00')).toBe('cuadra');
    expect(estadoDiferencia('100.00')).toBe('sobrante');
    expect(estadoDiferencia('-0.50')).toBe('faltante');
    expect(estadoDiferencia(null)).toBeNull();
    expect(textoDiferencia('-500.00')).toMatch(/^Faltante \$\s?500,00$/);
    expect(textoDiferencia('100.00')).toMatch(/^Sobrante \$\s?100,00$/);
    expect(textoDiferencia('0.00')).toBe('Cuadra');
    expect(textoDiferencia(null)).toBe('—');
    expect(claseDiferencia('-1.00')).toBe('badge-danger');
    expect(claseDiferencia('1.00')).toBe('badge-warn');
    expect(claseDiferencia('0.00')).toBe('badge-ok');
    expect(claseDiferencia(null)).toBe('badge-neutro');
  });

  it('calcula la diferencia mientras se completa el cierre', () => {
    expect(diferenciaEnVivo('14000.00', '13500.00')).toBe('-500.00');
    expect(diferenciaEnVivo('14000.00', null)).toBeNull();

    const a = [arqueo('EFECTIVO', '14000.00'), arqueo('TRANSFERENCIA', '3000.00'), arqueo('QR', '0.00'), arqueo('TARJETA', '0.00')];
    expect(totalesFormulario(a, { EFECTIVO: '13500.00', TRANSFERENCIA: '3000.00', QR: '0.00' })).toEqual(
      { esperado: '17000.00', contado: null, diferencia: null });
    expect(totalesFormulario(a, { EFECTIVO: '13500.00', TRANSFERENCIA: '3100.00', QR: '0.00', TARJETA: '0.00' })).toEqual(
      { esperado: '17000.00', contado: '16600.00', diferencia: '-400.00' });
  });

  it('exige el contado de los cuatro medios', () => {
    expect(validarContado({ EFECTIVO: '1.00', QR: null })).toEqual({ faltan: ['TRANSFERENCIA', 'QR', 'TARJETA'] });
    expect(validarContado({ EFECTIVO: '-1.00', TRANSFERENCIA: '0.00', QR: '0.00', TARJETA: '0.00' })).toEqual({ faltan: ['EFECTIVO'] });
    expect(validarContado({ EFECTIVO: '1.00', TRANSFERENCIA: '0.00', QR: '0.00', TARJETA: '0.00' })).toEqual(
      { contado: { EFECTIVO: '1.00', TRANSFERENCIA: '0.00', QR: '0.00', TARJETA: '0.00' } });
  });

  it('nombra los campos del historial de ediciones', () => {
    expect(etiquetaCampo('montoInicial')).toBe('Monto inicial');
    expect(etiquetaCampo('contado.EFECTIVO')).toBe('Contado en efectivo');
    expect(etiquetaCampo('esperado.QR')).toBe('Esperado en QR (recalculado)');
    expect(valorCambio({ campo: 'notas', antes: null, despues: 'hola' }, 'hola')).toBe('hola');
    expect(valorCambio({ campo: 'notas', antes: null, despues: 'hola' }, null)).toBe('—');
    expect(valorCambio({ campo: 'montoInicial', antes: '1000.00', despues: '900.00' }, '900.00')).toMatch(/900,00/);
  });
});
