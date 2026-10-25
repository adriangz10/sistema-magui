import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';

/** Pantalla provisoria para las secciones que se construyen en la Fase 8. */
@Component({
  selector: 'app-proximamente',
  imports: [EncabezadoComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-encabezado [titulo]="titulo()" />
    <div class="card px-6 py-16 text-center">
      <p class="font-medium text-ink-700 dark:text-ink-200">Esta sección se está construyendo</p>
      <p class="mt-1 text-sm text-ink-500">Va a estar disponible en la próxima etapa.</p>
    </div>
  `,
})
export class ProximamentePage {
  /** Viene de la propiedad data de la ruta. */
  readonly titulo = input('');
}
