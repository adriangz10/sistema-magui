import { ChangeDetectionStrategy, Component, forwardRef, input, signal } from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';
import { formatearNumero, interpretarDinero } from '../formato/dinero';

/**
 * Campo de monto en pesos. Muestra "59.200,50" y entrega al formulario "59200.50"
 * (el formato de la API), o null si está vacío o es inválido.
 *
 *   <app-input-dinero idCampo="precio" formControlName="precioVenta" />
 *
 * (Se llama idCampo y no id para no repetir el id en el componente y en el campo interno.)
 */
@Component({
  selector: 'app-input-dinero',
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [{ provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => InputDineroComponent), multi: true }],
  template: `
    <div class="relative">
      <span class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-sm text-muted" aria-hidden="true">$</span>
      <input type="text" inputmode="decimal" autocomplete="off" class="input pl-7 text-right tabular-nums"
             [id]="idCampo()" [placeholder]="placeholder()" [disabled]="deshabilitado()" [value]="texto()"
             [attr.aria-invalid]="invalido() || null" [attr.aria-describedby]="describedBy() || null"
             (input)="alEscribir($any($event.target).value)" (blur)="alSalir()" (focus)="$any($event.target).select()" />
    </div>
  `,
})
export class InputDineroComponent implements ControlValueAccessor {
  readonly idCampo = input<string>('');
  readonly placeholder = input('0,00');
  readonly invalido = input(false);
  readonly describedBy = input<string>('');

  protected readonly texto = signal('');
  protected readonly deshabilitado = signal(false);
  private valor: string | null = null;
  private alCambiar: (v: string | null) => void = () => {};
  private alTocar: () => void = () => {};

  writeValue(v: string | null): void {
    this.valor = v ?? null;
    this.texto.set(v ? formatearNumero(v) : '');
  }

  registerOnChange(fn: (v: string | null) => void): void {
    this.alCambiar = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.alTocar = fn;
  }

  setDisabledState(d: boolean): void {
    this.deshabilitado.set(d);
  }

  protected alEscribir(t: string): void {
    this.texto.set(t);
    this.valor = interpretarDinero(t);
    this.alCambiar(this.valor);
  }

  protected alSalir(): void {
    if (this.valor !== null) {
      this.texto.set(formatearNumero(this.valor));
    }
    this.alTocar();
  }
}
