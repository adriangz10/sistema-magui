import { CompraDetalle } from '../../core/api/compras.api';
import {
  armarEdicion, armarItems, cabeceraDe, detalleRenglonHistorial, diaArgentino, formatearValorCambio, hayCambios, hayCambiosRenglones,
  mapearErroresServidor, minimoCantidad, motivoNoQuitar, renglonNuevo,
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

const chronos = { id: 'p3', nombre: 'Chronos Crema', sku: 'P3', precioVenta: '20000.00', costo: '15000.00' };

describe('agregar y quitar renglones (US-001c)', () => {
  it('manda los nuevos sin id al final y omite los quitados', () => {
    const rs = renglonesDe(conItems);
    rs.splice(1, 0, renglonNuevo(1, chronos));
    rs[1].cantidad = '2';
    rs[2].quitado = true;
    const items = armarItems(rs);
    expect(items).toEqual([
      { id: 'i1', productoId: 'p1', cantidad: 4, costoUnitario: '35520.00', precioVentaNuevo: '62000.00', vencimiento: '2027-09-30' },
      { productoId: 'p3', cantidad: 2, costoUnitario: '15000.00', precioVentaNuevo: null, vencimiento: null },
    ]);
    expect('id' in items[1]).toBe(false);
    expect(totalRenglones(rs)).toBe('172080.00');
  });

  it('bloquea quitar con consumo y el último renglón activo', () => {
    const rs = renglonesDe(conItems);
    expect(motivoNoQuitar(rs[0], rs)).toBe('No se puede quitar. Podés bajar la cantidad hasta 3.');
    expect(motivoNoQuitar({ ...rs[0], consumido: 4 }, rs)).toBe('No se puede quitar porque ya salió todo el lote.');
    expect(motivoNoQuitar(rs[1], rs)).toBeNull();
    rs[1].quitado = true;
    const ultimo = { ...rs[0], consumido: 0 };
    expect(motivoNoQuitar(ultimo, [ultimo, rs[1]])).toBe('La compra necesita al menos un producto.');
    const nuevo = renglonNuevo(2, chronos);
    expect(motivoNoQuitar(nuevo, [ultimo, rs[1], nuevo])).toBeNull();
  });

  it('detecta cambios por agregar, quitar y deshacer', () => {
    const rs = renglonesDe(conItems);
    expect(hayCambiosRenglones(conItems, rs)).toBe(false);
    rs[1].quitado = true;
    expect(hayCambiosRenglones(conItems, rs)).toBe(true);
    rs[1].quitado = false;
    expect(hayCambiosRenglones(conItems, rs)).toBe(false);
    expect(hayCambiosRenglones(conItems, [...rs, renglonNuevo(1, chronos)])).toBe(true);
  });

  it('mapea items[i] al array realmente enviado y quitados[id] queda por id', () => {
    const rs = renglonesDe(conItems);
    rs[0].quitado = true; // en pantalla: [i1 quitado, i2, nuevo]
    const lista = [...rs, renglonNuevo(1, chronos)];
    const e = mapearErroresServidor(lista, {
      'items[0].cantidad': 'sobre i2', 'items[1].costoUnitario': 'sobre el nuevo', 'quitados[i1]': 'no se puede quitar', items: 'general',
    });
    expect(e).toEqual({
      'items[1].cantidad': 'sobre i2', 'items[2].costoUnitario': 'sobre el nuevo', 'quitados[i1]': 'no se puede quitar', items: 'general',
    });
  });

  it('un nuevo puesto antes de un existente igual se envía al final', () => {
    const rs = renglonesDe(conItems);
    const lista = [rs[0], renglonNuevo(1, chronos), rs[1]];
    expect(mapearErroresServidor(lista, { 'items[1].cantidad': 'x' })).toEqual({ 'items[2].cantidad': 'x' });
  });

  it('resume agregado y quitado', () => {
    const rs = renglonesDe(conItems);
    rs[1].quitado = true;
    const nuevo = renglonNuevo(1, chronos);
    nuevo.cantidad = '2';
    expect(resumenEfectos([...rs, nuevo])).toEqual([
      'Se quita Ekos Jabón (−10 del stock)',
      'Se agrega Chronos Crema (+2 al stock)',
    ]);
  });

  it('no deja la compra sin renglones activos ni pasar de 300', () => {
    const rs = renglonesDe(conItems);
    rs.forEach((r) => (r.quitado = true));
    expect(validarRenglones(rs)['items']).toBe('Agregá al menos un producto');
    const muchos = Array.from({ length: 301 }, (_, i) => renglonNuevo(i, chronos));
    expect(validarRenglones(muchos)['items']).toBe('Máximo 300 renglones por compra');
  });

  it('describe renglones agregados y quitados del historial', () => {
    const cambios = [{ campo: 'cantidad', antes: null, despues: 2 }, { campo: 'costoUnitario', antes: null, despues: '15000.00' }];
    expect(detalleRenglonHistorial({ accion: 'AGREGADO', productoId: 'p3', producto: 'Chronos Crema', cambios }).cantidad).toBe('×2');
    const q = detalleRenglonHistorial({
      accion: 'QUITADO', productoId: 'p2', producto: 'Ekos',
      cambios: [{ campo: 'cantidad', antes: 10, despues: null }, { campo: 'vencimiento', antes: '2027-09-30', despues: null }],
    });
    expect(q.cantidad).toBe('×10');
    expect(q.resto).toEqual(['vence 30/09/2027']);
  });
});
