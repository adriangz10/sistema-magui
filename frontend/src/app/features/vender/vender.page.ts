import { ChangeDetectionStrategy, Component, inject, signal, viewChild } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { CircleCheck, LucideAngularModule } from 'lucide-angular';
import { ApiError } from '../../core/api/api-error';
import { ProductoResumen } from '../../core/api/catalogo.api';
import { OrigenVenta, VentaDetalle, VentasApi } from '../../core/api/ventas.api';
import { ES_ESCRITORIO } from '../../core/config/plataforma';
import { EscritorioService } from '../../core/escritorio/escritorio.service';
import { ConexionService } from '../../core/red/conexion.service';
import { ConfirmacionService } from '../../core/ui/confirmacion.service';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe, deCentavos } from '../../shared/formato/dinero';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';
import { ModalComponent } from '../../shared/ui/modal.component';
import { BuscadorProductoComponent } from '../../shared/ui/buscador-producto.component';
import { CarritoService } from './carrito.service';
import { ListaCarritoComponent } from './lista-carrito.component';
import { PanelCobroComponent } from './panel-cobro.component';

/** Punto de venta: escanear o buscar, armar el carrito y cobrar con pagos combinados. */
@Component({
  selector: 'app-vender',
  imports: [LucideAngularModule, DineroPipe, EncabezadoComponent, ModalComponent, BuscadorProductoComponent,
    ListaCarritoComponent, PanelCobroComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown)': 'atajos($event)' },
  template: `
    <app-encabezado titulo="Vender" descripcion="Escaneá los productos o buscalos por nombre">
      @if (!carrito.vacio()) {
        <button type="button" class="btn-secundario" (click)="vaciar()">Vaciar carrito</button>
      }
    </app-encabezado>

    <div class="grid items-start gap-6 pb-16 lg:grid-cols-[minmax(0,1fr)_24rem] lg:pb-0">
      <div class="space-y-4">
        <app-buscador-producto (elegido)="agregar($event)" />
        <app-lista-carrito />
      </div>
      <app-panel-cobro id="panel-cobro" class="scroll-mt-20 lg:sticky lg:top-20" [cobrando]="cobrando()" (cobrar)="cobrar()" />
    </div>

    <!-- Celular: total siempre visible sobre la barra de navegación -->
    @if (!carrito.vacio()) {
      <div class="fixed inset-x-0 bottom-[calc(3.6rem+env(safe-area-inset-bottom))] z-30 border-t border-ink-200 bg-white/95 px-4 py-2 backdrop-blur lg:hidden dark:border-ink-800 dark:bg-ink-900/95">
        <div class="flex items-center gap-3">
          <div class="min-w-0 flex-1">
            <p class="text-lg font-bold tabular-nums">{{ total() | dinero }}</p>
            <p class="text-xs" [class]="carrito.cobro().falta > 0 ? 'text-danger dark:text-red-400' : 'text-ok dark:text-green-400'">
              @if (carrito.cobro().falta > 0) { Falta {{ falta() | dinero }} } @else if (carrito.cobro().vuelto > 0) { Vuelto {{ vuelto() | dinero }} } @else { Listo para cobrar }
            </p>
          </div>
          @if (carrito.cobro().listo) {
            <button type="button" class="btn-primario" [disabled]="cobrando()" (click)="cobrar()">Cobrar</button>
          } @else {
            <button type="button" class="btn-secundario" (click)="irAPagos()">Pagos</button>
          }
        </div>
      </div>
    }

    <app-modal [abierto]="!!ultimaVenta()" titulo="Venta registrada" (cerrar)="nuevaVenta()">
      @if (ultimaVenta(); as v) {
        <div class="text-center">
          <lucide-icon [img]="iconoOk" [size]="48" class="mx-auto text-ok dark:text-green-400" aria-hidden="true" />
          <p class="mt-2 text-sm text-ink-500">Total cobrado</p>
          <p class="text-3xl font-bold tabular-nums">{{ v.total | dinero }}</p>
          @if (v.vuelto !== '0.00') {
            <div class="mt-4 rounded-2xl bg-green-50 p-4 text-ok dark:bg-green-950 dark:text-green-300">
              <p class="text-sm font-medium">Vuelto</p>
              <p class="text-4xl font-bold tabular-nums">{{ v.vuelto | dinero }}</p>
            </div>
          }
          <p class="mt-4 text-sm text-ink-500">{{ v.unidades }} {{ v.unidades === 1 ? 'unidad' : 'unidades' }} · {{ metodos(v) }}</p>
          @if (escritorio.activo && !conexion.enLinea()) {
            <p class="mt-2 text-sm text-warn dark:text-amber-300">Guardada en esta PC: se sube sola cuando vuelva internet.</p>
          }
        </div>
      }
      <div pie>
        <button type="button" class="btn-primario w-full sm:w-auto" (click)="nuevaVenta()" autofocus>Nueva venta</button>
      </div>
    </app-modal>
  `,
})
export class VenderPage {
  protected readonly carrito = inject(CarritoService);
  private readonly api = inject(VentasApi);
  private readonly toast = inject(ToastService);
  private readonly confirmacion = inject(ConfirmacionService);
  protected readonly escritorio = inject(EscritorioService);
  protected readonly conexion = inject(ConexionService);
  private readonly buscador = viewChild.required(BuscadorProductoComponent);

