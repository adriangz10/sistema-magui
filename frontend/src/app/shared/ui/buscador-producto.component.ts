import { ChangeDetectionStrategy, Component, ElementRef, inject, input, output, signal, viewChild } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { LoaderCircle, LucideAngularModule, ScanBarcode } from 'lucide-angular';
import { CatalogoApi, ProductoResumen, TipoProducto } from '../../core/api/catalogo.api';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe } from '../../shared/formato/dinero';

/**
 * Buscador de productos (punto de venta, armado de combos, compras). Sirve para el lector de códigos (escribe el código y Enter)
 * y para buscar por nombre, SKU o código con sugerencias mientras se escribe.
 */
@Component({
  selector: 'app-buscador-producto',
  imports: [LucideAngularModule, DineroPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'relative block' },
  template: `
    <label [for]="idCampo()" class="sr-only">{{ etiqueta() }}</label>
    <div class="relative">
      <lucide-icon [img]="iconos.escanear" [size]="20" class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted" aria-hidden="true" />
      <input #campo [id]="idCampo()" type="search" autocomplete="off" spellcheck="false"
             class="input pr-10 pl-12" [class]="compacto() ? '' : 'h-12 text-base'"
             [placeholder]="placeholder()"
             role="combobox" aria-autocomplete="list" [attr.aria-controls]="idCampo() + '-sugerencias'"
             [attr.aria-expanded]="abierto()" [attr.aria-activedescendant]="activa() >= 0 ? idCampo() + '-sug-' + activa() : null"
             [value]="texto()" (input)="alEscribir($any($event.target).value)" (keydown)="teclas($event)"
             (blur)="cerrarLuego()" />
      @if (buscando()) {
        <lucide-icon [img]="iconos.cargando" [size]="18" class="absolute top-1/2 right-3 -translate-y-1/2 animate-spin text-muted" aria-hidden="true" />
      }
    </div>

    @if (abierto()) {
      <ul [id]="idCampo() + '-sugerencias'" role="listbox" aria-label="Productos encontrados"
          class="absolute inset-x-0 top-full z-20 mt-1 max-h-96 overflow-y-auto rounded-xl border border-ink-200 bg-white py-1 shadow-md dark:border-ink-700 dark:bg-ink-900">
        @for (p of sugerencias(); track p.id; let i = $index) {
          <li [id]="idCampo() + '-sug-' + i" role="option" [attr.aria-selected]="i === activa()"
              class="flex cursor-pointer items-center gap-3 px-3 py-2.5"
              [class]="i === activa() ? 'bg-ink-100 dark:bg-ink-800' : 'hover:bg-ink-50 dark:hover:bg-ink-800'"
              (mousedown)="$event.preventDefault(); elegir(p)">
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium">
                {{ p.nombre }}
                @if (p.tipo === 'COMBO') { <span class="badge-neutro ml-1">Combo</span> }
              </p>
              <p class="text-xs text-muted">{{ p.sku }}{{ p.codigoBarras ? ' · ' + p.codigoBarras : '' }}</p>
            </div>
            <div class="text-right">
              <p class="text-sm font-semibold tabular-nums">{{ p.precioVenta | dinero }}</p>
              <p class="text-xs" [class]="p.stock > 0 ? 'text-muted' : 'text-danger'">Stock: {{ p.stock }}</p>
            </div>
          </li>
        } @empty {
          <li class="px-3 py-4 text-center text-sm text-muted">No se encontraron productos activos</li>
        }
      </ul>
    }
  `,
})
export class BuscadorProductoComponent {
  readonly idCampo = input('buscador');
  readonly etiqueta = input('Buscar producto');
  readonly placeholder = input('Escaneá o buscá por nombre, SKU o código (F2)');
  readonly compacto = input(false);
  /** Limita a un tipo (ej. solo SIMPLE para armar combos o cargar compras). */
  readonly tipo = input<TipoProducto | ''>('');
  readonly elegido = output<ProductoResumen>();

