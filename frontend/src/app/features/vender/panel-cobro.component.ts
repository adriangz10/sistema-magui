import { ChangeDetectionStrategy, Component, inject, input, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Banknote, CreditCard, Landmark, LoaderCircle, LucideAngularModule, QrCode, Trash2 } from 'lucide-angular';
import { METODOS, MetodoPago } from '../../core/api/ventas.api';
import { DineroPipe, deCentavos } from '../../shared/formato/dinero';
import { InputDineroComponent } from '../../shared/ui/input-dinero.component';
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
      <h2 id="titulo-cobro" class="text-sm font-medium text-ink-500">Total a cobrar</h2>
      <p class="mt-1 text-4xl font-bold tracking-tight tabular-nums">{{ total() | dinero }}</p>
      <p class="text-sm text-ink-500">{{ carrito.unidades() }} {{ carrito.unidades() === 1 ? 'unidad' : 'unidades' }}</p>

      <h3 class="mt-5 mb-2 text-sm font-medium text-ink-700 dark:text-ink-300">Forma de pago</h3>
      <div class="grid grid-cols-2 gap-2">
        @for (m of metodos; track m.valor) {
          <button type="button" class="btn-secundario justify-start" [disabled]="carrito.vacio()" (click)="agregar(m.valor)">
            <lucide-icon [img]="iconos[m.valor]" [size]="18" aria-hidden="true" /> {{ m.etiqueta }}
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
                  <button type="button" class="btn-fantasma btn-icono min-h-8 w-8 text-ink-400 hover:text-danger"
                          (click)="carrito.quitarPago(i)" [attr.aria-label]="'Quitar pago en ' + etiqueta(p.metodo)">
                    <lucide-icon [img]="iconos.quitar" [size]="16" aria-hidden="true" />
                  </button>
                </div>
              </div>
              <app-input-dinero [id]="'pago-' + i" [ngModel]="p.monto" (ngModelChange)="carrito.cambiarMonto(i, $event)" />
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
        <div class="flex justify-between"><dt class="text-ink-500">Pagado</dt><dd class="tabular-nums">{{ pagado() | dinero }}</dd></div>
        <div class="flex justify-between font-medium" [class]="carrito.cobro().falta > 0 ? 'text-danger dark:text-red-400' : 'text-ink-500'">
          <dt>Falta</dt><dd class="tabular-nums" aria-live="polite">{{ falta() | dinero }}</dd>
        </div>
        @if (carrito.cobro().vuelto > 0) {
          <div class="flex items-center justify-between rounded-xl bg-green-50 px-3 py-2 text-ok dark:bg-green-950 dark:text-green-300">
            <dt class="font-medium">Vuelto</dt><dd class="text-xl font-bold tabular-nums" aria-live="polite">{{ vuelto() | dinero }}</dd>
          </div>
        }
      </dl>

      @if (carrito.cobro().problema; as problema) {
        <p class="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-sm text-warn dark:bg-amber-950 dark:text-amber-300" role="alert">{{ problema }}</p>
      }

      <button type="button" class="btn-primario mt-5 h-12 w-full text-base" [disabled]="!carrito.cobro().listo || cobrando()" (click)="cobrar.emit()">
        @if (cobrando()) { <lucide-icon [img]="iconos.cargando" [size]="18" class="animate-spin" aria-hidden="true" /> }
        Cobrar {{ total() | dinero }} <span class="text-xs font-normal opacity-75">(F9)</span>
      </button>
    </section>
  `,
})
export class PanelCobroComponent {
  readonly cobrando = input(false);
  readonly cobrar = output<void>();

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

  protected agregar(m: MetodoPago): void {
    const i = this.carrito.agregarPago(m);
    setTimeout(() => (document.getElementById('pago-' + i) as HTMLInputElement | null)?.focus());
  }
}
