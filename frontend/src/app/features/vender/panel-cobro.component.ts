import { ChangeDetectionStrategy, Component, inject, input, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Banknote, CreditCard, Landmark, LoaderCircle, LucideAngularModule, QrCode, Trash2 } from 'lucide-angular';
import { METODOS, MetodoPago } from '../../core/api/ventas.api';
import { DineroPipe, deCentavos, formatearDinero } from '../../shared/formato/dinero';
import { InputDineroComponent } from '../../shared/ui/input-dinero.component';
import { teclaDe } from './atajos';
import { CarritoService } from './carrito.service';

/** Billetes argentinos más usados, para sumar rápido el efectivo recibido. */
const BILLETES = [1000, 2000, 10000, 20000];

/** Total, pagos combinados y botón de cobrar. */
@Component({
  selector: 'app-panel-cobro',
  imports: [FormsModule, LucideAngularModule, DineroPipe, InputDineroComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="card p-5" aria-labelledby="titulo-cobro">
      <h2 id="titulo-cobro" class="text-sm font-medium text-muted">Total a cobrar</h2>
      <p class="mt-1 text-3xl font-semibold tracking-tight tabular-nums">{{ total() | dinero }}</p>
      <p class="text-sm text-muted">{{ carrito.unidades() }} {{ carrito.unidades() === 1 ? 'unidad' : 'unidades' }}</p>

      <h3 class="mt-5 mb-2 text-sm font-medium">Forma de pago</h3>
      <div class="grid grid-cols-2 gap-2">
        @for (m of metodos; track m.valor) {
          <!-- Ícono a la izquierda; nombre arriba y tecla abajo: el nombre usa todo el ancho y los cuatro quedan iguales. -->
          <button type="button" class="btn-secundario grid min-h-12 grid-cols-[1.125rem_minmax(0,1fr)] items-center gap-x-2.5 gap-y-0.5 px-3 py-1.5 text-left"
                  [disabled]="carrito.vacio()" (click)="agregar(m.valor)" [attr.aria-keyshortcuts]="tecla(m.valor)">
            <lucide-icon [img]="iconos[m.valor]" [size]="18" class="row-span-2" aria-hidden="true" />
            <span class="truncate leading-tight">{{ m.etiqueta }}</span>
            <kbd class="justify-self-start leading-tight font-normal text-muted">{{ tecla(m.valor) }}</kbd>
          </button>
        }
      </div>

      @if (carrito.pagos().length) {
        <ul class="mt-4 space-y-3" aria-label="Pagos cargados">
          @for (p of carrito.pagos(); track $index; let i = $index) {
            <li class="rounded-xl border border-ink-200 p-3 dark:border-ink-700">
              <div class="mb-2 flex items-center justify-between">
                <label [for]="'pago-' + i" class="text-sm font-medium">
                  {{ etiqueta(p.metodo) }}{{ p.metodo === 'EFECTIVO' ? ' · recibido' : '' }}
                </label>
                <div class="flex items-center gap-1">
                  <button type="button" class="btn-fantasma min-h-8 px-2 text-xs" (click)="carrito.completar(i)">
                    {{ p.metodo === 'EFECTIVO' ? 'Exacto' : 'Lo que falta' }}
                  </button>
                  <button type="button" class="btn-fantasma btn-icono min-h-8 w-8 text-muted hover:text-danger"
                          (click)="carrito.quitarPago(i)" [attr.aria-label]="'Quitar pago en ' + etiqueta(p.metodo)">
                    <lucide-icon [img]="iconos.quitar" [size]="16" aria-hidden="true" />
                  </button>
                </div>
              </div>
              <app-input-dinero [idCampo]="'pago-' + i" [ngModel]="p.monto" (ngModelChange)="carrito.cambiarMonto(i, $event)"
                                (keydown.enter)="enterEnMonto($event)" />
              @if (p.metodo === 'EFECTIVO') {
                <div class="mt-2 flex flex-wrap gap-1">
                  @for (b of billetes; track b) {
                    <button type="button" class="btn-secundario min-h-8 px-2.5 text-xs" (click)="carrito.sumarBillete(i, b)">
                      + {{ b | dinero }}
                    </button>
                  }
                </div>
              }
            </li>
          }
        </ul>
      }

      <dl class="mt-5 space-y-1.5 border-t border-ink-100 pt-4 text-sm dark:border-ink-800">
        <div class="flex justify-between"><dt class="text-muted">Pagado</dt><dd class="tabular-nums">{{ pagado() | dinero }}</dd></div>
        <div class="flex justify-between font-medium" [class]="carrito.cobro().falta > 0 ? 'text-danger' : 'text-muted'">
          <dt>Falta</dt><dd class="tabular-nums" aria-live="polite">{{ falta() | dinero }}</dd>
        </div>
        @if (carrito.cobro().vuelto > 0) {
          <div class="aviso-ok flex items-center justify-between">
            <dt class="font-medium">Vuelto</dt><dd class="text-xl font-semibold tabular-nums" aria-live="polite">{{ vuelto() | dinero }}</dd>
          </div>
        }
      </dl>

      @if (carrito.cobro().problema; as problema) {
        <p class="aviso-warn mt-3" role="alert">{{ problema }}</p>
      }

      <button type="button" class="btn-primario mt-5 h-12 w-full text-base" [disabled]="!carrito.cobro().listo || cobrando()" (click)="cobrar.emit()">
        @if (cobrando()) { <lucide-icon [img]="iconos.cargando" [size]="18" class="animate-spin" aria-hidden="true" /> }
        Cobrar {{ total() | dinero }} <span class="text-xs font-normal opacity-75">(Enter)</span>
      </button>
    </section>
  `,
})
export class PanelCobroComponent {
  readonly cobrando = input(false);
  readonly cobrar = output<void>();
  /** Enter sin poder cobrar todavía: por qué. */
  readonly aviso = output<string>();

  protected readonly carrito = inject(CarritoService);
  protected readonly metodos = METODOS;
  protected readonly billetes = BILLETES;
  protected readonly iconos: Record<MetodoPago | 'quitar' | 'cargando', typeof Banknote> = {
    EFECTIVO: Banknote, TRANSFERENCIA: Landmark, QR: QrCode, TARJETA: CreditCard, quitar: Trash2, cargando: LoaderCircle,
  };

  protected total = () => deCentavos(this.carrito.totalCentavos());
  protected pagado = () => deCentavos(this.carrito.cobro().pagado);
  protected falta = () => deCentavos(this.carrito.cobro().falta);
  protected vuelto = () => deCentavos(this.carrito.cobro().vuelto);

  protected etiqueta(m: MetodoPago): string {
    return METODOS.find((x) => x.valor === m)?.etiqueta ?? m;
  }

  protected tecla = teclaDe;

  /**
   * Elige una forma de pago y pone el cursor en su monto (seleccionado): en efectivo se escribe lo que
   * entrega el cliente y se ve el vuelto; después, Enter cobra.
   */
  agregar(m: MetodoPago): void {
    if (this.carrito.vacio()) {
      return;
    }
    const i = this.carrito.agregarPago(m);
    setTimeout(() => {
      const campo = document.getElementById('pago-' + i) as HTMLInputElement | null;
      campo?.focus();
      campo?.select();
    });
  }

  /** Enter en un monto: cobra si ya está todo; si no, avisa qué falta y deja el cursor ahí. */
  protected enterEnMonto(e: Event): void {
    e.preventDefault();
    if (this.carrito.cobro().listo && !this.cobrando()) {
      this.cobrar.emit();
    } else if (!this.cobrando()) {
      this.aviso.emit(this.carrito.cobro().problema ?? `Falta cobrar ${formatearDinero(this.falta())}`);
    }
  }
}
