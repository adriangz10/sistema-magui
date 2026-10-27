import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { LucideAngularModule, Plus } from 'lucide-angular';
import { CatalogoApi, Empresa } from '../../core/api/catalogo.api';
import { CompraDetalle, CompraResumen, ComprasApi } from '../../core/api/compras.api';
import { Pagina } from '../../core/api/modelos';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe } from '../../shared/formato/dinero';
import { FechaPipe } from '../../shared/formato/fecha';
import { CampoComponent } from '../../shared/ui/campo.component';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';
import { ModalComponent } from '../../shared/ui/modal.component';
import { TablaComponent } from '../../shared/ui/tabla.component';
import { CompraFormComponent } from './compra-form.component';

/** Compras de mercadería: historial y carga de lo que llega. */
@Component({
  selector: 'app-compras',
  imports: [FormsModule, LucideAngularModule, DineroPipe, FechaPipe, CampoComponent, EncabezadoComponent, ModalComponent, TablaComponent, CompraFormComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-encabezado titulo="Compras" descripcion="Mercadería que llegó: suma stock, guarda el costo y actualiza precios">
      <button type="button" class="btn-primario" (click)="nueva.set(true)">
        <lucide-icon [img]="iconoMas" [size]="18" aria-hidden="true" /> Registrar compra
      </button>
    </app-encabezado>

    <div class="card mb-4 grid gap-3 p-4 sm:grid-cols-3">
      <app-campo etiqueta="Desde" para="cf-desde">
        <input id="cf-desde" type="date" class="input" [ngModel]="filtro().desde" (ngModelChange)="filtrar({ desde: $event })" />
      </app-campo>
      <app-campo etiqueta="Hasta" para="cf-hasta">
        <input id="cf-hasta" type="date" class="input" [ngModel]="filtro().hasta" (ngModelChange)="filtrar({ hasta: $event })" />
      </app-campo>
      <app-campo etiqueta="Empresa" para="cf-emp">
        <select id="cf-emp" class="input" [ngModel]="filtro().empresaId" (ngModelChange)="filtrar({ empresaId: $event })">
          <option value="">Todas</option>
          @for (e of empresas(); track e.id) { <option [value]="e.id">{{ e.nombre }}</option> }
        </select>
      </app-campo>
    </div>

    <app-tabla [pagina]="pagina()" [cargando]="cargando()" mensajeVacio="No hay compras registradas"
               ayudaVacio="Cuando llegue mercadería, registrala acá para sumar el stock." (paginaCambio)="filtrar({ page: $event }, false)">
      <table class="tabla">
        <caption class="sr-only">Compras</caption>
        <thead><tr><th scope="col">Fecha</th><th scope="col">Empresa</th><th scope="col">Detalle</th><th scope="col" class="text-right">Total</th><th scope="col"><span class="sr-only">Acciones</span></th></tr></thead>
        <tbody>
          @for (c of pagina()?.items; track c.id) {
            <tr>
              <td class="whitespace-nowrap">{{ c.fecha | fecha: 'fecha' }}</td>
              <td>{{ c.empresa ?? '—' }}</td>
              <td class="text-muted">{{ c.unidades }} unidades · {{ c.productos }} {{ c.productos === 1 ? 'producto' : 'productos' }}{{ c.notas ? ' · ' + c.notas : '' }}</td>
              <td class="text-right font-semibold tabular-nums">{{ c.total | dinero }}</td>
              <td class="text-right">
                <button type="button" class="btn-fantasma min-h-8 px-3" (click)="ver(c)" [attr.aria-label]="'Ver compra del ' + (c.fecha | fecha: 'fecha')">Ver</button>
              </td>
            </tr>
          }
        </tbody>
      </table>
    </app-tabla>

    <app-modal [abierto]="nueva()" titulo="Registrar compra" ancho="grande" (cerrar)="nueva.set(false)">
      @if (nueva()) {
        <app-compra-form [empresas]="empresasActivas()" (guardado)="guardada($event)" (cancelado)="nueva.set(false)" />
      }
    </app-modal>

    <app-modal [abierto]="!!detalle()" titulo="Detalle de la compra" ancho="grande" (cerrar)="detalle.set(null)">
      @if (detalle(); as d) {
        <p class="mb-3 text-sm text-muted">{{ d.fecha | fecha: 'fecha' }}{{ d.empresa ? ' · ' + d.empresa : '' }}{{ d.notas ? ' · ' + d.notas : '' }}</p>
        <div class="overflow-x-auto">
          <table class="tabla">
            <thead><tr><th scope="col">Producto</th><th scope="col" class="text-right">Cant.</th><th scope="col" class="text-right">Costo</th>
              <th scope="col" class="text-right">Subtotal</th><th scope="col">Nuevo precio</th><th scope="col">Vence</th></tr></thead>
            <tbody>
              @for (it of d.items; track it.id) {
                <tr>
                  <td><p class="font-medium">{{ it.producto }}</p><p class="text-xs text-muted">{{ it.sku }}</p></td>
                  <td class="text-right tabular-nums">{{ it.cantidad }}</td>
                  <td class="text-right tabular-nums">{{ it.costoUnitario | dinero }}</td>
                  <td class="text-right font-medium tabular-nums">{{ it.subtotal | dinero }}</td>
                  <td class="tabular-nums">{{ it.precioVentaNuevo ? (it.precioVentaNuevo | dinero) : '—' }}</td>
                  <td class="whitespace-nowrap">{{ it.vencimiento | fecha }}</td>
                </tr>
              }
            </tbody>
          </table>
        </div>
        <p class="mt-4 text-right text-lg">Total <strong class="tabular-nums">{{ d.total | dinero }}</strong></p>
      }
      <div pie><button type="button" class="btn-secundario" (click)="detalle.set(null)">Cerrar</button></div>
    </app-modal>
  `,
})
export class ComprasPage {
  private readonly api = inject(ComprasApi);
  private readonly catalogo = inject(CatalogoApi);
  private readonly toast = inject(ToastService);

  protected readonly iconoMas = Plus;
  protected readonly filtro = signal({ desde: '', hasta: '', empresaId: '', page: 1 });
  protected readonly pagina = signal<Pagina<CompraResumen> | null>(null);
  protected readonly cargando = signal(false);
  protected readonly empresas = signal<Empresa[]>([]);
  protected readonly empresasActivas = signal<Empresa[]>([]);
  protected readonly nueva = signal(false);
  protected readonly detalle = signal<CompraDetalle | null>(null);

  constructor() {
    void this.cargar();
    void this.catalogo.todasLasEmpresas().then((es) => {
      this.empresas.set(es);
      this.empresasActivas.set(es.filter((e) => e.activo));
    }).catch((e) => this.toast.error(e));
  }

  protected filtrar(cambio: Partial<ReturnType<typeof this.filtro>>, reiniciar = true): void {
    this.filtro.update((f) => ({ ...f, ...cambio, page: reiniciar ? 1 : (cambio.page ?? f.page) }));
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

  protected async ver(c: CompraResumen): Promise<void> {
    try {
      this.detalle.set(await firstValueFrom(this.api.obtener(c.id)));
    } catch (e) {
      this.toast.error(e);
    }
  }

  protected guardada(c: CompraDetalle): void {
    this.nueva.set(false);
    this.toast.exito(`Compra registrada: ${c.unidades} unidades sumadas al stock`);
    void this.cargar();
  }
}
