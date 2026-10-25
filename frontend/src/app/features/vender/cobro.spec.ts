import { calcularCobro, PagoCargado } from './cobro';

const p = (metodo: PagoCargado['metodo'], monto: string | null): PagoCargado => ({ metodo, monto });

describe('calcularCobro (mismo criterio que el servidor)', () => {
  const total = 1_000_000; // $10.000

  it('efectivo exacto y con vuelto', () => {
    expect(calcularCobro(total, [p('EFECTIVO', '10000.00')])).toMatchObject({ falta: 0, vuelto: 0, listo: true });
    expect(calcularCobro(total, [p('EFECTIVO', '20000.00')])).toMatchObject({ falta: 0, vuelto: 1_000_000, listo: true });
  });

  it('pago combinado: transferencia + efectivo con vuelto', () => {
    const r = calcularCobro(total, [p('TRANSFERENCIA', '6000.00'), p('EFECTIVO', '5000.00')]);
    expect(r).toMatchObject({ pagado: 1_000_000, falta: 0, vuelto: 100_000, listo: true, problema: null });
  });

  it('muestra cuánto falta', () => {
    const r = calcularCobro(total, [p('QR', '3000.00'), p('EFECTIVO', '2000.00')]);
    expect(r).toMatchObject({ falta: 500_000, vuelto: 0, listo: false });
  });

  it('sin pagos falta todo', () => {
    expect(calcularCobro(total, [])).toMatchObject({ falta: total, listo: false });
  });

  it('tarjeta de más no se permite', () => {
    const r = calcularCobro(total, [p('TARJETA', '10000.01')]);
    expect(r.listo).toBe(false);
    expect(r.problema).toContain('superan el total');
  });

  it('efectivo sobrante cuando ya está cubierto', () => {
    const r = calcularCobro(total, [p('QR', '10000.00'), p('EFECTIVO', '500.00')]);
    expect(r.listo).toBe(false);
    expect(r.problema).toContain('ya está cubierto');
  });

  it('montos vacíos o en cero no completan el cobro', () => {
    expect(calcularCobro(total, [p('QR', null)]).listo).toBe(false);
    expect(calcularCobro(total, [p('QR', '0.00'), p('EFECTIVO', '10000.00')]).problema).toContain('mayores a cero');
  });

  it('carrito vacío no se puede cobrar', () => {
    expect(calcularCobro(0, [p('EFECTIVO', '100.00')]).listo).toBe(false);
  });
});
