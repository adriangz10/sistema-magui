import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { throwError } from 'rxjs';
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
