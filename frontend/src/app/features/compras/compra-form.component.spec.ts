import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { CompraDetalle, ComprasApi } from '../../core/api/compras.api';
import { ToastService } from '../../core/ui/toast.service';
import { CompraFormComponent } from './compra-form.component';

const compra = {
  id: 'c1', fecha: '2026-10-20T15:00:00.000Z', empresaId: 'e1', empresa: 'Natura', total: '100.00', productos: 1, unidades: 1,
  notas: null, version: 1, editadaEn: null, items: [], ediciones: [],
} as CompraDetalle;

function errorApi(status: number, code: string, fields?: Record<string, string>) {
  return throwError(() => new HttpErrorResponse({ status, error: { error: { code, message: 'x', fields } } }));
}

async function montar(editar: ReturnType<typeof vi.fn>) {
  TestBed.configureTestingModule({ providers: [{ provide: ComprasApi, useValue: { editar } }] });
  const fixture = TestBed.createComponent(CompraFormComponent);
  fixture.componentRef.setInput('compra', compra);
  fixture.componentRef.setInput('empresas', [{ id: 'e1', nombre: 'Natura', activo: true }]);
  fixture.detectChanges();
  await fixture.whenStable();
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  const notas = el.querySelector<HTMLInputElement>('#c-notas')!;
  notas.value = 'Pedido ciclo 15';
  notas.dispatchEvent(new Event('input'));
  fixture.detectChanges();
  const guardar = () => [...el.querySelectorAll('button')].find((b) => b.textContent?.includes('Guardar cambios'))!;
  return { fixture, el, guardar };
}

describe('CompraFormComponent en modo edición', () => {
  it('409 version_desactualizada muestra el aviso y deshabilita Guardar', async () => {
    const editar = vi.fn(() => errorApi(409, 'version_desactualizada'));
    const { fixture, el, guardar } = await montar(editar);
    guardar().click();
    await fixture.whenStable();
    fixture.detectChanges();
    expect(editar).toHaveBeenCalledOnce();
    expect(el.textContent).toContain('Otra persona modificó esta compra');
    expect(guardar().disabled).toBe(true);
  });

  it('422 con fields pinta errores por campo', async () => {
    const editar = vi.fn(() => errorApi(422, 'datos_invalidos', { notas: 'Máximo 500 caracteres' }));
    const { fixture, el, guardar } = await montar(editar);
    guardar().click();
    await fixture.whenStable();
    fixture.detectChanges();
    expect(el.textContent).toContain('Máximo 500 caracteres');
  });

  it('404 empresa_no_encontrada marca el campo empresa', async () => {
    const editar = vi.fn(() => errorApi(404, 'empresa_no_encontrada'));
    const { fixture, el, guardar } = await montar(editar);
    guardar().click();
    await fixture.whenStable();
    fixture.detectChanges();
    expect(el.textContent).toContain('La empresa elegida ya no existe');
  });

  it('404 compra_no_encontrada avisa y cierra', async () => {
    const editar = vi.fn(() => errorApi(404, 'compra_no_encontrada'));
    const { fixture, guardar } = await montar(editar);
    const error = vi.spyOn(TestBed.inject(ToastService), 'error');
    const cancelado = vi.fn();
    fixture.componentInstance.cancelado.subscribe(cancelado);
    guardar().click();
    await fixture.whenStable();
    expect(error).toHaveBeenCalledWith(expect.objectContaining({ message: 'La compra ya no existe' }));
    expect(cancelado).toHaveBeenCalled();
  });

  it('sin cambios no llama a la API y emite sinCambios', async () => {
    const editar = vi.fn();
    const { fixture, el, guardar } = await montar(editar);
    const notas = el.querySelector<HTMLInputElement>('#c-notas')!;
    notas.value = '';
    notas.dispatchEvent(new Event('input'));
    const sin = vi.fn();
    fixture.componentInstance.sinCambios.subscribe(sin);
    guardar().click();
    expect(editar).not.toHaveBeenCalled();
    expect(sin).toHaveBeenCalled();
  });
});

const compraItems = {
  ...compra, total: '163080.00', unidades: 14, productos: 2,
  items: [
    { id: 'i1', productoId: 'p1', sku: 'P1', producto: 'Kaiak Clásico', cantidad: 4, costoUnitario: '35520.00', subtotal: '142080.00',
      precioVentaNuevo: '62000.00', vencimiento: '2027-09-30', consumido: 3 },
    { id: 'i2', productoId: 'p2', sku: 'P2', producto: 'Ekos Jabón', cantidad: 10, costoUnitario: '2100.00', subtotal: '21000.00',
      precioVentaNuevo: null, vencimiento: null, consumido: 0 },
  ],
} as CompraDetalle;

