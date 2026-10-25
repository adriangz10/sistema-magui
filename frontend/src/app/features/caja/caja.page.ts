import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { LucideAngularModule, Plus } from 'lucide-angular';
import { ApiError } from '../../core/api/api-error';
import { CajaApi, Gasto, ListadoGastos } from '../../core/api/caja.api';
import { METODOS, MetodoPago } from '../../core/api/ventas.api';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe } from '../../shared/formato/dinero';
import { FechaPipe, hoyISO } from '../../shared/formato/fecha';
import { CampoComponent } from '../../shared/ui/campo.component';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';
import { InputDineroComponent } from '../../shared/ui/input-dinero.component';
import { ModalComponent } from '../../shared/ui/modal.component';
import { MotivoModalComponent } from '../../shared/ui/motivo-modal.component';
import { TablaComponent } from '../../shared/ui/tabla.component';

/** Primer día del mes actual en Argentina. */
function inicioDeMes(): string {
  return hoyISO().slice(0, 8) + '01';
}

/** Gastos y retiros de caja (lo que sale y no es compra de mercadería). */
@Component({
  selector: 'app-caja',
  imports: [FormsModule, LucideAngularModule, DineroPipe, FechaPipe, CampoComponent, EncabezadoComponent, InputDineroComponent,
    ModalComponent, MotivoModalComponent, TablaComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-encabezado titulo="Caja" descripcion="Gastos y retiros de dinero que no son compra de mercadería">
      <button type="button" class="btn-primario" (click)="abrirNuevo()">
        <lucide-icon [img]="iconoMas" [size]="18" aria-hidden="true" /> Registrar gasto
      </button>
    </app-encabezado>

    <div class="mb-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_18rem]">
      <div class="card grid gap-3 p-4 sm:grid-cols-3">
        <app-campo etiqueta="Desde" para="g-desde">
          <input id="g-desde" type="date" class="input" [ngModel]="filtro().desde" (ngModelChange)="filtrar({ desde: $event })" />
        </app-campo>
        <app-campo etiqueta="Hasta" para="g-hasta">
          <input id="g-hasta" type="date" class="input" [ngModel]="filtro().hasta" (ngModelChange)="filtrar({ hasta: $event })" />
        </app-campo>
        <app-campo etiqueta="Categoría" para="g-cat">
          <select id="g-cat" class="input" [ngModel]="filtro().categoria" (ngModelChange)="filtrar({ categoria: $event })">
            <option value="">Todas</option>
            @for (c of categorias(); track c) { <option [value]="c">{{ c }}</option> }
          </select>
        </app-campo>
        <label class="inline-flex items-center gap-2 text-sm sm:col-span-3">
          <input type="checkbox" class="size-4 rounded accent-brand-600" [ngModel]="filtro().incluirAnulados" (ngModelChange)="filtrar({ incluirAnulados: $event })" />
          Mostrar también los anulados
        </label>
      </div>
      <div class="card flex flex-col justify-center p-5">
        <p class="text-sm text-ink-500">{{ textoPeriodo() }}</p>
        <p class="mt-1 text-3xl font-bold tabular-nums">{{ (listado()?.totalMonto ?? '0') | dinero }}</p>
        <p class="text-sm text-ink-500">{{ listado()?.total ?? 0 }} {{ listado()?.total === 1 ? 'movimiento' : 'movimientos' }}</p>
      </div>
    </div>

    <app-tabla [pagina]="listado()" [cargando]="cargando()" mensajeVacio="No hay gastos en este período"
               (paginaCambio)="filtrar({ page: $event }, false)">
      <table class="tabla">
        <caption class="sr-only">Gastos de caja</caption>
        <thead><tr><th scope="col">Fecha</th><th scope="col">Motivo</th><th scope="col">Categoría</th><th scope="col">Método</th>
          <th scope="col" class="text-right">Monto</th><th scope="col"><span class="sr-only">Acciones</span></th></tr></thead>
        <tbody>
          @for (g of listado()?.items; track g.id) {
            <tr [class.opacity-60]="g.anulado">
              <td class="whitespace-nowrap">{{ g.fecha | fecha }}</td>
              <td>
                {{ g.motivo }}
                @if (g.anulado) { <p class="text-xs text-danger dark:text-red-400">Anulado: {{ g.motivoAnulacion }}</p> }
              </td>
              <td>{{ g.categoria ?? '—' }}</td>
              <td><span class="badge-neutro">{{ nombreMetodo(g.metodo) }}</span></td>
              <td class="text-right font-semibold tabular-nums" [class.line-through]="g.anulado">{{ g.monto | dinero }}</td>
              <td class="text-right">
                @if (!g.anulado) {
                  <button type="button" class="btn-fantasma min-h-8 px-3" (click)="anulando.set(g)" [attr.aria-label]="'Anular gasto: ' + g.motivo">Anular</button>
                }
              </td>
            </tr>
          }
        </tbody>
      </table>
    </app-tabla>

    <app-modal [abierto]="nuevo()" titulo="Registrar gasto" (cerrar)="nuevo.set(false)">
      <form id="form-gasto" class="grid gap-4 sm:grid-cols-2" (ngSubmit)="guardar()">
        <app-campo etiqueta="Monto" para="ng-monto" [error]="errores()['monto']">
          <app-input-dinero idCampo="ng-monto" name="monto" [(ngModel)]="form.monto" [invalido]="!!errores()['monto']"
                            [describedBy]="errores()['monto'] ? 'ng-monto-error' : ''" />
        </app-campo>
        <app-campo etiqueta="Fecha" para="ng-fecha" [error]="errores()['fecha']">
          <input id="ng-fecha" name="fecha" type="date" class="input" [max]="hoy" [(ngModel)]="form.fecha" />
        </app-campo>
        <app-campo class="sm:col-span-2" etiqueta="Motivo" para="ng-motivo" [error]="errores()['motivo']">
          <input id="ng-motivo" name="motivo" class="input" maxlength="255" [(ngModel)]="form.motivo" placeholder="Ej.: boleta de luz"
                 [attr.aria-invalid]="errores()['motivo'] ? true : null" [attr.aria-describedby]="errores()['motivo'] ? 'ng-motivo-error' : null" />
        </app-campo>
        <app-campo etiqueta="Categoría" para="ng-cat" [opcional]="true">
          <input id="ng-cat" name="categoria" class="input" maxlength="60" list="categorias-gasto" [(ngModel)]="form.categoria" />
          <datalist id="categorias-gasto">@for (c of categorias(); track c) { <option [value]="c"></option> }</datalist>
        </app-campo>
        <app-campo etiqueta="Salió de" para="ng-metodo">
          <select id="ng-metodo" name="metodo" class="input" [(ngModel)]="form.metodo">
            @for (m of metodos; track m.valor) { <option [value]="m.valor">{{ m.etiqueta }}</option> }
          </select>
        </app-campo>
      </form>
      <div pie>
        <button type="button" class="btn-secundario" (click)="nuevo.set(false)">Cancelar</button>
        <button type="submit" form="form-gasto" class="btn-primario" [disabled]="guardando()">Registrar</button>
      </div>
    </app-modal>

    <app-motivo-modal [abierto]="!!anulando()" titulo="Anular gasto" textoConfirmar="Anular" [procesando]="guardando()"
                      descripcion="El gasto queda registrado como anulado y deja de sumarse." placeholder="Ej.: se cargó dos veces"
                      [error]="errorAnulacion()" (confirmar)="anular($event)" (cerrar)="anulando.set(null)" />
  `,
})
export class CajaPage {
  private readonly api = inject(CajaApi);
  private readonly toast = inject(ToastService);

  protected readonly iconoMas = Plus;
  protected readonly metodos = METODOS;
  protected readonly hoy = hoyISO();
  protected readonly filtro = signal({ desde: inicioDeMes(), hasta: '', categoria: '', incluirAnulados: false, page: 1 });
  protected readonly listado = signal<ListadoGastos | null>(null);
  protected readonly cargando = signal(false);
  protected readonly categorias = signal<string[]>([]);
  protected readonly nuevo = signal(false);
  protected readonly guardando = signal(false);
  protected readonly errores = signal<Record<string, string>>({});
  protected readonly anulando = signal<Gasto | null>(null);
  protected readonly errorAnulacion = signal('');
  protected form = { monto: null as string | null, fecha: hoyISO(), motivo: '', categoria: '', metodo: 'EFECTIVO' as MetodoPago };

  protected readonly textoPeriodo = computed(() => {
    const f = this.filtro();
    if (f.desde === inicioDeMes() && !f.hasta) {
      return 'Gastado este mes';
    }
    return f.desde || f.hasta ? 'Gastado en el período' : 'Gastado en total';
  });

  constructor() {
    void this.cargar();
    void this.cargarCategorias();
  }

  protected nombreMetodo(m: MetodoPago): string {
    return METODOS.find((x) => x.valor === m)?.etiqueta ?? m;
  }

  private async cargarCategorias(): Promise<void> {
    try {
      this.categorias.set(await firstValueFrom(this.api.categorias()));
    } catch (e) {
      this.toast.error(e);
    }
  }

  protected filtrar(cambio: Partial<ReturnType<typeof this.filtro>>, reiniciar = true): void {
    this.filtro.update((f) => ({ ...f, ...cambio, page: reiniciar ? 1 : (cambio.page ?? f.page) }));
    void this.cargar();
  }

  private async cargar(): Promise<void> {
    this.cargando.set(true);
    try {
      this.listado.set(await firstValueFrom(this.api.listar(this.filtro())));
    } catch (e) {
      this.toast.error(e);
    } finally {
      this.cargando.set(false);
    }
  }

  protected abrirNuevo(): void {
    this.form = { monto: null, fecha: hoyISO(), motivo: '', categoria: '', metodo: 'EFECTIVO' };
    this.errores.set({});
    this.nuevo.set(true);
  }

  protected async guardar(): Promise<void> {
    const locales: Record<string, string> = {};
    if (!this.form.monto || this.form.monto.startsWith('-') || this.form.monto === '0.00') {
      locales['monto'] = 'Ingresá un monto mayor a cero';
    }
    if (!this.form.motivo.trim()) {
      locales['motivo'] = 'Contá en qué se gastó';
    }
    this.errores.set(locales);
    if (Object.keys(locales).length) {
      return;
    }
    this.guardando.set(true);
    try {
      // Si es de hoy, se manda sin fecha para que el servidor guarde la hora exacta (si no, quedaría al mediodía).
      const fecha = this.form.fecha === hoyISO() ? '' : this.form.fecha;
      await firstValueFrom(this.api.registrar({ ...this.form, fecha, monto: this.form.monto! }));
      this.nuevo.set(false);
      this.toast.exito('Gasto registrado');
      void this.cargar();
      void this.cargarCategorias();
    } catch (e) {
      const err = ApiError.desde(e);
      if (Object.keys(err.campos).length) {
        this.errores.set({ ...err.campos });
      } else {
        this.toast.error(err);
      }
    } finally {
      this.guardando.set(false);
    }
  }

  protected async anular(motivo: string): Promise<void> {
    const g = this.anulando();
    if (!g) {
      return;
    }
    this.guardando.set(true);
    this.errorAnulacion.set('');
    try {
      await firstValueFrom(this.api.anular(g.id, motivo));
      this.anulando.set(null);
      this.toast.exito('Gasto anulado');
      void this.cargar();
    } catch (e) {
      const err = ApiError.desde(e);
      this.errorAnulacion.set(err.campos['motivo'] ?? err.message);
    } finally {
      this.guardando.set(false);
    }
  }
}
