import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { LucideAngularModule, Package } from 'lucide-angular';
import { CatalogoApi } from '../../core/api/catalogo.api';

/**
 * Foto de un producto (US-004), o un ícono si no tiene o si no carga (por ejemplo, sin conexión).
 * Decorativa: el nombre del producto siempre va al lado.
 */
@Component({
  selector: 'app-imagen-producto',
  imports: [NgOptimizedImage, LucideAngularModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'inline-flex shrink-0 overflow-hidden rounded-lg bg-ink-100 dark:bg-ink-800', '[style.width.px]': 'tamano()', '[style.height.px]': 'tamano()' },
  template: `
    @if (vistaPrevia(); as vp) {
      <!-- data: URL: NgOptimizedImage no la acepta -->
      <img [src]="vp" [width]="tamano()" [height]="tamano()" alt="" class="size-full bg-white object-cover" />
    } @else if (url(); as u) {
      <img [ngSrc]="u" [width]="tamano()" [height]="tamano()" alt="" class="size-full bg-white object-cover" loading="lazy" (error)="fallida.set(u)" />
    } @else {
      <lucide-icon [img]="icono" [size]="tamano() / 2" class="m-auto text-muted" aria-hidden="true" />
    }
  `,
})
export class ImagenProductoComponent {
  private readonly api = inject(CatalogoApi);
  readonly productoId = input.required<string>();
  /** 0 = sin foto. */
  readonly version = input(0);
  readonly tamano = input(40);
  /** Vista previa local (data: URL) que reemplaza a la guardada mientras se edita. */
  readonly vistaPrevia = input<string | null>(null);

  protected readonly icono = Package;
  protected readonly fallida = signal<string | null>(null);
  protected readonly url = computed(() => {
    const u = this.version() > 0 ? this.api.urlImagen(this.productoId(), this.version()) : null;
    return u && u !== this.fallida() ? u : null;
  });
}
