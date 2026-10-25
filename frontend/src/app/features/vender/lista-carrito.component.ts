import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { LucideAngularModule, Minus, Plus, ShoppingCart, Trash2 } from 'lucide-angular';
import { aCentavos, DineroPipe, deCentavos } from '../../shared/formato/dinero';
import { CANTIDAD_MAXIMA, CarritoService, LineaCarrito } from './carrito.service';

/** Productos del carrito con cantidades editables. */
@Component({
  selector: 'app-lista-carrito',
  imports: [LucideAngularModule, DineroPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (carrito.vacio()) {
      <div class="card flex flex-col items-center px-6 py-16 text-center">
        <div class="mb-3 flex size-14 items-center justify-center rounded-2xl bg-brand-50 text-brand-600 dark:bg-brand-950 dark:text-brand-300">
          <lucide-icon [img]="iconos.carrito" [size]="26" aria-hidden="true" />
        </div>
        <p class="font-medium">El carrito está vacío</p>
        <p class="mt-1 text-sm text-ink-500">Escaneá un código o buscá un producto para empezar.</p>
      </div>
    } @else {
      <ul class="card divide-y divide-ink-100 dark:divide-ink-800" aria-label="Productos en el carrito">
        @for (l of carrito.lineas(); track l.productoId) {
          <li class="px-4 py-3">
            <div class="flex items-start gap-3">
              <div class="min-w-0 flex-1">
                <p class="line-clamp-2 font-medium">
                  {{ l.nombre }}
                  @if (l.tipo === 'COMBO') { <span class="badge-neutro ml-1">Combo</span> }
                </p>
                <p class="text-xs text-ink-500">
                  {{ l.sku }} · {{ l.precio | dinero }} c/u
                  @if (l.cantidad > l.stock) {
                    <span class="badge-warn ml-1">Stock registrado: {{ l.stock }}</span>
                  }
                </p>
              </div>
              <button type="button" class="btn-fantasma btn-icono -mt-1 -mr-2 text-ink-400 hover:text-danger" (click)="carrito.quitar(l.productoId)"
                      [attr.aria-label]="'Quitar ' + l.nombre + ' del carrito'">
                <lucide-icon [img]="iconos.quitar" [size]="18" aria-hidden="true" />
              </button>
            </div>
            <div class="mt-2 flex items-center justify-between gap-3">
              <div class="flex items-center gap-1" role="group" [attr.aria-label]="'Cantidad de ' + l.nombre">
                <button type="button" class="btn-secundario btn-icono" (click)="carrito.cambiarCantidad(l.productoId, l.cantidad - 1)"
                        [attr.aria-label]="'Quitar una unidad de ' + l.nombre">
                  <lucide-icon [img]="iconos.menos" [size]="16" aria-hidden="true" />
                </button>
                <input type="number" inputmode="numeric" min="1" [max]="maximo" class="input w-16 text-center tabular-nums"
                       [value]="l.cantidad" [attr.aria-label]="'Cantidad de ' + l.nombre"
                       (change)="carrito.cambiarCantidad(l.productoId, $any($event.target).valueAsNumber)" />
                <button type="button" class="btn-secundario btn-icono" (click)="carrito.cambiarCantidad(l.productoId, l.cantidad + 1)"
                        [attr.aria-label]="'Agregar una unidad de ' + l.nombre">
                  <lucide-icon [img]="iconos.mas" [size]="16" aria-hidden="true" />
                </button>
              </div>
              <p class="text-right text-lg font-semibold tabular-nums">{{ subtotal(l) | dinero }}</p>
            </div>
          </li>
        }
      </ul>
    }
  `,
})
export class ListaCarritoComponent {
  protected readonly carrito = inject(CarritoService);
  protected readonly maximo = CANTIDAD_MAXIMA;
  protected readonly iconos = { mas: Plus, menos: Minus, quitar: Trash2, carrito: ShoppingCart };

  protected subtotal(l: LineaCarrito): string {
    return deCentavos(aCentavos(l.precio) * l.cantidad);
  }
}