  protected readonly cobrando = signal(false);
  protected readonly ultimaVenta = signal<VentaDetalle | null>(null);
  protected readonly iconoOk = CircleCheck;

  protected agregar(p: ProductoResumen): void {
    const cantidad = this.carrito.agregar(p);
    if (cantidad > p.stock) {
      this.toast.info(`${p.nombre}: hay ${p.stock} en stock registrado`);
    }
  }

  protected atajos(e: KeyboardEvent): void {
    if (e.key === 'F2') {
      e.preventDefault();
      this.buscador().enfocar();
    } else if (e.key === 'F9') {
      e.preventDefault();
      if (this.carrito.cobro().listo && !this.cobrando()) {
        void this.cobrar();
      }
    }
  }

  protected async vaciar(): Promise<void> {
    if (await this.confirmacion.preguntar({ titulo: 'Vaciar carrito', mensaje: '¿Quitar todos los productos y pagos cargados?', confirmar: 'Vaciar', peligro: true })) {
      this.carrito.vaciar();
      this.buscador().enfocar();
    }
  }

  protected async cobrar(permitirSinStock = false): Promise<void> {
    if (this.cobrando()) {
      return;
    }
    this.cobrando.set(true);
    try {
      const venta = await firstValueFrom(this.api.registrar({
        items: this.carrito.lineas().map((l) => ({ productoId: l.productoId, cantidad: l.cantidad })),
        pagos: this.carrito.pagos().filter((p) => p.monto !== null).map((p) => ({ metodo: p.metodo, monto: p.monto! })),
        origen: origenDispositivo(),
        permitirSinStock,
      }));
      this.carrito.vaciar();
      this.ultimaVenta.set(venta);
      void this.escritorio.actualizar(); // contador de ventas por subir
    } catch (e) {
      const err = ApiError.desde(e);
      this.cobrando.set(false);
      if (err.codigo === 'stock_insuficiente' && !permitirSinStock) {
        const ok = await this.confirmacion.preguntar({
          titulo: 'Sin stock registrado',
          mensaje: `${err.message}. Si igual tenés el producto en la mano, podés vender y el stock quedará en negativo hasta corregirlo.`,
          confirmar: 'Vender igual',
        });
        if (ok) {
          await this.cobrar(true);
        }
        return;
      }
      this.toast.error(err);
    } finally {
      this.cobrando.set(false);
    }
  }

  protected total = () => deCentavos(this.carrito.totalCentavos());
  protected falta = () => deCentavos(this.carrito.cobro().falta);
  protected vuelto = () => deCentavos(this.carrito.cobro().vuelto);

  protected irAPagos(): void {
    document.getElementById('panel-cobro')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }

  protected nuevaVenta(): void {
    this.ultimaVenta.set(null);
    setTimeout(() => this.buscador().enfocar());
  }

  protected metodos(v: VentaDetalle): string {
    const nombres: Record<string, string> = { EFECTIVO: 'Efectivo', TRANSFERENCIA: 'Transferencia', QR: 'QR', TARJETA: 'Tarjeta' };
    return v.metodos.map((m) => nombres[m] ?? m).join(' + ');
  }
}

/** En pantallas táctiles (celular) la venta queda marcada como MOVIL; en el escritorio, DESKTOP. */
function origenDispositivo(): OrigenVenta {
  if (ES_ESCRITORIO) {
    return 'DESKTOP';
  }
  return globalThis.matchMedia?.('(pointer: coarse)').matches ? 'MOVIL' : 'WEB';
}
