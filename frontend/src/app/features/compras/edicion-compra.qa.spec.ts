import { CompraDetalle } from '../../core/api/compras.api';
import {
  armarItems, detalleRenglonHistorial, hayCambiosRenglones, mapearErroresServidor, motivoNoQuitar, renglonNuevo, renglonesDe,
  resumenEfectos, totalRenglones, validarRenglones,
} from './edicion-compra';

// QA US-001c: casos borde de la lógica de edición y del historial (sin "undefined"/"null" en pantalla).
const detalle = {
  id: 'c1', fecha: '2026-10-20T15:00:00.000Z', empresaId: null, empresa: null, total: '163080.00', productos: 2, unidades: 14,
  notas: null, version: 4, editadaEn: null, ediciones: [],
  items: [
    { id: 'i1', productoId: 'p1', producto: 'Kaiak Clásico', sku: 'K1', cantidad: 4, costoUnitario: '35520.00', precioVentaNuevo: '62000.00', vencimiento: '2027-09-30', consumido: 0, precioVentaActual: '62000.00' },
    { id: 'i2', productoId: 'p2', producto: 'Ekos Jabón', sku: 'E1', cantidad: 10, costoUnitario: '2100.00', precioVentaNuevo: null, vencimiento: null, consumido: 0, precioVentaActual: '3000.00' },
  ],
} as unknown as CompraDetalle;
const chronos = { id: 'p3', nombre: 'Chronos Crema', sku: 'C1', precioVenta: '20000.00', costo: '12000.00' };

describe('QA US-001c: edición', () => {
  it('un renglón nuevo agregado y quitado antes de guardar no deja cambios ni se envía', () => {
    const rs = renglonesDe(detalle);
    const nuevo = renglonNuevo(1, chronos);
    expect(hayCambiosRenglones(detalle, [...rs, nuevo])).toBe(true);
    // quitar un nuevo lo descarta de la lista (compra-form lo filtra): el estado vuelve a "sin cambios"
    expect(hayCambiosRenglones(detalle, rs)).toBe(false);
    expect(armarItems(rs).every((i) => i.id)).toBe(true);
  });

  it('un producto sin costo conocido da error local de costo en el renglón nuevo', () => {
    const rs = [...renglonesDe(detalle), renglonNuevo(1, { ...chronos, costo: null })];
    expect(validarRenglones(rs)['items[2].costoUnitario']).toBe('Ingresá el costo');
  });

  it('300 activos es válido y quitados no cuentan para el tope', () => {
    const rs = Array.from({ length: 300 }, (_, i) => renglonNuevo(i, chronos));
    expect(validarRenglones(rs)['items']).toBeUndefined();
    const con301 = [...rs, { ...renglonNuevo(999, chronos), quitado: true }];
    expect(validarRenglones(con301)['items']).toBeUndefined();
  });

  it('el total ignora los quitados y respeta decimales', () => {
    const rs = renglonesDe(detalle);
    const nuevo = renglonNuevo(1, chronos);
    nuevo.cantidad = '3';
    nuevo.costo = '1234.56';
    rs[1].quitado = true;
    expect(totalRenglones([...rs, nuevo])).toBe('145783.68');
  });

  it('el resumen de un quitado no habla de precios y un nuevo con precio avisa la condición', () => {
    const rs = renglonesDe(detalle);
    rs[0].quitado = true;
    rs[0].precioNuevo = '99999.00'; // aunque se haya tocado antes de quitar
    expect(resumenEfectos(rs)).toEqual(['Se quita Kaiak Clásico (−4 del stock)']);
    const nuevo = renglonNuevo(1, chronos);
    nuevo.cantidad = '2';
    nuevo.precioNuevo = '24000.00';
    expect(resumenEfectos([nuevo]).join('|')).toContain('si es la compra más reciente de Chronos Crema');
  });

  it('errores del servidor con índice fuera de rango o sin índice no rompen ni se pierden', () => {
    const rs = renglonesDe(detalle);
    expect(mapearErroresServidor(rs, { 'items[9].cantidad': 'x', 'quitados[zzz]': 'y', fecha: 'z' }))
      .toEqual({ 'items[9].cantidad': 'x', 'quitados[zzz]': 'y', fecha: 'z' });
  });

  it('con dos nuevos y un quitado en el medio, el índice del servidor cae en el renglón correcto', () => {
    const rs = renglonesDe(detalle);
    rs[0].quitado = true;
    const n1 = renglonNuevo(1, chronos);
    const n2 = renglonNuevo(2, { ...chronos, id: 'p4', nombre: 'Otro' });
    // pantalla: [i1 (quitado), i2, n1, n2]; enviado: [i2, n1, n2]
    expect(mapearErroresServidor([...rs, n1, n2], { 'items[2].cantidad': 'sobre n2' })).toEqual({ 'items[3].cantidad': 'sobre n2' });
  });

  it('quitar un renglón con consumo devuelve el motivo y uno sin consumo y con otro activo no', () => {
    const rs = renglonesDe(detalle);
    rs[1].consumido = 1;
    expect(motivoNoQuitar(rs[1], rs)).toContain('Podés bajar la cantidad hasta 1');
    expect(motivoNoQuitar(rs[0], rs)).toBeNull();
  });
});

describe('QA US-001c: historial sin "undefined"', () => {
  const sucio = (s: string) => /undefined|null|NaN/.test(s);

  it('QUITADO solo con cantidad y costo (sin precioVentaNuevo ni vencimiento)', () => {
    const d = detalleRenglonHistorial({
      accion: 'QUITADO', productoId: 'p2', producto: 'Ekos Jabón',
      cambios: [{ campo: 'cantidad', antes: 10, despues: null }, { campo: 'costoUnitario', antes: '2100.00', despues: null }],
    });
    expect(d.cantidad).toBe('×10');
    expect(d.resto).toHaveLength(1);
    expect(sucio(`${d.cantidad} ${d.resto.join(' ')}`)).toBe(false);
  });

  it('AGREGADO sin precio ni vencimiento, y con vencimiento null explícito', () => {
    const d = detalleRenglonHistorial({
      accion: 'AGREGADO', productoId: 'p3', producto: 'Chronos',
      cambios: [{ campo: 'cantidad', antes: null, despues: 2 }, { campo: 'costoUnitario', antes: null, despues: '15000.00' },
        { campo: 'vencimiento', antes: null, despues: null }],
    });
    expect(d.cantidad).toBe('×2');
    expect(d.resto).toEqual([d.resto[0]]);
    expect(sucio(d.resto.join(' '))).toBe(false);
  });

  it('un renglón sin cambios de cantidad no muestra "×null"', () => {
    const d = detalleRenglonHistorial({ accion: 'QUITADO', productoId: 'p2', producto: 'Ekos', cambios: [] });
    expect(d.cantidad).toBeNull();
    expect(d.resto).toEqual([]);
  });
});
