import { CompraDetalle } from '../../core/api/compras.api';
import {
  armarEdicion, armarItems, cabeceraDe, diaArgentino, formatearValorCambio, hayCambios, hayCambiosRenglones, minimoCantidad,
  renglonesDe, resumenEfectos, seAplicoEdicion, totalRenglones, validarRenglones,
} from './edicion-compra';

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

const conItems = {
  ...compra,
  items: [
    { id: 'i1', productoId: 'p1', sku: 'P1', producto: 'Kaiak Clásico', cantidad: 4, costoUnitario: '35520.00', subtotal: '142080.00',
      precioVentaNuevo: '62000.00', vencimiento: '2027-09-30', consumido: 3 },
    { id: 'i2', productoId: 'p2', sku: 'P2', producto: 'Ekos Jabón', cantidad: 10, costoUnitario: '2100.00', subtotal: '21000.00',
      precioVentaNuevo: null, vencimiento: null, consumido: 0 },
  ],
} as CompraDetalle;

describe('edicion de renglones (US-001b)', () => {
  it('el mínimo es lo consumido y nunca menos de 1', () => {
    expect(minimoCantidad(3)).toBe(3);
    expect(minimoCantidad(0)).toBe(1);
  });

  it('bloquea bajar la cantidad por debajo de lo consumido', () => {
    const rs = renglonesDe(conItems);
    expect(validarRenglones(rs)).toEqual({});
    rs[0].cantidad = 2;
    expect(validarRenglones(rs)['items[0].cantidad']).toBe('Ya salieron 3 unidades de este lote, no podés dejar menos de 3');
    rs[0].cantidad = 3;
    expect(validarRenglones(rs)).toEqual({});
    rs[1].cantidad = 0;
    expect(validarRenglones(rs)['items[1].cantidad']).toBe('Mayor a cero');
  });

  it('valida costo y precio nuevo', () => {
    const rs = renglonesDe(conItems);
    rs[0].costo = null;
    rs[0].precioNuevo = '0.00';
    const e = validarRenglones(rs);
    expect(e['items[0].costoUnitario']).toBeDefined();
    expect(e['items[0].precioVentaNuevo']).toBeDefined();
  });

  it('detecta cambios solo si cambió algún campo del renglón', () => {
    const rs = renglonesDe(conItems);
    expect(hayCambiosRenglones(conItems, rs)).toBe(false);
    rs[1].vencimiento = '2027-12-31';
    expect(hayCambiosRenglones(conItems, rs)).toBe(true);
    const rs2 = renglonesDe(conItems);
    rs2[0].costo = '36000.00';
    expect(hayCambiosRenglones(conItems, rs2)).toBe(true);
  });

  it('arma items completos con id y el total recalculado', () => {
    const rs = renglonesDe(conItems);
    rs[0].cantidad = '6';
    rs[0].costo = '36000.00';
    rs[0].precioNuevo = '65000.00';
    const items = armarItems(rs);
    expect(items[0]).toEqual({ id: 'i1', productoId: 'p1', cantidad: 6, costoUnitario: '36000.00', precioVentaNuevo: '65000.00', vencimiento: '2027-09-30' });
    expect(items[1]).toEqual({ id: 'i2', productoId: 'p2', cantidad: 10, costoUnitario: '2100.00', precioVentaNuevo: null, vencimiento: null });
    expect(totalRenglones(rs)).toBe('237000.00');
    const cuerpo = armarEdicion(conItems, cabeceraDe(conItems), 'WEB', items);
    expect(cuerpo.items).toHaveLength(2);
    expect(resumenEfectos(rs)[0]).toBe('El stock de Kaiak Clásico sube 2');
  });

  it('resume el cambio de precio solo si cambió, difiere del actual y avisa la condición', () => {
    const base = { ...conItems, items: conItems.items.map((i) => ({ ...i, precioVentaActual: '62000.00' })) } as CompraDetalle;
    const rs = renglonesDe(base);
    expect(resumenEfectos(rs)).toEqual([]);
    rs[0].precioNuevo = '65000.00';
    expect(resumenEfectos(rs)).toEqual([expect.stringContaining('pasará de')]);
    expect(resumenEfectos(rs)[0]).toContain('si es la compra más reciente de Kaiak Clásico');
    rs[0].precioNuevo = '62000.00';
    expect(resumenEfectos(rs)).toEqual([]);
    rs[1].precioNuevo = '3000.00';
    rs[1].precioActual = null;
    expect(resumenEfectos(rs)[0]).toContain('queda en');
  });
});
