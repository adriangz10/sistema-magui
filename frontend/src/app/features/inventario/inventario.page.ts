import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ApiError } from '../../core/api/api-error';
import { Alertas, InventarioApi, Movimiento, TIPOS_MOVIMIENTO, TipoMovimiento, Vencimiento } from '../../core/api/inventario.api';
import { Pagina } from '../../core/api/modelos';
import { ToastService } from '../../core/ui/toast.service';
import { FechaPipe } from '../../shared/formato/fecha';
import { CampoComponent } from '../../shared/ui/campo.component';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';
import { MotivoModalComponent } from '../../shared/ui/motivo-modal.component';
import { TablaComponent } from '../../shared/ui/tabla.component';
import { AjusteStockComponent } from './ajuste-stock.component';
import { claseEstadoLote, textoEstadoLote } from './estados';

type Pestana = 'vencimientos' | 'ajuste' | 'movimientos';
type FiltroVenc = 'vencidos' | 'por_vencer' | 'todos';

/** Stock por lote, vencimientos, ajustes y movimientos. */
@Component({
  selector: 'app-inventario',
  imports: [FormsModule, FechaPipe, CampoComponent, EncabezadoComponent, MotivoModalComponent, TablaComponent, AjusteStockComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-encabezado titulo="Inventario" descripcion="Vencimientos, correcciones de stock e historial de movimientos" />

    @if (alertas(); as a) {
      <div class="mb-6 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <button type="button" class="card p-4 text-left transition hover:border-red-300" (click)="verVencimientos('vencidos')">
          <p class="text-sm text-ink-500">Lotes vencidos</p>
          <p class="mt-1 text-2xl font-bold" [class]="a.vencidos ? 'text-danger dark:text-red-400' : ''">{{ a.vencidos }}</p>
        </button>
        <button type="button" class="card p-4 text-left transition hover:border-amber-300" (click)="verVencimientos('por_vencer')">
          <p class="text-sm text-ink-500">Por vencer ({{ a.diasAlerta }} días)</p>
          <p class="mt-1 text-2xl font-bold" [class]="a.porVencer ? 'text-warn dark:text-amber-300' : ''">{{ a.porVencer }}</p>
        </button>
        <div class="card p-4">
          <p class="text-sm text-ink-500">Productos sin stock</p>
          <p class="mt-1 text-2xl font-bold" [class]="a.sinStock ? 'text-danger dark:text-red-400' : ''">{{ a.sinStock }}</p>
        </div>
        <div class="card p-4">
          <p class="text-sm text-ink-500">Con stock bajo</p>
          <p class="mt-1 text-2xl font-bold" [class]="a.stockBajo ? 'text-warn dark:text-amber-300' : ''">{{ a.stockBajo }}</p>
        </div>
      </div>
    }

    <div class="mb-4 flex gap-1 overflow-x-auto border-b border-ink-200 dark:border-ink-800" role="tablist" aria-label="Secciones de inventario">
      @for (p of pestanas; track p.id) {
        <button type="button" role="tab" [id]="'tab-' + p.id" [attr.aria-selected]="pestana() === p.id" [attr.aria-controls]="'panel-' + p.id"
                class="-mb-px border-b-2 px-4 py-2.5 text-sm font-medium whitespace-nowrap transition"
                [class]="pestana() === p.id ? 'border-brand-600 text-brand-700 dark:text-brand-300' : 'border-transparent text-ink-500 hover:text-ink-800 dark:hover:text-ink-200'"
                (click)="cambiarPestana(p.id)">{{ p.etiqueta }}</button>
      }
    </div>

    @switch (pestana()) {
      @case ('vencimientos') {
        <div id="panel-vencimientos" role="tabpanel" aria-labelledby="tab-vencimientos">
          <div class="mb-3 flex flex-wrap gap-2" role="group" aria-label="Filtrar vencimientos">
            @for (f of filtrosVenc; track f.id) {
              <button type="button" class="btn min-h-9 rounded-full px-4" [attr.aria-pressed]="filtroVenc() === f.id"
                      [class]="filtroVenc() === f.id ? 'bg-ink-900 text-white dark:bg-ink-100 dark:text-ink-900' : 'bg-white text-ink-700 ring-1 ring-ink-200 dark:bg-ink-900 dark:text-ink-200 dark:ring-ink-700'"
                      (click)="verVencimientos(f.id)">{{ f.etiqueta }}</button>
            }
          </div>
          <app-tabla [pagina]="vencimientos()" [cargando]="cargando()" mensajeVacio="No hay lotes en esta lista"
                     ayudaVacio="Solo aparecen lotes con stock y fecha de vencimiento." (paginaCambio)="cargarVencimientos($event)">
            <table class="tabla">
              <caption class="sr-only">Lotes por vencimiento</caption>
              <thead><tr><th scope="col">Producto</th><th scope="col">Vence</th><th scope="col">Estado</th><th scope="col" class="text-right">Stock</th><th scope="col"><span class="sr-only">Acciones</span></th></tr></thead>
              <tbody>
                @for (v of vencimientos()?.items; track v.loteId) {
                  <tr>
                    <td><p class="font-medium">{{ v.producto }}</p><p class="text-xs text-ink-500">{{ v.sku }}</p></td>
                    <td class="whitespace-nowrap">{{ v.vencimiento | fecha }}</td>
                    <td><span [class]="claseEstado(v.estado)">{{ textoEstado(v.estado, v.diasRestantes) }}</span></td>
                    <td class="text-right font-semibold tabular-nums">{{ v.stock }}</td>
                    <td class="text-right">
                      <button type="button" class="btn-fantasma min-h-8 px-3" (click)="descartando.set(v)"
                              [attr.aria-label]="'Descartar ' + v.stock + ' unidades de ' + v.producto">Descartar</button>
                    </td>
                  </tr>
                }
              </tbody>
            </table>
          </app-tabla>
        </div>
      }
      @case ('ajuste') {
        <div id="panel-ajuste" role="tabpanel" aria-labelledby="tab-ajuste">
          <app-ajuste-stock (ajustado)="cargarAlertas()" />
        </div>
      }
      @case ('movimientos') {
        <div id="panel-movimientos" role="tabpanel" aria-labelledby="tab-movimientos">
          <div class="card mb-4 grid gap-3 p-4 sm:grid-cols-3">
            <app-campo etiqueta="Tipo" para="m-tipo">
              <select id="m-tipo" class="input" [ngModel]="filtroMov().tipo" (ngModelChange)="filtrarMov({ tipo: $event })">
                <option value="">Todos</option>
                @for (t of tipos; track t[0]) { <option [value]="t[0]">{{ t[1] }}</option> }
              </select>
            </app-campo>
            <app-campo etiqueta="Desde" para="m-desde">
              <input id="m-desde" type="date" class="input" [ngModel]="filtroMov().desde" (ngModelChange)="filtrarMov({ desde: $event })" />
            </app-campo>
            <app-campo etiqueta="Hasta" para="m-hasta">
              <input id="m-hasta" type="date" class="input" [ngModel]="filtroMov().hasta" (ngModelChange)="filtrarMov({ hasta: $event })" />
            </app-campo>
          </div>
          <app-tabla [pagina]="movimientos()" [cargando]="cargando()" mensajeVacio="No hay movimientos con estos filtros"
                     (paginaCambio)="filtrarMov({ page: $event }, false)">
            <table class="tabla">
              <caption class="sr-only">Movimientos de stock</caption>
              <thead><tr><th scope="col">Fecha</th><th scope="col">Producto</th><th scope="col">Tipo</th><th scope="col" class="text-right">Cantidad</th><th scope="col">Detalle</th></tr></thead>
              <tbody>
                @for (m of movimientos()?.items; track m.id) {
                  <tr>
                    <td class="whitespace-nowrap">{{ m.fecha | fecha }}</td>
                    <td><p class="font-medium">{{ m.producto }}</p><p class="text-xs text-ink-500">{{ m.sku }}</p></td>
                    <td><span class="badge-neutro">{{ nombreTipo(m.tipo) }}</span></td>
                    <td class="text-right font-semibold tabular-nums" [class]="m.cantidad > 0 ? 'text-ok dark:text-green-400' : 'text-danger dark:text-red-400'">
                      {{ m.cantidad > 0 ? '+' : '' }}{{ m.cantidad }}
                    </td>
                    <td class="text-ink-500">{{ m.nota || (m.loteVencimiento ? 'Lote que vence el ' + (m.loteVencimiento | fecha) : '—') }}</td>
                  </tr>
                }
              </tbody>
            </table>
          </app-tabla>
        </div>
      }
    }

    <app-motivo-modal [abierto]="!!descartando()" titulo="Descartar unidades vencidas" textoConfirmar="Descartar" [procesando]="procesando()"
                      [descripcion]="descartando() ? 'Se quitan ' + descartando()!.stock + ' unidades de ' + descartando()!.producto + ' del stock.' : ''"
                      placeholder="Ej.: vencido, se tiró" [error]="errorDescarte()"
                      (confirmar)="descartar($event)" (cerrar)="descartando.set(null)" />
  `,
})
export class InventarioPage {
  private readonly api = inject(InventarioApi);
  private readonly toast = inject(ToastService);

  protected readonly pestanas: { id: Pestana; etiqueta: string }[] = [
    { id: 'vencimientos', etiqueta: 'Vencimientos' }, { id: 'ajuste', etiqueta: 'Ajustar stock' }, { id: 'movimientos', etiqueta: 'Movimientos' },
  ];
  protected readonly filtrosVenc: { id: FiltroVenc; etiqueta: string }[] = [
    { id: 'vencidos', etiqueta: 'Vencidos' }, { id: 'por_vencer', etiqueta: 'Por vencer' }, { id: 'todos', etiqueta: 'Todos con fecha' },
  ];
  protected readonly tipos = Object.entries(TIPOS_MOVIMIENTO) as [TipoMovimiento, string][];
  protected readonly claseEstado = claseEstadoLote;
  protected readonly textoEstado = textoEstadoLote;

  protected readonly pestana = signal<Pestana>('vencimientos');
  protected readonly alertas = signal<Alertas | null>(null);
  protected readonly filtroVenc = signal<FiltroVenc>('vencidos');
  protected readonly vencimientos = signal<Pagina<Vencimiento> | null>(null);
  protected readonly filtroMov = signal({ tipo: '' as TipoMovimiento | '', desde: '', hasta: '', page: 1 });
  protected readonly movimientos = signal<Pagina<Movimiento> | null>(null);
  protected readonly cargando = signal(false);
  protected readonly descartando = signal<Vencimiento | null>(null);
  protected readonly procesando = signal(false);
  protected readonly errorDescarte = signal('');

  constructor() {
    void this.cargarAlertas();
    void this.cargarVencimientos(1);
  }

  protected nombreTipo(t: TipoMovimiento): string {
    return TIPOS_MOVIMIENTO[t] ?? t;
  }

  async cargarAlertas(): Promise<void> {
    try {
      this.alertas.set(await firstValueFrom(this.api.alertas()));
    } catch (e) {
      this.toast.error(e);
    }
  }

  protected cambiarPestana(p: Pestana): void {
    this.pestana.set(p);
    if (p === 'movimientos' && !this.movimientos()) {
      void this.cargarMovimientos();
    }
  }

  protected verVencimientos(f: FiltroVenc): void {
    this.pestana.set('vencimientos');
    this.filtroVenc.set(f);
    void this.cargarVencimientos(1);
  }

  protected async cargarVencimientos(page: number): Promise<void> {
    this.cargando.set(true);
    try {
      this.vencimientos.set(await firstValueFrom(this.api.vencimientos(this.filtroVenc(), page)));
    } catch (e) {
      this.toast.error(e);
    } finally {
      this.cargando.set(false);
    }
  }

  protected filtrarMov(cambio: Partial<ReturnType<typeof this.filtroMov>>, reiniciar = true): void {
    this.filtroMov.update((f) => ({ ...f, ...cambio, page: reiniciar ? 1 : (cambio.page ?? f.page) }));
    void this.cargarMovimientos();
  }

  private async cargarMovimientos(): Promise<void> {
    this.cargando.set(true);
    try {
      this.movimientos.set(await firstValueFrom(this.api.movimientos(this.filtroMov())));
    } catch (e) {
      this.toast.error(e);
    } finally {
      this.cargando.set(false);
    }
  }

  protected async descartar(motivo: string): Promise<void> {
    const v = this.descartando();
    if (!v) {
      return;
    }
    this.procesando.set(true);
    this.errorDescarte.set('');
    try {
      await firstValueFrom(this.api.ajustar({ productoId: v.productoId, cantidad: -v.stock, motivo, loteId: v.loteId }));
      this.descartando.set(null);
      this.toast.exito(`Se descartaron ${v.stock} unidades de ${v.producto}`);
      void this.cargarAlertas();
      void this.cargarVencimientos(this.vencimientos()?.page ?? 1);
      this.movimientos.set(null);
    } catch (e) {
      const err = ApiError.desde(e);
      this.errorDescarte.set(err.campos['motivo'] ?? err.message);
    } finally {
      this.procesando.set(false);
    }
  }
}
