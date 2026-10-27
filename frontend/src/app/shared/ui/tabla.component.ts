import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';
import { Pagina } from '../../core/api/modelos';
import { PaginadorComponent } from './paginador.component';

/**
 * Contenedor de tablas paginadas (máximo 10 filas). La tabla se proyecta adentro:
 *
 *   <app-tabla [pagina]="productos()" [cargando]="cargando()" (paginaCambio)="ir($event)">
 *     <table class="tabla"> ... </table>
 *   </app-tabla>
 */
@Component({
  selector: 'app-tabla',
  imports: [PaginadorComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="card overflow-hidden">
      <div class="relative overflow-x-auto" [attr.aria-busy]="cargando()">
        @if (cargando() && vacia()) {
          <div class="space-y-3 p-4" aria-label="Cargando">
            @for (_ of filasEsqueleto; track $index) {
              <div class="h-10 animate-pulse rounded-lg bg-ink-100 dark:bg-ink-800"></div>
            }
          </div>
        } @else if (vacia()) {
          <div class="px-6 py-14 text-center">
            <p class="font-medium text-ink-700 dark:text-ink-200">{{ mensajeVacio() }}</p>
            @if (ayudaVacio()) {
              <p class="mt-1 text-sm text-muted">{{ ayudaVacio() }}</p>
            }
          </div>
        } @else {
          <div [class.opacity-60]="cargando()" class="transition-opacity">
            <ng-content />
          </div>
        }
      </div>
      @if (pagina(); as p) {
        <div class="border-t border-ink-100 dark:border-ink-800">
          <app-paginador [pagina]="p.page" [totalPaginas]="p.totalPages" [total]="p.total" [tamano]="p.size"
                         (cambio)="paginaCambio.emit($event)" />
        </div>
      }
    </div>
  `,
})
export class TablaComponent {
  readonly pagina = input<Pagina<unknown> | null>(null);
  readonly cargando = input(false);
  readonly mensajeVacio = input('No hay datos para mostrar');
  readonly ayudaVacio = input('');
  readonly paginaCambio = output<number>();

  protected readonly filasEsqueleto = Array.from({ length: 5 });
  protected readonly vacia = computed(() => (this.pagina()?.items.length ?? 0) === 0);
}