async function montarItems(editar: ReturnType<typeof vi.fn>, obtener: ReturnType<typeof vi.fn> = vi.fn()) {
  TestBed.configureTestingModule({ providers: [{ provide: ComprasApi, useValue: { editar, obtener } }] });
  const fixture = TestBed.createComponent(CompraFormComponent);
  fixture.componentRef.setInput('compra', compraItems);
  fixture.componentRef.setInput('empresas', [{ id: 'e1', nombre: 'Natura', activo: true }]);
  fixture.detectChanges();
  await fixture.whenStable();
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  const poner = (id: string, valor: string) => {
    const i = el.querySelector<HTMLInputElement>(id)!;
    i.value = valor;
    i.dispatchEvent(new Event('input'));
    fixture.detectChanges();
  };
  const guardar = () => [...el.querySelectorAll('button')].find((b) => b.textContent?.includes('Guardar cambios'))!;
  return { fixture, el, poner, guardar };
}

describe('CompraFormComponent edición de renglones', () => {
  it('muestra lo consumido y bloquea bajar de ese número sin llamar a la API', async () => {
    const editar = vi.fn();
    const { fixture, el, poner, guardar } = await montarItems(editar);
    expect(el.textContent).toContain('ya salieron 3 de 4');
    poner('#e-cant-i1', '2');
    guardar().click();
    await fixture.whenStable();
    fixture.detectChanges();
    expect(editar).not.toHaveBeenCalled();
    expect(el.textContent).toContain('Ya salieron 3 unidades de este lote, no podés dejar menos de 3');
  });

  it('envía items completos con id y muestra los efectos', async () => {
    const editar = vi.fn(() => errorApi(500, 'x'));
    const { fixture, el, poner, guardar } = await montarItems(editar);
    poner('#e-cant-i2', '12');
    expect(el.textContent).toContain('El stock de Ekos Jabón sube 2');
    guardar().click();
    await fixture.whenStable();
    const llamada = editar.mock.calls[0] as unknown as [string, { items: { id: string; cantidad: number }[] }];
    expect(llamada[1].items.map((i) => [i.id, i.cantidad])).toEqual([['i1', 4], ['i2', 12]]);
  });

  it('409 compra_consumida pinta el mensaje en el renglón correcto', async () => {
    const msg = 'Kaiak Clásico: de las 4 unidades ya salieron 3. Solo podés bajar hasta 3 unidades.';
    const editar = vi.fn(() => errorApi(409, 'compra_consumida', { 'items[0].cantidad': msg }));
    const { fixture, el, poner, guardar } = await montarItems(editar);
    poner('#e-cant-i2', '9');
    guardar().click();
    await fixture.whenStable();
    fixture.detectChanges();
    const filas = el.querySelectorAll('ul[aria-label="Renglones de la compra"] > li');
    expect(filas[0].textContent).toContain(msg);
    expect(filas[1].textContent).not.toContain(msg);
  });

  it('tras 409 compra_consumida recarga el consumo y actualiza el mínimo sin pisar lo tipeado', async () => {
    const editar = vi.fn(() => errorApi(409, 'compra_consumida', { 'items[1].cantidad': 'Ekos Jabón: ya salieron 6' }));
    const fresca = { ...compraItems, items: compraItems.items.map((i) => (i.id === 'i2' ? { ...i, consumido: 6 } : i)) };
    const obtener = vi.fn(() => of(fresca));
    const { fixture, el, poner, guardar } = await montarItems(editar, obtener);
    poner('#e-cant-i2', '5');
    guardar().click();
    await fixture.whenStable();
    fixture.detectChanges();
    expect(obtener).toHaveBeenCalledWith('c1');
    expect(el.querySelector<HTMLInputElement>('#e-cant-i2')!.min).toBe('6');
    expect(el.querySelector<HTMLInputElement>('#e-cant-i2')!.value).toBe('5');
    expect(el.textContent).toContain('Ekos Jabón: ya salieron 6');
  });

  it('409 operacion_concurrente avisa que se reintente', async () => {
    const editar = vi.fn(() => errorApi(409, 'operacion_concurrente'));
    const { fixture, el, poner, guardar } = await montarItems(editar);
    poner('#e-cant-i2', '11');
    guardar().click();
    await fixture.whenStable();
    fixture.detectChanges();
    expect(el.textContent).toContain('Otra operación se cruzó, intentá de nuevo');
    expect(guardar().disabled).toBe(false);
  });

  it('si solo cambia la cabecera no manda items', async () => {
    const editar = vi.fn(() => errorApi(500, 'x'));
    const { fixture, el, guardar } = await montarItems(editar);
    const notas = el.querySelector<HTMLInputElement>('#c-notas')!;
    notas.value = 'Hola';
    notas.dispatchEvent(new Event('input'));
    guardar().click();
    await fixture.whenStable();
    expect('items' in (editar.mock.calls[0] as unknown as [string, object])[1]).toBe(false);
  });
});

