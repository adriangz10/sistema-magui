import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';
import { ChevronLeft, ChevronRight, LucideAngularModule } from 'lucide-angular';

type Boton = number | '…';

/** Paginador: "Mostrando 11–20 de 353" con anterior, siguiente y números de página. */
@Component({
  selector: 'app-paginador',
  imports: [LucideAngularModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (total() > 0) {
      <nav class="flex flex-col items-center justify-between gap-3 px-4 py-3 text-sm sm:flex-row" aria-label="Paginación">
        <p class="text-ink-500" aria-live="polite">
          Mostrando <span class="font-medium text-ink-800 dark:text-ink-200">{{ desde() }}–{{ hasta() }}</span>
          de <span class="font-medium text-ink-800 dark:text-ink-200">{{ total() }}</span>
        </p>
        @if (totalPaginas() > 1) {
          <div class="flex items-center gap-1">
            <button type="button" class="btn-fantasma btn-icono" [disabled]="pagina() <= 1"
                    (click)="ir(pagina() - 1)" aria-label="Página anterior">
              <lucide-icon [img]="iconoAnterior" [size]="18" aria-hidden="true" />
            </button>
            @for (b of botones(); track $index) {
              @if (b === '…') {
                <span class="px-2 text-ink-400" aria-hidden="true">…</span>
              } @else {
                <button type="button" class="btn btn-icono"
                        [class]="b === pagina() ? 'bg-brand-600 text-white' : 'text-ink-700 hover:bg-ink-100 dark:text-ink-200 dark:hover:bg-ink-800'"
                        [attr.aria-current]="b === pagina() ? 'page' : null" [attr.aria-label]="'Página ' + b"
                        (click)="ir(b)">{{ b }}</button>
              }
            }
            <button type="button" class="btn-fantasma btn-icono" [disabled]="pagina() >= totalPaginas()"
                    (click)="ir(pagina() + 1)" aria-label="Página siguiente">
              <lucide-icon [img]="iconoSiguiente" [size]="18" aria-hidden="true" />
            </button>
          </div>
        }
      </nav>
    }
  `,
})
export class PaginadorComponent {
  readonly pagina = input.required<number>();
  readonly totalPaginas = input.required<number>();
  readonly total = input.required<number>();
  readonly tamano = input(10);
  readonly cambio = output<number>();

  protected readonly iconoAnterior = ChevronLeft;
  protected readonly iconoSiguiente = ChevronRight;

  protected readonly desde = computed(() => (this.pagina() - 1) * this.tamano() + 1);
  protected readonly hasta = computed(() => Math.min(this.pagina() * this.tamano(), this.total()));

  /** Primera, última, la actual y sus vecinas; el resto se resume con "…". */
  protected readonly botones = computed<Boton[]>(() => calcularBotones(this.pagina(), this.totalPaginas()));

  protected ir(p: number): void {
    if (p >= 1 && p <= this.totalPaginas() && p !== this.pagina()) {
      this.cambio.emit(p);
    }
  }
}

export function calcularBotones(actual: number, total: number): Boton[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }
  const visibles = new Set([1, total, actual - 1, actual, actual + 1].filter((p) => p >= 1 && p <= total));
  const ordenadas = [...visibles].sort((a, b) => a - b);
  const out: Boton[] = [];
  ordenadas.forEach((p, i) => {
    if (i > 0 && p - ordenadas[i - 1] > 1) {
      out.push('…');
    }
    out.push(p);
  });
  return out;
}
