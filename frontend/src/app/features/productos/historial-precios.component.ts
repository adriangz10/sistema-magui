import { ChangeDetectionStrategy, Component, effect, inject, input, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { CambioPrecio, CatalogoApi, ProductoResumen } from '../../core/api/catalogo.api';
import { Pagina } from '../../core/api/modelos';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe } from '../../shared/formato/dinero';
import { FechaPipe } from '../../shared/formato/fecha';
import { PaginadorComponent } from '../../shared/ui/paginador.component';

const ORIGEN: Record<CambioPrecio['origen'], string> = { MANUAL: 'Edición', CASCADA: 'Por categoría', COMPRA: 'Compra' };

/** Cambios de precio de un producto, del más reciente al más viejo. */
@Component({
  selector: 'app-historial-precios',
  imports: [DineroPipe, FechaPipe, PaginadorComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <p class="mb-3 text-sm text-ink-500">{{ producto().nombre }} · precio actual <strong>{{ producto().precioVenta | dinero }}</strong></p>
    @if (pagina(); as p) {
      @if (p.items.length) {
        <ul class="divide-y divide-ink-100 dark:divide-ink-800">
          @for (c of p.items; track c.id) {
            <li class="flex items-center justify-between gap-3 py-2.5 text-sm">
              <div>
                <p class="tabular-nums"><span class="text-ink-500 line-through">{{ c.precioAnterior | dinero }}</span> → <strong>{{ c.precioNuevo | dinero }}</strong></p>
                <p class="text-xs text-ink-500">{{ c.fecha | fecha }} · {{ origen(c) }}{{ c.motivo ? ' · ' + c.motivo : '' }}</p>
              </div>
              <span [class]="sube(c) ? 'badge-warn' : 'badge-ok'">{{ sube(c) ? '▲' : '▼' }}</span>
            </li>
          }
        </ul>
        <app-paginador [pagina]="p.page" [totalPaginas]="p.totalPages" [total]="p.total" (cambio)="cargar($event)" />
      } @else {
        <p class="py-8 text-center text-sm text-ink-500">Todavía no tuvo cambios de precio.</p>
      }
    }
  `,
})
export class HistorialPreciosComponent {
  readonly producto = input.required<ProductoResumen>();

  private readonly api = inject(CatalogoApi);
  private readonly toast = inject(ToastService);
  protected readonly pagina = signal<Pagina<CambioPrecio> | null>(null);

  constructor() {
    effect(() => void this.cargar(1, this.producto().id));
  }

  protected async cargar(page: number, id = this.producto().id): Promise<void> {
    try {
      this.pagina.set(await firstValueFrom(this.api.historialPrecios(id, page)));
    } catch (e) {
      this.toast.error(e);
    }
  }

  protected origen(c: CambioPrecio): string {
    return ORIGEN[c.origen] ?? c.origen;
  }

  protected sube(c: CambioPrecio): boolean {
    return Number(c.precioNuevo) > Number(c.precioAnterior);
  }
}
