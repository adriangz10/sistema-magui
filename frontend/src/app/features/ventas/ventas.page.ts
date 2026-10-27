import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ApiError } from '../../core/api/api-error';
import { Pagina } from '../../core/api/modelos';
import { EstadoVenta, FiltroVentas, METODOS, MetodoPago, VentaDetalle, VentaResumen, VentasApi } from '../../core/api/ventas.api';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe } from '../../shared/formato/dinero';
import { FechaPipe, hoyISO } from '../../shared/formato/fecha';
import { CampoComponent } from '../../shared/ui/campo.component';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';
import { ModalComponent } from '../../shared/ui/modal.component';
import { MotivoModalComponent } from '../../shared/ui/motivo-modal.component';
import { TablaComponent } from '../../shared/ui/tabla.component';

const NOMBRE_METODO: Record<MetodoPago, string> = { EFECTIVO: 'Efectivo', TRANSFERENCIA: 'Transferencia', QR: 'QR', TARJETA: 'Tarjeta' };

/** Historial de ventas con filtros, detalle y anulación. */
@Component({
  selector: 'app-ventas',
  imports: [FormsModule, DineroPipe, FechaPipe, CampoComponent, EncabezadoComponent, ModalComponent, MotivoModalComponent, TablaComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-encabezado titulo="Ventas" descripcion="Todas las ventas registradas, de la más reciente a la más vieja" />

    <button type="button" class="btn-secundario mb-3 w-full lg:hidden" (click)="filtrosAbiertos.set(!filtrosAbiertos())"
            [attr.aria-expanded]="filtrosAbiertos()" aria-controls="filtros-ventas">
      {{ filtrosAbiertos() ? 'Ocultar filtros' : 'Filtros' }}{{ filtrosActivos() ? ' (' + filtrosActivos() + ')' : '' }}
    </button>
    <form id="filtros-ventas" class="card mb-4 gap-3 p-4 sm:grid-cols-2 lg:grid lg:grid-cols-[repeat(4,minmax(0,1fr))_auto]"
          [class]="filtrosAbiertos() ? 'grid' : 'hidden'" (submit)="$event.preventDefault()">
      <app-campo etiqueta="Desde" para="f-desde">
        <input id="f-desde" type="date" class="input" [ngModel]="filtro().desde" (ngModelChange)="filtrar({ desde: $event })" name="desde" />
      </app-campo>
      <app-campo etiqueta="Hasta" para="f-hasta">
        <input id="f-hasta" type="date" class="input" [ngModel]="filtro().hasta" (ngModelChange)="filtrar({ hasta: $event })" name="hasta" />
      </app-campo>
      <app-campo etiqueta="Estado" para="f-estado">
        <select id="f-estado" class="input" [ngModel]="filtro().estado" (ngModelChange)="filtrar({ estado: $event })" name="estado">
          <option value="">Todas</option>
          <option value="CONFIRMADA">Confirmadas</option>
          <option value="ANULADA">Anuladas</option>
        </select>
      </app-campo>
      <app-campo etiqueta="Forma de pago" para="f-metodo">
        <select id="f-metodo" class="input" [ngModel]="filtro().metodo" (ngModelChange)="filtrar({ metodo: $event })" name="metodo">
          <option value="">Todas</option>
          @for (m of metodos; track m.valor) { <option [value]="m.valor">{{ m.etiqueta }}</option> }
        </select>
      </app-campo>
      <div class="flex items-end gap-2">
        <button type="button" class="btn-secundario" (click)="filtrar({ desde: hoy, hasta: hoy })">Hoy</button>
        <button type="button" class="btn-fantasma" (click)="limpiar()">Limpiar</button>
      </div>
    </form>

    <app-tabla [pagina]="pagina()" [cargando]="cargando()" mensajeVacio="No hay ventas con estos filtros"
               ayudaVacio="Las ventas aparecen acá apenas se cobran desde Vender." (paginaCambio)="filtrar({ page: $event }, false)">
      <table class="tabla">
        <caption class="sr-only">Ventas</caption>
        <thead>
          <tr><th scope="col">Fecha</th><th scope="col">Detalle</th><th scope="col">Forma de pago</th>
            <th scope="col" class="text-right">Total</th><th scope="col">Estado</th><th scope="col"><span class="sr-only">Acciones</span></th></tr>
        </thead>
        <tbody>
          @for (v of pagina()?.items; track v.id) {
            <tr>
              <td class="whitespace-nowrap">{{ v.fecha | fecha }}</td>
              <td class="text-muted">{{ v.unidades }} {{ v.unidades === 1 ? 'unidad' : 'unidades' }} · {{ v.productos }} {{ v.productos === 1 ? 'producto' : 'productos' }}</td>
              <td>
                <div class="flex flex-wrap gap-1">
                  @for (m of v.metodos; track m) { <span class="badge-neutro">{{ nombreMetodo(m) }}</span> }
                </div>
              </td>
              <td class="text-right font-semibold tabular-nums" [class.line-through]="v.estado === 'ANULADA'">{{ v.total | dinero }}</td>
              <td><span [class]="v.estado === 'ANULADA' ? 'badge-danger' : 'badge-neutro'">{{ v.estado === 'ANULADA' ? 'Anulada' : 'Confirmada' }}</span></td>
              <td class="text-right">
                <button type="button" class="btn-fantasma min-h-8 px-3" (click)="verDetalle(v)"
                        [attr.aria-label]="'Ver venta del ' + (v.fecha | fecha) + ' por ' + (v.total | dinero)">Ver</button>
              </td>
            </tr>
          }
        </tbody>
      </table>
    </app-tabla>

    <!-- Detalle -->
    <app-modal [abierto]="!!detalle()" titulo="Detalle de la venta" ancho="grande" (cerrar)="detalle.set(null)">
      @if (detalle(); as d) {
        <div class="mb-4 flex flex-wrap items-center gap-2 text-sm text-muted">
          <span>{{ d.fecha | fecha }}</span>
          <span [class]="d.estado === 'ANULADA' ? 'badge-danger' : 'badge-neutro'">{{ d.estado === 'ANULADA' ? 'Anulada' : 'Confirmada' }}</span>
          <span class="badge-neutro">{{ d.origen === 'MOVIL' ? 'Celular' : d.origen === 'DESKTOP' ? 'Escritorio' : 'Web' }}</span>
        </div>
        @if (d.estado === 'ANULADA') {
          <p class="aviso-danger mb-4">
            Anulada el {{ d.anuladaEn | fecha }}: {{ d.motivoAnulacion }}
          </p>
        }
        <div class="overflow-x-auto">
          <table class="tabla">
            <thead><tr><th scope="col">Producto</th><th scope="col" class="text-right">Cant.</th><th scope="col" class="text-right">Precio</th><th scope="col" class="text-right">Subtotal</th></tr></thead>
            <tbody>
              @for (it of d.items; track it.productoId) {
                <tr>
                  <td><p class="font-medium">{{ it.producto }}</p><p class="text-xs text-muted">{{ it.sku }}{{ it.tipo === 'COMBO' ? ' · Combo' : '' }}</p></td>
                  <td class="text-right tabular-nums">{{ it.cantidad }}</td>
                  <td class="text-right tabular-nums">{{ it.precioUnitario | dinero }}</td>
                  <td class="text-right font-medium tabular-nums">{{ it.subtotal | dinero }}</td>
                </tr>
              }
            </tbody>
          </table>
        </div>
        <div class="mt-4 grid gap-4 sm:grid-cols-2">
          <div>
            <h3 class="mb-2 text-sm font-medium">Pagos</h3>
            <ul class="space-y-1 text-sm">
              @for (p of d.pagos; track $index) {
                <li class="flex justify-between">
                  <span>{{ nombreMetodo(p.metodo) }}@if (p.montoRecibido) { <span class="text-muted"> (recibido {{ p.montoRecibido | dinero }}, vuelto {{ p.vuelto | dinero }})</span> }</span>
                  <span class="tabular-nums">{{ p.monto | dinero }}</span>
                </li>
              }
            </ul>
          </div>
          <dl class="space-y-1 text-sm">
            <div class="flex justify-between font-semibold"><dt>Total</dt><dd class="tabular-nums">{{ d.total | dinero }}</dd></div>
            <div class="flex justify-between text-muted"><dt>Costo de la mercadería</dt><dd class="tabular-nums">{{ d.costoTotal | dinero }}</dd></div>
            <div class="flex justify-between font-medium">
              <dt>Ganancia{{ d.costoCompleto ? '' : ' aproximada' }}</dt><dd class="tabular-nums">{{ d.ganancia | dinero }}</dd>
            </div>
            @if (!d.costoCompleto) {
              <p class="pt-1 text-xs text-warn">
                Algún producto de esta venta no tiene costo cargado, así que la ganancia real es menor.
                Se completa sola cuando se registra una compra de ese producto.
              </p>
            }
          </dl>
        </div>
      }
      <div pie>
        @if (detalle()?.estado === 'CONFIRMADA') {
          <button type="button" class="btn-peligro mr-auto" (click)="pedirAnulacion()">Anular venta</button>
        }
        <button type="button" class="btn-secundario" (click)="detalle.set(null)">Cerrar</button>
      </div>
    </app-modal>

    <!-- Anulación -->
    <app-motivo-modal [abierto]="anulando()" titulo="Anular venta" textoConfirmar="Anular" [procesando]="procesando()"
                      [error]="errorMotivo()" placeholder="Ej.: devolución del cliente, error de carga"
                      descripcion="El stock vuelve a los lotes de donde salió. La venta queda registrada como anulada."
                      (confirmar)="anular($event)" (cerrar)="anulando.set(false)" />
  `,
})
export class VentasPage {
  private readonly api = inject(VentasApi);
  private readonly toast = inject(ToastService);

  protected readonly metodos = METODOS;
  protected readonly hoy = hoyISO();
  protected readonly filtro = signal<Required<Omit<FiltroVentas, 'estado' | 'metodo'>> & { estado: EstadoVenta | ''; metodo: MetodoPago | '' }>(
    { desde: '', hasta: '', estado: '', metodo: '', page: 1 });
  protected readonly pagina = signal<Pagina<VentaResumen> | null>(null);
  protected readonly cargando = signal(false);
  protected readonly filtrosAbiertos = signal(false);
  protected readonly filtrosActivos = computed(() => {
    const f = this.filtro();
    return [f.desde, f.hasta, f.estado, f.metodo].filter(Boolean).length;
  });
  protected readonly detalle = signal<VentaDetalle | null>(null);
  protected readonly anulando = signal(false);
  protected readonly procesando = signal(false);
  protected readonly errorMotivo = signal('');

  constructor() {
    void this.cargar();
  }

  protected nombreMetodo(m: MetodoPago): string {
    return NOMBRE_METODO[m] ?? m;
  }

  /** Cambia filtros y recarga (vuelve a la página 1 salvo al paginar). */
  protected filtrar(cambio: Partial<ReturnType<typeof this.filtro>>, reiniciarPagina = true): void {
    this.filtro.update((f) => ({ ...f, ...cambio, page: reiniciarPagina ? 1 : (cambio.page ?? f.page) }));
    void this.cargar();
  }

  protected limpiar(): void {
    this.filtro.set({ desde: '', hasta: '', estado: '', metodo: '', page: 1 });
    void this.cargar();
  }

  private async cargar(): Promise<void> {
    this.cargando.set(true);
    try {
      this.pagina.set(await firstValueFrom(this.api.listar(this.filtro())));
    } catch (e) {
      this.toast.error(e);
    } finally {
      this.cargando.set(false);
    }
  }

  protected async verDetalle(v: VentaResumen): Promise<void> {
    try {
      this.detalle.set(await firstValueFrom(this.api.obtener(v.id)));
    } catch (e) {
      this.toast.error(e);
    }
  }

  protected pedirAnulacion(): void {
    this.errorMotivo.set('');
    this.anulando.set(true);
  }

  protected async anular(motivo: string): Promise<void> {
    const d = this.detalle();
    if (!d) {
      return;
    }
    this.procesando.set(true);
    try {
      const anulada = await firstValueFrom(this.api.anular(d.id, motivo));
      this.anulando.set(false);
      this.detalle.set(anulada);
      this.toast.exito('Venta anulada. El stock se repuso.');
      void this.cargar();
    } catch (e) {
      const err = ApiError.desde(e);
      if (err.campos['motivo']) {
        this.errorMotivo.set(err.campos['motivo']);
      } else {
        this.toast.error(err);
      }
    } finally {
      this.procesando.set(false);
    }
  }
}
