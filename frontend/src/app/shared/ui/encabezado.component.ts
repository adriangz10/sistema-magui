import { ChangeDetectionStrategy, Component, input } from '@angular/core';

/** Título de pantalla con descripción opcional y acciones a la derecha. */
@Component({
  selector: 'app-encabezado',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between' },
  template: `
    <div>
      <h1 class="titulo-pagina">{{ titulo() }}</h1>
      @if (descripcion()) {
        <p class="mt-1 text-sm text-muted">{{ descripcion() }}</p>
      }
    </div>
    <div class="flex flex-wrap gap-2">
      <ng-content />
    </div>
  `,
})
export class EncabezadoComponent {
  readonly titulo = input.required<string>();
  readonly descripcion = input('');
}
