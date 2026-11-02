import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { NegocioService } from '../../core/negocio/negocio.service';

const TAMANOS = { chico: 32, mediano: 36, grande: 48 } as const;

/** Logo del negocio, o su inicial si no tiene (o si la imagen no carga sin conexión). Decorativo: el nombre va al lado. */
@Component({
  selector: 'app-logo-negocio',
  imports: [NgOptimizedImage],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'inline-flex shrink-0' },
  template: `
    @if (imagen(); as url) {
      <img [ngSrc]="url" [width]="px()" [height]="px()" alt="" class="rounded-lg object-contain" (error)="fallida.set(url)" />
    } @else {
      <span aria-hidden="true" [style.width.px]="px()" [style.height.px]="px()"
            class="flex items-center justify-center rounded-lg bg-ink-900 font-semibold text-white dark:bg-ink-100 dark:text-ink-900"
            [class.text-xl]="tamano() === 'grande'" [class.text-sm]="tamano() !== 'grande'">{{ negocio.inicial() }}</span>
    }
  `,
})
export class LogoNegocioComponent {
  protected readonly negocio = inject(NegocioService);
  readonly tamano = input<keyof typeof TAMANOS>('mediano');
  protected readonly px = computed(() => TAMANOS[this.tamano()]);
  /** URL que no cargó (ej. sin conexión): se muestra la inicial hasta que cambie el logo. */
  protected readonly fallida = signal<string | null>(null);
  protected readonly imagen = computed(() => {
    const url = this.negocio.logo();
    return url && url !== this.fallida() ? url : null;
  });
}
