import { CompraDetalle } from '../../core/api/compras.api';
import { armarEdicion, cabeceraDe, diaArgentino, formatearValorCambio, hayCambios, seAplicoEdicion } from './edicion-compra';

const compra = {
  id: 'c1', fecha: '2026-10-20T15:00:00.000Z', empresaId: 'e1', empresa: 'Natura', total: '163080.00', productos: 2, unidades: 8,
  notas: null, version: 3, editadaEn: null, items: [], ediciones: [],
} as CompraDetalle;

describe('edicion-compra', () => {
  it('toma el día argentino del instante', () => {
    expect(diaArgentino('2026-10-20T15:00:00Z')).toBe('2026-10-20');
    expect(diaArgentino('2026-10-21T01:00:00Z')).toBe('2026-10-20');
    expect(diaArgentino('2026-10-20')).toBe('2026-10-20');
  });

  it('precarga la cabecera', () => {
    expect(cabeceraDe(compra)).toEqual({ fecha: '2026-10-20', empresaId: 'e1', notas: '', motivo: '' });
  });

  it('detecta cambios y no cuenta el motivo solo', () => {
    const base = cabeceraDe(compra);
    expect(hayCambios(compra, { ...base, motivo: 'Error de carga' })).toBe(false);
    expect(hayCambios(compra, { ...base, notas: '  ' })).toBe(false);
    expect(hayCambios(compra, { ...base, fecha: '2026-10-19' })).toBe(true);
    expect(hayCambios(compra, { ...base, empresaId: '' })).toBe(true);
    expect(hayCambios(compra, { ...base, notas: 'Pedido ciclo 15' })).toBe(true);
  });

  it('arma el payload con version y origen, sin items ni motivo vacío', () => {
    const base = cabeceraDe(compra);
    const sin = armarEdicion(compra, { ...base, notas: ' Pedido ciclo 15 ' }, 'WEB');
    expect(sin).toEqual({ fecha: '2026-10-20', empresaId: 'e1', notas: 'Pedido ciclo 15', version: 3, origen: 'WEB' });
    expect('items' in sin).toBe(false);
    expect('motivo' in sin).toBe(false);
    const con = armarEdicion(compra, { ...base, empresaId: '', motivo: ' Error de carga ' }, 'DESKTOP');
    expect(con.motivo).toBe('Error de carga');
    expect(con.empresaId).toBe('');
    expect(con.origen).toBe('DESKTOP');
  });

  it('distingue edición aplicada de "sin cambios" por la versión', () => {
    expect(seAplicoEdicion(3, { version: 4 })).toBe(true);
    expect(seAplicoEdicion(3, { version: 3 })).toBe(false);
  });

  it('formatea los valores del historial', () => {
    expect(formatearValorCambio('fecha', '2026-10-20T15:00:00.000Z')).toBe('20/10/2026');
    expect(formatearValorCambio('fecha', '2026-10-20T15:00:00Z', '2026-10-20T18:30:00Z')).toBe('20/10/2026, 12:00');
    expect(formatearValorCambio('empresa', null)).toBe('sin empresa');
    expect(formatearValorCambio('notas', null)).toBe('sin notas');
    expect(formatearValorCambio('notas', 'Pedido ciclo 15')).toBe('Pedido ciclo 15');
    expect(formatearValorCambio('cantidad', 6)).toBe('6');
  });
});