describe('CompraFormComponent agregar y quitar renglones', () => {
  const boton = (el: HTMLElement, etiqueta: string) => el.querySelector<HTMLButtonElement>(`button[aria-label="${etiqueta}"]`)!;

  it('Quitar está deshabilitado con consumo y explica por qué', async () => {
    const { el } = await montarItems(vi.fn());
    expect(boton(el, 'Quitar Kaiak Clásico').getAttribute('aria-disabled')).toBe('true');
    const desc = boton(el, 'Quitar Kaiak Clásico').getAttribute('aria-describedby')!;
    expect(el.querySelector('#' + desc)!.textContent).toContain('Podés bajar la cantidad hasta 3');
    expect(el.textContent).toContain('ya salieron 3 de 4');
    expect(boton(el, 'Quitar Ekos Jabón').getAttribute('aria-disabled')).toBeNull();
  });

  it('quitar marca "Se quitará", resume el efecto, no manda el renglón y deshacer lo revierte', async () => {
    const editar = vi.fn(() => errorApi(500, 'x'));
    const { fixture, el, guardar } = await montarItems(editar);
    boton(el, 'Quitar Ekos Jabón').click();
    fixture.detectChanges();
    expect(el.textContent).toContain('Se quitará');
    expect(el.textContent).toContain('Se quita Ekos Jabón (−10 del stock)');
    boton(el, 'Deshacer quitar Ekos Jabón').click();
    fixture.detectChanges();
    expect(el.textContent).not.toContain('Se quitará');
    expect(el.textContent).not.toContain('Se quita Ekos');
    boton(el, 'Quitar Ekos Jabón').click();
    fixture.detectChanges();
    guardar().click();
    await fixture.whenStable();
    const cuerpo = (editar.mock.calls[0] as unknown as [string, { items: { id?: string }[] }])[1];
    expect(cuerpo.items.map((i) => i.id)).toEqual(['i1']);
  });

  it('no permite quitar el último renglón activo', async () => {
    const un = { ...compraItems, items: [{ ...compraItems.items[1] }] } as CompraDetalle;
    TestBed.configureTestingModule({ providers: [{ provide: ComprasApi, useValue: { editar: vi.fn(), obtener: vi.fn() } }] });
    const fixture = TestBed.createComponent(CompraFormComponent);
    fixture.componentRef.setInput('compra', un);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    const el = fixture.nativeElement as HTMLElement;
    expect(boton(el, 'Quitar Ekos Jabón').getAttribute('aria-disabled')).toBe('true');
    expect(el.textContent).toContain('La compra necesita al menos un producto');
  });

  it('409 con items[i] y quitados[id] se pinta en el renglón correcto aunque haya quitados', async () => {
    const editar = vi.fn(() => errorApi(409, 'compra_consumida', { 'items[0].cantidad': 'MSG-ITEM', 'quitados[i2]': 'MSG-QUITADO' }));
    const obtener = vi.fn(() => of({ ...compraItems, items: compraItems.items.map((i) => (i.id === 'i2' ? { ...i, consumido: 1 } : i)) }));
    const { fixture, el, guardar } = await montarItems(editar, obtener);
    boton(el, 'Quitar Ekos Jabón').click();
    fixture.detectChanges();
    guardar().click();
    await fixture.whenStable();
    fixture.detectChanges();
    const filas = el.querySelectorAll('ul[aria-label="Renglones de la compra"] > li');
    expect(filas[0].textContent).toContain('MSG-ITEM');
    expect(filas[1].textContent).toContain('MSG-QUITADO');
    expect(filas[1].textContent).not.toContain('Se quitará');
  });

  it('un aria-disabled sigue enfocable y no quita; quitar/deshacer conserva el foco en el mismo botón', async () => {
    const { fixture, el } = await montarItems(vi.fn());
    const bloqueado = boton(el, 'Quitar Kaiak Clásico');
    bloqueado.focus();
    bloqueado.click();
    fixture.detectChanges();
    expect(document.activeElement).toBe(bloqueado);
    expect(el.textContent).not.toContain('Se quitará');
    const b = boton(el, 'Quitar Ekos Jabón');
    b.focus();
    b.click();
    fixture.detectChanges();
    const mismo = boton(el, 'Deshacer quitar Ekos Jabón');
    expect(mismo).toBe(b);
    expect(document.activeElement).toBe(b);
    mismo.click();
    fixture.detectChanges();
    expect(boton(el, 'Quitar Ekos Jabón')).toBe(b);
    expect(document.activeElement).toBe(b);
  });

  it('muestra errores items[i].id e items[i].productoId del servidor en el renglón', async () => {
    const editar = vi.fn(() => errorApi(422, 'datos_invalidos', { 'items[0].id': 'MSG-ID', 'items[1].productoId': 'MSG-PROD' }));
    const { fixture, el, poner, guardar } = await montarItems(editar);
    poner('#e-cant-i2', '11');
    guardar().click();
    await fixture.whenStable();
    fixture.detectChanges();
    const filas = el.querySelectorAll('ul[aria-label="Renglones de la compra"] > li');
    expect(filas[0].textContent).toContain('MSG-ID');
    expect(filas[1].textContent).toContain('MSG-PROD');
  });
});