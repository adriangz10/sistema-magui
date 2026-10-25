import { ChangeDetectionStrategy, Component, DestroyRef, ElementRef, afterNextRender, computed, inject, input, signal } from '@angular/core';
import { compacto, marcasEje } from './escala';

/** Un punto del gráfico. */
export interface PuntoColumna {
  /** Etiqueta del eje X (ej. "lun 20"). */
  etiqueta: string;
  /** Valor numérico (para dinero, en pesos). */
  valor: number;
  /** Texto completo para el tooltip y la tabla (ej. "$ 98.900,00"). */
  texto: string;
  /** Detalle secundario opcional (ej. "3 ventas"). */
  detalle?: string;
}

const ALTO_PLOT = 180;
const MARGEN = { arriba: 22, derecha: 8, abajo: 28, izquierda: 52 };

/**
 * Columnas de una sola serie, en SVG, con eje Y redondeado, cuadrícula fina,
 * tooltip al pasar el mouse o con el teclado, el máximo rotulado y vista de tabla.
 * Sigue la guía de visualización del proyecto (columnas ≤ 24px, punta redondeada de 4px,
 * separación de 2px, color de marca validado para claro y oscuro).
 */
@Component({
  selector: 'app-grafico-columnas',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <div class="relative" [style.height.px]="altoTotal">
      @if (ancho() > 0) {
        <svg [attr.width]="ancho()" [attr.height]="altoTotal" role="img" [attr.aria-label]="titulo() + '. Detalle en la tabla de abajo.'"
             class="overflow-visible">
          <!-- cuadrícula y eje Y -->
          @for (m of marcas(); track m) {
            <line [attr.x1]="margen.izquierda" [attr.x2]="ancho() - margen.derecha" [attr.y1]="y(m)" [attr.y2]="y(m)"
                  class="stroke-ink-200 dark:stroke-ink-800" stroke-width="1" />
            <text [attr.x]="margen.izquierda - 8" [attr.y]="y(m)" text-anchor="end" dominant-baseline="middle"
                  class="fill-ink-500 text-[11px] tabular-nums">{{ formatoEje(m) }}</text>
          }
          <!-- columnas -->
          @for (p of datos(); track p.etiqueta; let i = $index) {
            <g class="cursor-default outline-none" tabindex="0" [attr.aria-label]="p.etiqueta + ': ' + p.texto + (p.detalle ? ', ' + p.detalle : '')"
               (pointerenter)="activa.set(i)" (pointerleave)="activa.set(-1)" (focus)="activa.set(i)" (blur)="activa.set(-1)">
              <!-- zona de interacción más grande que la columna -->
              <rect [attr.x]="xBanda(i)" [attr.y]="margen.arriba" [attr.width]="banda()" [attr.height]="ALTO_PLOT" fill="transparent" />
              @if (p.valor > 0) {
                <path [attr.d]="columna(i, p.valor)" class="transition-opacity"
                      [class]="activa() >= 0 && activa() !== i ? 'fill-brand-600 opacity-40 dark:fill-brand-500' : 'fill-brand-600 dark:fill-brand-500'" />
              }
              @if (i === indiceMaximo() && p.valor > 0) {
                <text [attr.x]="centro(i)" [attr.y]="y(p.valor) - 6" text-anchor="middle" class="fill-ink-700 text-[11px] font-medium dark:fill-ink-300">{{ formatoEje(p.valor) }}</text>
              }
              @if (mostrarEtiqueta(i)) {
                <text [attr.x]="centro(i)" [attr.y]="margen.arriba + ALTO_PLOT + 18" text-anchor="middle" class="fill-ink-500 text-[11px]">{{ p.etiqueta }}</text>
              }
            </g>
          }
          <line [attr.x1]="margen.izquierda" [attr.x2]="ancho() - margen.derecha" [attr.y1]="y(0)" [attr.y2]="y(0)" class="stroke-ink-300 dark:stroke-ink-700" stroke-width="1" />
        </svg>
        @if (activa() >= 0) {
          @let p = datos()[activa()];
          <div class="pointer-events-none absolute z-10 -translate-x-1/2 -translate-y-full rounded-lg border border-ink-200 bg-white px-3 py-2 text-xs shadow-lg dark:border-ink-700 dark:bg-ink-900"
               [style.left.px]="centro(activa())" [style.top.px]="Math.max(y(p.valor) - 8, 0)" role="status">
            <p class="text-sm font-semibold whitespace-nowrap">{{ p.texto }}</p>
            <p class="whitespace-nowrap text-ink-500">{{ p.etiqueta }}{{ p.detalle ? ' · ' + p.detalle : '' }}</p>
          </div>
        }
      }
    </div>
    <details class="mt-2 text-sm">
      <summary class="cursor-pointer text-ink-500 hover:text-ink-700 dark:hover:text-ink-300">Ver como tabla</summary>
      <div class="mt-2 max-h-64 overflow-y-auto">
        <table class="tabla">
          <caption class="sr-only">{{ titulo() }}</caption>
          <thead><tr><th scope="col">{{ etiquetaX() }}</th><th scope="col" class="text-right">Valor</th><th scope="col">Detalle</th></tr></thead>
          <tbody>
            @for (p of datos(); track p.etiqueta) {
              <tr><td>{{ p.etiqueta }}</td><td class="text-right tabular-nums">{{ p.texto }}</td><td class="text-ink-500">{{ p.detalle ?? '' }}</td></tr>
            }
          </tbody>
        </table>
      </div>
    </details>
  `,
})
export class GraficoColumnasComponent {
  readonly datos = input.required<PuntoColumna[]>();
  readonly titulo = input.required<string>();
  readonly etiquetaX = input('Período');
  /** Prefijo del eje Y (ej. "$"). */
  readonly prefijo = input('');

  protected readonly ALTO_PLOT = ALTO_PLOT;
  protected readonly altoTotal = ALTO_PLOT + MARGEN.arriba + MARGEN.abajo;
  protected readonly margen = MARGEN;
  protected readonly Math = Math;
  protected readonly ancho = signal(0);
  protected readonly activa = signal(-1);

  protected readonly marcas = computed(() => marcasEje(Math.max(0, ...this.datos().map((d) => d.valor))));
  private readonly maximoEje = computed(() => this.marcas()[this.marcas().length - 1] || 1);
  protected readonly banda = computed(() => (this.ancho() - MARGEN.izquierda - MARGEN.derecha) / Math.max(1, this.datos().length));
  protected readonly anchoColumna = computed(() => Math.max(2, Math.min(24, this.banda() - 2)));
  protected readonly indiceMaximo = computed(() => {
    const d = this.datos();
    return d.reduce((iMax, p, i) => (p.valor > d[iMax].valor ? i : iMax), 0);
  });

  constructor() {
    const el = inject(ElementRef<HTMLElement>).nativeElement;
    const destroy = inject(DestroyRef);
    afterNextRender(() => {
      // Medida inicial directa: no depende de que el navegador dispare el ResizeObserver
      // (no lo hace si la ventana está oculta o minimizada).
      this.ancho.set(Math.floor(el.getBoundingClientRect().width));
      const obs = new ResizeObserver((entradas) => this.ancho.set(Math.floor(entradas[0].contentRect.width)));
      obs.observe(el);
      destroy.onDestroy(() => obs.disconnect());
    });
  }

  protected y(v: number): number {
    return MARGEN.arriba + ALTO_PLOT - (v / this.maximoEje()) * ALTO_PLOT;
  }

  protected xBanda(i: number): number {
    return MARGEN.izquierda + i * this.banda();
  }

  protected centro(i: number): number {
    return this.xBanda(i) + this.banda() / 2;
  }

  /** Columna con punta redondeada (4px) y base recta. */
  protected columna(i: number, v: number): string {
    const w = this.anchoColumna();
    const x = this.centro(i) - w / 2;
    const base = this.y(0);
    const tope = this.y(v);
    const r = Math.min(4, w / 2, base - tope);
    return `M${x},${base} L${x},${tope + r} Q${x},${tope} ${x + r},${tope} L${x + w - r},${tope} Q${x + w},${tope} ${x + w},${tope + r} L${x + w},${base} Z`;
  }

  /** Con muchas columnas, se rotula una de cada tantas para que no se encimen. */
  protected mostrarEtiqueta(i: number): boolean {
    const cada = Math.ceil(44 / Math.max(1, this.banda()));
    return i % cada === 0;
  }

  protected formatoEje(v: number): string {
    return compacto(v, this.prefijo());
  }
}
