import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

/**
 * Tarjeta con un número destacado, su etiqueta y, opcionalmente, la variación contra el
 * período anterior (▲ verde si sube, ▼ rojo si baja; con texto, nunca solo color).
 */
@Component({
  selector: 'app-tarjeta-dato',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'card block p-5' },
  template: `
    <p class="text-sm text-muted">{{ etiqueta() }}</p>
    <p class="mt-1 font-semibold tracking-tight" [class]="destacado() ? 'text-4xl sm:text-5xl' : 'text-2xl'">{{ valor() }}</p>
    @if (variacion() !== null && variacion() !== undefined) {
      <p class="mt-1 text-sm" [class]="sube() ? 'text-ok' : 'text-danger'">
        <span aria-hidden="true">{{ sube() ? '▲' : '▼' }}</span>
        {{ variacionTexto() }} {{ comparacion() }}
      </p>
    } @else if (detalle()) {
      <p class="mt-1 text-sm text-muted">{{ detalle() }}</p>
    }
  `,
})
export class TarjetaDatoComponent {
  readonly etiqueta = input.required<string>();
  readonly valor = input.required<string>();
  readonly detalle = input('');
  /** Variación en % contra el período anterior (null = sin base para comparar). */
  readonly variacion = input<number | null | undefined>(undefined);
  readonly comparacion = input('vs. período anterior');
  readonly destacado = input(false);

  protected readonly sube = computed(() => (this.variacion() ?? 0) >= 0);
  protected readonly variacionTexto = computed(() => {
    const v = this.variacion() ?? 0;
    return `${v >= 0 ? '+' : ''}${new Intl.NumberFormat('es-AR', { maximumFractionDigits: 1 }).format(v)} %`;
  });
}
