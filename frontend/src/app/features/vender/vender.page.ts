import { ChangeDetectionStrategy, Component, DestroyRef, afterNextRender, inject, signal, viewChild } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { CircleCheck, LucideAngularModule, Printer } from 'lucide-angular';
import { ApiError } from '../../core/api/api-error';
import { CajaApi } from '../../core/api/caja.api';
import { NegocioService } from '../../core/negocio/negocio.service';
import { ProductoResumen } from '../../core/api/catalogo.api';
import { VentaDetalle, VentasApi } from '../../core/api/ventas.api';
import { origenDispositivo } from '../../core/config/origen';
import { EscritorioService } from '../../core/escritorio/escritorio.service';
import { LectorCodigosService } from '../../core/hardware/lector-codigos.service';
import { ConexionService } from '../../core/red/conexion.service';
import { ImpresionTicketService } from '../../core/ticket/impresion-ticket.service';
import { ConfirmacionService } from '../../core/ui/confirmacion.service';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe, deCentavos } from '../../shared/formato/dinero';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';
import { ModalComponent } from '../../shared/ui/modal.component';
import { BuscadorProductoComponent } from '../../shared/ui/buscador-producto.component';
import { ATAJOS_PAGO, TECLA_BUSCAR, TECLA_COBRAR } from './atajos';
import { CarritoService } from './carrito.service';
import { ListaCarritoComponent } from './lista-carrito.component';
import { PanelCobroComponent } from './panel-cobro.component';

