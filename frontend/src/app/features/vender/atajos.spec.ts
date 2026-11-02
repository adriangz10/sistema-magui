import { METODOS } from '../../core/api/ventas.api';
import { ATAJOS_PAGO, TECLA_BUSCAR, TECLA_COBRAR, teclaDe } from './atajos';

describe('atajos del punto de venta', () => {
  it('cada forma de pago tiene su tecla y no se repiten', () => {
    const teclas = METODOS.map((m) => teclaDe(m.valor));
    expect(teclas).toEqual(['F3', 'F4', 'F5', 'F6']);
    expect(new Set([...teclas, TECLA_BUSCAR, TECLA_COBRAR]).size).toBe(6);
  });

  it('la tecla lleva a su forma de pago', () => {
    expect(ATAJOS_PAGO['F3']).toBe('EFECTIVO');
    expect(ATAJOS_PAGO['F6']).toBe('TARJETA');
    expect(ATAJOS_PAGO['F7']).toBeUndefined();
  });
});
