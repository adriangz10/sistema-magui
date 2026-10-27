import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

export interface FilaBarra {
  etiqueta: string;
  /** Valor para el largo de la barra. */
  valor: number;
  /** Texto del valor, al final de la barra. */
  texto: string;
  detalle?: string;
}

/**
 * Barras horizontales de una sola serie para rankings y repartos (más vendidos, formas de pago).
 * Cada fila lleva su nombre y su valor en texto, así la información no depende del color.
 */
@Component({
  selector: 'app-barras-horizontales',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (filas().length) {
      <ol class="space-y-3" [attr.aria-label]="titulo()">
        @for (f of filas(); track f.etiqueta; let i = $index) {
          <li>
            <div class="mb-1 flex items-baseline justify-between gap-3 text-sm">
              <span class="min-w-0 truncate" [title]="f.etiqueta">
                @if (numerada()) { <span class="mr-1 text-muted tabular-nums">{{ i + 1 }}.</span> }{{ f.etiqueta }}
              </span>
              <span class="shrink-0 font-medium tabular-nums">{{ f.texto }}</span>
            </div>
            <div class="h-2 rounded-full bg-ink-100 dark:bg-ink-800" aria-hidden="true">
              <div class="h-2 rounded-full bg-brand-600 dark:bg-brand-400" [style.width.%]="porcentaje(f.valor)"></div>
            </div>
            @if (f.detalle) { <p class="mt-0.5 text-xs text-muted">{{ f.detalle }}</p> }
          </li>
        }
      </ol>
    } @else {
      <p class="py-6 text-center text-sm text-muted">{{ vacio() }}</p>
    }
  `,
})
export class BarrasHorizontalesComponent {
  readonly filas = input.required<FilaBarra[]>();
  readonly titulo = input.required<string>();
  readonly numerada = input(false);
  readonly vacio = input('Sin datos en este período');

  private readonly maximo = computed(() => Math.max(0, ...this.filas().map((f) => f.valor)));

  protected porcentaje(v: number): number {
    const m = this.maximo();
    return m > 0 ? Math.max(1, (v / m) * 100) : 0;
  }
}
