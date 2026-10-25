import { ChangeDetectionStrategy, Component, input } from '@angular/core';

/**
 * Etiqueta + control + ayuda/error, con los atributos de accesibilidad conectados.
 * El control proyectado debe tener [id]="para" y, si hay error, [attr.aria-invalid]="true"
 * y [attr.aria-describedby]="para + '-error'".
 *
 *   <app-campo etiqueta="Nombre" para="nombre" [error]="errores()['nombre']">
 *     <input id="nombre" class="input" ... />
 *   </app-campo>
 */
@Component({
  selector: 'app-campo',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <label [for]="para()" class="etiqueta">
      {{ etiqueta() }}
      @if (opcional()) {
        <span class="font-normal text-ink-400">(opcional)</span>
      }
    </label>
    <ng-content />
    @if (error()) {
      <p [id]="para() + '-error'" class="mt-1 text-sm text-danger dark:text-red-400" role="alert">{{ error() }}</p>
    } @else if (ayuda()) {
      <p [id]="para() + '-ayuda'" class="mt-1 text-xs text-ink-500">{{ ayuda() }}</p>
    }
  `,
})
export class CampoComponent {
  readonly etiqueta = input.required<string>();
  readonly para = input.required<string>();
  readonly error = input<string | null | undefined>(null);
  readonly ayuda = input('');
  readonly opcional = input(false);
}
