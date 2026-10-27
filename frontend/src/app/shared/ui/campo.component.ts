import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { CircleAlert, LucideAngularModule } from 'lucide-angular';

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
  imports: [LucideAngularModule],
  host: { class: 'block' },
  template: `
    <label [for]="para()" class="etiqueta">
      {{ etiqueta() }}
      @if (opcional()) {
        <span class="font-normal text-muted">(opcional)</span>
      }
    </label>
    <ng-content />
    @if (error()) {
      <p [id]="para() + '-error'" class="mt-1 flex items-start gap-1.5 text-sm text-danger" role="alert">
        <lucide-icon [img]="iconoError" [size]="16" class="mt-0.5 shrink-0" aria-hidden="true" />
        <span>{{ error() }}</span>
      </p>
    } @else if (ayuda()) {
      <p [id]="para() + '-ayuda'" class="mt-1 text-xs text-muted">{{ ayuda() }}</p>
    }
  `,
})
export class CampoComponent {
  protected readonly iconoError = CircleAlert;
  readonly etiqueta = input.required<string>();
  readonly para = input.required<string>();
  readonly error = input<string | null | undefined>(null);
  readonly ayuda = input('');
  readonly opcional = input(false);
}