/** Punto de venta: escanear o buscar, armar el carrito y cobrar con pagos combinados. */
@Component({
  selector: 'app-vender',
  imports: [RouterLink, LucideAngularModule, DineroPipe, EncabezadoComponent, ModalComponent, BuscadorProductoComponent,
    ListaCarritoComponent, PanelCobroComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { '(document:keydown)': 'atajos($event)' },
  template: `
    <app-encabezado titulo="Vender" descripcion="Escaneá los productos o buscalos por nombre">
      @if (!carrito.vacio()) {
        <button type="button" class="btn-secundario" (click)="vaciar()">Vaciar carrito</button>
      }
    </app-encabezado>

    @if (cajaCerrada()) {
      <div class="aviso-warn mb-4 flex flex-wrap items-center justify-between gap-2 rounded-xl p-3" role="status">
        <p class="text-sm font-medium">La caja está cerrada: abrila para poder cobrar.</p>
        <a routerLink="/caja" class="btn-secundario min-h-8 px-3">Abrir caja</a>
      </div>
    }

    <div class="grid items-start gap-6 pb-16 lg:grid-cols-[minmax(0,1fr)_24rem] lg:pb-0">
      <div class="space-y-4">
        <app-buscador-producto (elegido)="agregar($event)" (enterVacio)="enterEnBuscador()" />
        <p class="hidden text-xs text-muted lg:block" id="ayuda-atajos">
          <kbd>F2</kbd> buscar · <kbd>F3</kbd> Efectivo · <kbd>F4</kbd> Transferencia · <kbd>F5</kbd> QR · <kbd>F6</kbd> Tarjeta ·
          <kbd>Enter</kbd> en el monto cobra
        </p>
        <app-lista-carrito />
      </div>
      <app-panel-cobro id="panel-cobro" class="scroll-mt-20 lg:sticky lg:top-20" [cobrando]="cobrando()" (cobrar)="cobrar()"
                       (aviso)="toast.info($event)" />
    </div>

    <!-- Celular: total siempre visible sobre la barra de navegación -->
    @if (!carrito.vacio()) {
      <div class="fixed inset-x-0 bottom-[calc(3.6rem+env(safe-area-inset-bottom))] z-30 border-t border-ink-200 bg-white px-4 py-2 lg:hidden dark:border-ink-800 dark:bg-ink-900">
        <div class="flex items-center gap-3">
          <div class="min-w-0 flex-1">
            <p class="text-lg font-semibold tabular-nums">{{ total() | dinero }}</p>
            <p class="text-xs" [class]="carrito.cobro().falta > 0 ? 'text-danger' : 'text-muted'">
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
          <lucide-icon [img]="iconoOk" [size]="48" class="mx-auto text-ok" aria-hidden="true" />
          <p class="mt-2 text-sm text-muted">Total cobrado</p>
          <p class="text-3xl font-semibold tabular-nums">{{ v.total | dinero }}</p>
          @if (v.vuelto !== '0.00') {
            <div class="aviso-ok mt-4 rounded-xl p-4">
              <p class="text-sm font-medium">Vuelto</p>
              <p class="text-4xl font-semibold tabular-nums">{{ v.vuelto | dinero }}</p>
            </div>
          }
          <p class="mt-4 text-sm text-muted">{{ v.unidades }} {{ v.unidades === 1 ? 'unidad' : 'unidades' }} · {{ metodos(v) }}</p>
          @if (escritorio.activo && !conexion.enLinea()) {
            <p class="mt-2 text-sm text-warn">Guardada en esta PC: se sube sola cuando vuelva internet.</p>
          }
        </div>
      }
      <div pie>
        @if (impresion.habilitado && ultimaVenta(); as v) {
          <button type="button" class="btn-secundario w-full sm:w-auto" (click)="imprimir(v)">
            <lucide-icon [img]="iconoImprimir" [size]="18" aria-hidden="true" /> Imprimir ticket
          </button>
        }
        <button type="button" class="btn-primario w-full sm:w-auto" (click)="nuevaVenta()" autofocus>Nueva venta</button>
      </div>
    </app-modal>
  `,
})
export class VenderPage {
  protected readonly carrito = inject(CarritoService);
  private readonly api = inject(VentasApi);
  protected readonly toast = inject(ToastService);
  private readonly confirmacion = inject(ConfirmacionService);
  protected readonly escritorio = inject(EscritorioService);
  protected readonly conexion = inject(ConexionService);
  private readonly buscador = viewChild.required(BuscadorProductoComponent);
  private readonly panel = viewChild.required(PanelCobroComponent);

  private readonly caja = inject(CajaApi);
  protected readonly cobrando = signal(false);
  /** Sin caja abierta no se vende (US-002). El escritorio cobra sin conexión y no lo verifica. */
  protected readonly cajaCerrada = signal(false);
  protected readonly ultimaVenta = signal<VentaDetalle | null>(null);
  protected readonly iconoOk = CircleCheck;
  protected readonly iconoImprimir = Printer;
  protected readonly impresion = inject(ImpresionTicketService);
  private readonly negocio = inject(NegocioService);

  constructor() {
    // Al entrar, el cursor queda en el buscador: el lector de códigos agrega directo al carrito.
    afterNextRender(() => this.buscador().enfocar());
    void this.revisarCaja();
    // Lector de códigos con el foco fuera del buscador (por ejemplo, después de tocar un botón).
    inject(LectorCodigosService).lecturas.pipe(takeUntilDestroyed(inject(DestroyRef))).subscribe((l) => {
      if (!document.querySelector('dialog[open]')) {
        this.buscador().buscarCodigo(l.codigo);
      }
    });
  }

  protected async imprimir(v: VentaDetalle): Promise<void> {
    await this.impresion.imprimir(v);
  }

  private async revisarCaja(): Promise<void> {
    if (this.escritorio.activo) {
      return;
    }
    try {
      this.cajaCerrada.set((await firstValueFrom(this.caja.actual())) === null);
    } catch {
      // Sin conexión o error: no se muestra el aviso; el servidor igual lo valida al cobrar.
    }
  }

  protected agregar(p: ProductoResumen): void {
    const cantidad = this.carrito.agregar(p);
    if (cantidad > p.stock) {
      this.toast.info(`${p.nombre}: hay ${p.stock} en stock registrado`);
    }
  }

  protected atajos(e: KeyboardEvent): void {
    // Esc en Vender: lista para la próxima lectura (si no lo usó un diálogo o la lista de sugerencias).
    if (e.key === 'Escape') {
      if (!e.defaultPrevented && !document.querySelector('dialog[open]')) {
        this.buscador().enfocar();
      }
      return;
    }
    const metodo = ATAJOS_PAGO[e.key];
    if (e.key !== TECLA_BUSCAR && e.key !== TECLA_COBRAR && !metodo) {
      return;
    }
    e.preventDefault(); // F3 buscaría en la página y F5 la recargaría
    if (e.ctrlKey || e.altKey || e.metaKey || document.querySelector('dialog[open]')) {
      return;
    }
    if (e.key === TECLA_BUSCAR) {
      this.buscador().enfocar();
    } else if (e.key === TECLA_COBRAR) {
      this.cobrarSiEstaListo();
    } else if (this.carrito.vacio()) {
      this.toast.info('Primero agregá los productos');
    } else {
      this.panel().agregar(metodo);
    }
  }

  /** Enter con el buscador vacío: cobra si ya está todo pagado; si no, recuerda cómo elegir el pago. */
  protected enterEnBuscador(): void {
    if (this.carrito.vacio()) {
      return;
    }
    if (this.carrito.pagos().length === 0) {
      this.toast.info('Elegí la forma de pago: F3 Efectivo · F4 Transferencia · F5 QR · F6 Tarjeta');
      return;
    }
    this.cobrarSiEstaListo();
  }

  private cobrarSiEstaListo(): void {
    if (this.carrito.cobro().listo && !this.cobrando()) {
      void this.cobrar();
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
      if (this.impresion.habilitado && this.negocio.ticket().imprimirAlCobrar) {
        void this.impresion.imprimir(venta);
      }
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
      if (err.codigo === 'caja_cerrada') {
        this.cajaCerrada.set(true);
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
