import { Component, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { ToastService } from '../../core/ui/toast.service';
import { InputDineroComponent } from './input-dinero.component';
import { PaginadorComponent, calcularBotones } from './paginador.component';

describe('Paginador', () => {
  it('resume páginas lejanas con "…"', () => {
    expect(calcularBotones(1, 5)).toEqual([1, 2, 3, 4, 5]);
    expect(calcularBotones(1, 36)).toEqual([1, 2, '…', 36]);
    expect(calcularBotones(18, 36)).toEqual([1, '…', 17, 18, 19, '…', 36]);
    expect(calcularBotones(36, 36)).toEqual([1, '…', 35, 36]);
  });

  it('muestra el rango y emite la página elegida', async () => {
    @Component({
      imports: [PaginadorComponent],
      template: `<app-paginador [pagina]="pagina()" [totalPaginas]="36" [total]="353" (cambio)="pagina.set($event)" />`,
    })
    class Anfitrion {
      pagina = signal(2);
    }
    const f = TestBed.createComponent(Anfitrion);
    await f.whenStable();
    const el = f.nativeElement as HTMLElement;
    expect(el.textContent).toContain('Mostrando 11–20 de 353');
    (el.querySelector('[aria-label="Página siguiente"]') as HTMLButtonElement).click();
    await f.whenStable();
    expect(f.componentInstance.pagina()).toBe(3);
    expect(el.querySelector('[aria-current="page"]')?.textContent?.trim()).toBe('3');
  });
});

describe('InputDinero', () => {
  @Component({
    imports: [InputDineroComponent, ReactiveFormsModule],
    template: `<app-input-dinero idCampo="precio" [formControl]="control" />`,
  })
  class Anfitrion {
    control = new FormControl<string | null>('59200.5');
  }

  it('muestra el valor formateado y entrega el formato de la API', async () => {
    const f = TestBed.createComponent(Anfitrion);
    await f.whenStable();
    const input = (f.nativeElement as HTMLElement).querySelector('input')!;
    expect(input.value).toBe('59.200,50');
    // el id va solo en el campo interno (si se repite, la etiqueta no queda conectada)
    expect((f.nativeElement as HTMLElement).querySelectorAll('#precio')).toHaveLength(1);
    expect(input.id).toBe('precio');

    input.value = '1.500,5';
    input.dispatchEvent(new Event('input'));
    expect(f.componentInstance.control.value).toBe('1500.50');

    input.value = 'abc';
    input.dispatchEvent(new Event('input'));
    expect(f.componentInstance.control.value).toBeNull();
  });
});

describe('ToastService', () => {
  it('muestra avisos y los cierra solos', () => {
    vi.useFakeTimers();
    const t = TestBed.inject(ToastService);
    t.exito('Venta registrada');
    t.error({ message: 'raro' });
    expect(t.toasts().map((x) => x.tipo)).toEqual(['exito', 'error']);
    vi.advanceTimersByTime(4000);
    expect(t.toasts().map((x) => x.tipo)).toEqual(['error']);
    vi.advanceTimersByTime(3000);
    expect(t.toasts()).toEqual([]);
    vi.useRealTimers();
  });
});