  private readonly api = inject(CatalogoApi);
  private readonly toast = inject(ToastService);
  private readonly campo = viewChild.required<ElementRef<HTMLInputElement>>('campo');

  protected readonly texto = signal('');
  protected readonly sugerencias = signal<ProductoResumen[]>([]);
  protected readonly abierto = signal(false);
  protected readonly activa = signal(-1);
  protected readonly buscando = signal(false);
  protected readonly iconos = { escanear: ScanBarcode, cargando: LoaderCircle };

  private temporizador: ReturnType<typeof setTimeout> | undefined;
  private consulta = 0;

  enfocar(): void {
    this.campo().nativeElement.focus();
  }

  protected alEscribir(t: string): void {
    this.texto.set(t);
    clearTimeout(this.temporizador);
    if (t.trim().length < 2) {
      this.abierto.set(false);
      return;
    }
    this.temporizador = setTimeout(() => void this.buscar(t.trim()), 250);
  }

  private async buscar(q: string): Promise<void> {
    const n = ++this.consulta;
    this.buscando.set(true);
    try {
      const pag = await firstValueFrom(this.api.listarProductos({ q, size: 8, estado: 'activos', tipo: this.tipo() }));
      if (n === this.consulta) {
        this.sugerencias.set(pag.items);
        this.activa.set(pag.items.length ? 0 : -1);
        this.abierto.set(true);
      }
    } catch (e) {
      this.toast.error(e);
    } finally {
      if (n === this.consulta) {
        this.buscando.set(false);
      }
    }
  }

  protected teclas(e: KeyboardEvent): void {
    const total = this.sugerencias().length;
    if (e.key === 'ArrowDown' && this.abierto() && total) {
      e.preventDefault();
      this.activa.set((this.activa() + 1) % total);
    } else if (e.key === 'ArrowUp' && this.abierto() && total) {
      e.preventDefault();
      this.activa.set((this.activa() - 1 + total) % total);
    } else if (e.key === 'Escape') {
      this.abierto.set(false);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      void this.confirmar();
    }
  }

  /** Enter: primero el código exacto (lector), si no la sugerencia marcada. */
  private async confirmar(): Promise<void> {
    const t = this.texto().trim();
    if (!t) {
      return;
    }
    clearTimeout(this.temporizador);
    if (this.abierto() && this.activa() >= 0 && !this.pareceCodigo(t)) {
      this.elegir(this.sugerencias()[this.activa()]);
      return;
    }
    this.consulta++; // descarta búsquedas pendientes
    this.buscando.set(true);
    try {
      const p = await firstValueFrom(this.api.porCodigo(t));
      if (!p.activo) {
        this.toast.error(`${p.nombre} está desactivado`);
        return;
      }
      if (this.tipo() && p.tipo !== this.tipo()) {
        this.toast.error(`${p.nombre} es un combo: elegí sus productos por separado`);
        return;
      }
      this.elegir(p);
    } catch (e) {
      if (e instanceof HttpErrorResponse && e.status === 404) {
        if (this.abierto() && this.activa() >= 0) {
          this.elegir(this.sugerencias()[this.activa()]);
        } else {
          this.toast.error(`No se encontró ningún producto con el código "${t}"`);
          this.campo().nativeElement.select();
        }
      } else {
        this.toast.error(e);
      }
    } finally {
      this.buscando.set(false);
    }
  }

  /** Un código escaneado no tiene espacios y suele tener números. */
  private pareceCodigo(t: string): boolean {
    return !/\s/.test(t) && /\d/.test(t);
  }

  protected elegir(p: ProductoResumen): void {
    this.elegido.emit(p);
    this.texto.set('');
    this.sugerencias.set([]);
    this.abierto.set(false);
    this.enfocar();
  }

  protected cerrarLuego(): void {
    setTimeout(() => this.abierto.set(false), 150);
  }
}
