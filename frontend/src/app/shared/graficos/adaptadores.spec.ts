import { columnasPorDia, columnasPorHora, etiquetaDia, filasMasVendidos } from './adaptadores';

describe('adaptadores de gráficos', () => {
  it('rotula días en castellano', () => {
    expect(etiquetaDia('2026-10-19')).toBe('lun 19');
    expect(etiquetaDia('2026-10-25')).toBe('dom 25');
    expect(etiquetaDia('2026-10-25', true)).toBe('25/10');
  });

  it('convierte ventas por día a columnas en pesos', () => {
    const [c] = columnasPorDia([{ fecha: '2026-10-25', cantidad: 3, total: '98900.00', ganancia: '39560.00' }]);
    expect(c.valor).toBe(98900);
    expect(c.texto.replace(/\s/g, ' ')).toBe('$ 98.900,00');
    expect(c.detalle).toContain('3 ventas');
  });

  it('muestra al menos el horario comercial y amplía si hubo ventas fuera de él', () => {
    const horas = Array.from({ length: 24 }, (_, hora) => ({ hora, cantidad: hora === 22 ? 1 : 0, total: hora === 22 ? '100.00' : '0.00' }));
    const cols = columnasPorHora(horas);
    expect(cols[0].etiqueta).toBe('9 h');
    expect(cols[cols.length - 1].etiqueta).toBe('22 h');
  });

  it('arma el ranking por unidades o por importe', () => {
    const lista = [{ productoId: 'a', sku: 'A', nombre: 'Crema', tipo: 'SIMPLE', unidades: 5, total: '27500.00', ganancia: null }];
    expect(filasMasVendidos(lista, 'unidades')[0]).toMatchObject({ valor: 5, texto: '5 u.' });
    expect(filasMasVendidos(lista, 'importe')[0].valor).toBe(2_750_000);
  });
});
