import { ChangeDetectionStrategy, Component, computed, effect, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { LucideAngularModule, Trash2 } from 'lucide-angular';
import { ApiError } from '../../core/api/api-error';
import { Empresa, ProductoResumen } from '../../core/api/catalogo.api';
import { CompraDetalle, ComprasApi } from '../../core/api/compras.api';
import { origenDispositivo } from '../../core/config/origen';
import { ToastService } from '../../core/ui/toast.service';
import { aCentavos, DineroPipe, deCentavos } from '../../shared/formato/dinero';
import { FechaPipe, hoyISO } from '../../shared/formato/fecha';
import { BuscadorProductoComponent } from '../../shared/ui/buscador-producto.component';
import { CampoComponent } from '../../shared/ui/campo.component';
import { InputDineroComponent } from '../../shared/ui/input-dinero.component';
import { armarEdicion, cabeceraDe, hayCambios, seAplicoEdicion } from './edicion-compra';

interface Renglon {
  clave: number;
  productoId: string;
  nombre: string;
  sku: string;
  precioActual: string;
  cantidad: number;
  costo: string | null;
  precioNuevo: string | null;
  vencimiento: string;
}

/** Carga de una compra: productos, cantidades, costo, vencimiento y nuevo precio de reventa. */
@Component({
  selector: 'app-compra-form',
  imports: [FormsModule, LucideAngularModule, DineroPipe, FechaPipe, BuscadorProductoComponent, CampoComponent, InputDineroComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="space-y-5">
      <div class="grid gap-4 sm:grid-cols-3">
        <app-campo etiqueta="Fecha" para="c-fecha" [error]="errores()['fecha']">
          <input id="c-fecha" type="date" class="input" [max]="hoy" [(ngModel)]="fecha" />
        </app-campo>
        <app-campo etiqueta="Empresa" para="c-empresa" [opcional]="true" [error]="errores()['empresaId']">
          <select id="c-empresa" class="input" [(ngModel)]="empresaId">
            <option value="">Sin empresa</option>
            @for (e of empresas(); track e.id) { <option [value]="e.id">{{ e.nombre }}</option> }
          </select>
        </app-campo>
        <app-campo etiqueta="Notas" para="c-notas" [opcional]="true" [error]="errores()['notas']">
          <input id="c-notas" class="input" maxlength="500" [(ngModel)]="notas" placeholder="Ej.: pedido ciclo 15" [attr.aria-invalid]="errores()['notas'] ? true : null" />
        </app-campo>
      </div>

      @if (esEdicion()) {
        <app-campo etiqueta="Motivo de la edición" para="c-motivo" [opcional]="true" [error]="errores()['motivo']">
          <input id="c-motivo" class="input" maxlength="255" [(ngModel)]="motivo" placeholder="Ej.: error de carga" />
        </app-campo>

        @if (desactualizada()) {
          <div class="aviso-danger flex flex-wrap items-center justify-between gap-3" role="alert">
            <span>Otra persona modificó esta compra, recargá para ver los datos actuales. Tus cambios no se guardaron.</span>
            <button type="button" class="btn-secundario" (click)="recargar.emit()">Recargar</button>
          </div>
        }

        <p class="aviso-info">Por ahora solo se puede editar la cabecera. Cambiar los renglones llega en una próxima versión.</p>
        <div class="overflow-x-auto">
          <table class="tabla">
            <caption class="sr-only">Renglones de la compra (solo lectura)</caption>
            <thead><tr><th scope="col">Producto</th><th scope="col" class="text-right">Cant.</th><th scope="col" class="text-right">Costo</th>
              <th scope="col">Vence</th><th scope="col" class="text-right">Subtotal</th></tr></thead>
            <tbody>
              @for (it of compra()!.items; track it.id) {
                <tr>
                  <td><p class="font-medium">{{ it.producto }}</p><p class="text-xs text-muted">{{ it.sku }}</p></td>
                  <td class="text-right tabular-nums">{{ it.cantidad }}</td>
                  <td class="text-right tabular-nums">{{ it.costoUnitario | dinero }}</td>
                  <td class="whitespace-nowrap">{{ it.vencimiento | fecha }}</td>
                  <td class="text-right font-medium tabular-nums">{{ it.subtotal | dinero }}</td>
                </tr>
              }
            </tbody>
          </table>
        </div>
        <div class="flex flex-wrap items-center justify-between gap-3 border-t border-ink-100 pt-4 dark:border-ink-800">
          <p class="text-lg">Total de la compra <strong class="tabular-nums">{{ compra()!.total | dinero }}</strong></p>
          <div class="flex gap-2">
            <button type="button" class="btn-secundario" (click)="cancelado.emit()">Cancelar</button>
            <button type="button" class="btn-primario" [disabled]="guardando() || desactualizada()" (click)="guardarEdicion()">Guardar cambios</button>
          </div>
        </div>
      } @else {

      <app-buscador-producto idCampo="compra-buscar" etiqueta="Agregar producto a la compra" [compacto]="true" tipo="SIMPLE"
                             placeholder="Escaneá o buscá un producto para agregarlo" (elegido)="agregar($event)" />

      @if (renglones().length) {
        <ul class="space-y-3" aria-label="Productos de la compra">
          @for (r of renglones(); track r.clave; let i = $index) {
            <li class="rounded-xl border border-ink-200 p-4 dark:border-ink-700">
              <div class="mb-3 flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <p class="font-medium">{{ r.nombre }}</p>
                  <p class="text-xs text-muted">{{ r.sku }} · precio actual {{ r.precioActual | dinero }}</p>
                </div>
                <button type="button" class="btn-fantasma btn-icono -mt-1 -mr-2 text-muted hover:text-danger" (click)="quitar(r)" [attr.aria-label]="'Quitar ' + r.nombre">
                  <lucide-icon [img]="iconoQuitar" [size]="18" aria-hidden="true" />
                </button>
              </div>
              <div class="grid gap-3 sm:grid-cols-4">
                <app-campo etiqueta="Cantidad" [para]="'c-cant-' + r.clave" [error]="errorRenglon(i, 'cantidad')">
                  <input [id]="'c-cant-' + r.clave" type="number" min="1" inputmode="numeric" class="input" [(ngModel)]="r.cantidad" (ngModelChange)="refrescar()" />
                </app-campo>
                <app-campo etiqueta="Costo c/u" [para]="'c-costo-' + r.clave" [error]="errorRenglon(i, 'costoUnitario')">
                  <app-input-dinero [idCampo]="'c-costo-' + r.clave" [(ngModel)]="r.costo" (ngModelChange)="refrescar()" />
                </app-campo>
                <app-campo etiqueta="Nuevo precio" [para]="'c-precio-' + r.clave" [opcional]="true" [error]="errorRenglon(i, 'precioVentaNuevo')">
                  <app-input-dinero [idCampo]="'c-precio-' + r.clave" [(ngModel)]="r.precioNuevo" placeholder="Sin cambio" />
                </app-campo>
                <app-campo etiqueta="Vencimiento" [para]="'c-venc-' + r.clave" [opcional]="true">
                  <input [id]="'c-venc-' + r.clave" type="date" class="input" [(ngModel)]="r.vencimiento" />
                </app-campo>
              </div>
              <p class="mt-2 text-right text-sm text-muted">Subtotal <strong class="text-ink-900 tabular-nums dark:text-ink-100">{{ subtotal(r) | dinero }}</strong></p>
            </li>
          }
        </ul>
      } @else {
        <p class="rounded-xl border border-dashed border-ink-300 px-6 py-10 text-center text-sm text-muted dark:border-ink-700">
          Agregá los productos que llegaron. Si un producto viene con dos vencimientos distintos, agregalo dos veces.
        </p>
      }
      @if (errores()['items']) { <p class="text-sm text-danger" role="alert">{{ errores()['items'] }}</p> }

      <div class="flex flex-wrap items-center justify-between gap-3 border-t border-ink-100 pt-4 dark:border-ink-800">
        <p class="text-lg">Total de la compra <strong class="tabular-nums">{{ total() | dinero }}</strong>
          <span class="text-sm text-muted">· {{ unidades() }} unidades</span></p>
        <div class="flex gap-2">
          <button type="button" class="btn-secundario" (click)="cancelado.emit()">Cancelar</button>
          <button type="button" class="btn-primario" [disabled]="guardando() || !renglones().length" (click)="guardar()">Registrar compra</button>
        </div>
      </div>
      }
    </div>
  `,
})
export class CompraFormComponent {
  readonly empresas = input<Empresa[]>([]);
  /** Si viene, el formulario edita la cabecera de esa compra (renglones en solo lectura). */
  readonly compra = input<CompraDetalle | null>(null);
  readonly guardado = output<CompraDetalle>();
  readonly cancelado = output<void>();
  /** Edición sin cambios en el servidor (misma versión). */
  readonly sinCambios = output<void>();
  /** Pidió recargar la compra tras un 409 version_desactualizada. */
  readonly recargar = output<void>();

  private readonly api = inject(ComprasApi);
  private readonly toast = inject(ToastService);

  protected readonly iconoQuitar = Trash2;
  protected readonly hoy = hoyISO();
  protected fecha = hoyISO();
  protected empresaId = '';
  protected notas = '';
  protected motivo = '';
  protected readonly esEdicion = computed(() => this.compra() !== null);
  protected readonly desactualizada = signal(false);
  protected readonly renglones = signal<Renglon[]>([]);
  protected readonly errores = signal<Record<string, string>>({});
  protected readonly guardando = signal(false);
  /** Se incrementa para recalcular totales cuando cambia un campo editado en el lugar. */
  private readonly version = signal(0);
  private siguienteClave = 1;

  constructor() {
    effect(() => {
      const c = this.compra();
      if (c) {
        const base = cabeceraDe(c);
        this.fecha = base.fecha;
        this.empresaId = base.empresaId;
        this.notas = base.notas;
        this.motivo = '';
        this.errores.set({});
        this.desactualizada.set(false);
      }
    });
  }

  protected readonly total = computed(() => {
    this.version();
    return deCentavos(this.renglones().reduce((t, r) => t + aCentavos(r.costo) * (Number(r.cantidad) || 0), 0));
  });
  protected readonly unidades = computed(() => {
    this.version();
    return this.renglones().reduce((t, r) => t + (Number(r.cantidad) || 0), 0);
  });

  protected agregar(p: ProductoResumen): void {
    this.renglones.update((rs) => [...rs, {
      clave: this.siguienteClave++, productoId: p.id, nombre: p.nombre, sku: p.sku, precioActual: p.precioVenta,
      cantidad: 1, costo: p.costo, precioNuevo: null, vencimiento: '',
    }]);
    if (p.empresaId && !this.empresaId) {
      this.empresaId = p.empresaId;
    }
  }

  protected quitar(r: Renglon): void {
    this.renglones.update((rs) => rs.filter((x) => x.clave !== r.clave));
  }

  protected refrescar(): void {
    this.version.update((v) => v + 1);
  }

  protected subtotal(r: Renglon): string {
    return deCentavos(aCentavos(r.costo) * (Number(r.cantidad) || 0));
  }

  protected errorRenglon(i: number, campo: string): string | null {
    return this.errores()[`items[${i}].${campo}`] ?? null;
  }

  protected async guardarEdicion(): Promise<void> {
    const original = this.compra();
    if (!original) {
      return;
    }
    const actual = { fecha: this.fecha, empresaId: this.empresaId, notas: this.notas, motivo: this.motivo };
    const locales: Record<string, string> = {};
    if (!actual.fecha) {
      locales['fecha'] = 'Ingresá la fecha';
    } else if (actual.fecha > hoyISO()) {
      locales['fecha'] = 'La fecha no puede ser futura';
    }
    if (actual.notas.length > 500) {
      locales['notas'] = 'Máximo 500 caracteres';
    }
    if (actual.motivo.length > 255) {
      locales['motivo'] = 'Máximo 255 caracteres';
    }
    this.errores.set(locales);
    if (Object.keys(locales).length) {
      return;
    }
    if (!hayCambios(original, actual)) {
      this.sinCambios.emit();
      return;
    }
    this.guardando.set(true);
    try {
      const compra = await firstValueFrom(this.api.editar(original.id, armarEdicion(original, actual, origenDispositivo())));
      if (seAplicoEdicion(original.version, compra)) {
        this.guardado.emit(compra);
      } else {
        this.sinCambios.emit();
      }
    } catch (e) {
      const err = ApiError.desde(e);
      if (err.codigo === 'version_desactualizada') {
        this.desactualizada.set(true);
      } else if (err.codigo === 'compra_no_encontrada') {
        this.toast.error(new ApiError(404, err.codigo, 'La compra ya no existe'));
        this.cancelado.emit();
      } else if (err.codigo === 'empresa_no_encontrada') {
        this.errores.set({ empresaId: 'La empresa elegida ya no existe' });
      } else if (err.status === 422 && Object.keys(err.campos).length) {
        this.errores.set({ ...err.campos });
      } else {
        this.toast.error(err);
      }
    } finally {
      this.guardando.set(false);
    }
  }

  protected async guardar(): Promise<void> {
    const locales: Record<string, string> = {};
    this.renglones().forEach((r, i) => {
      if (!Number.isInteger(Number(r.cantidad)) || Number(r.cantidad) < 1) {
        locales[`items[${i}].cantidad`] = 'Mayor a cero';
      }
      if (r.costo === null) {
        locales[`items[${i}].costoUnitario`] = 'Ingresá el costo';
      }
    });
    this.errores.set(locales);
    if (Object.keys(locales).length) {
      return;
    }
    this.guardando.set(true);
    try {
      const compra = await firstValueFrom(this.api.registrar({
        // Si es de hoy, sin fecha: el servidor guarda la hora exacta.
        fecha: this.fecha === hoyISO() ? '' : this.fecha, empresaId: this.empresaId, notas: this.notas,
        items: this.renglones().map((r) => ({
          productoId: r.productoId, cantidad: Number(r.cantidad), costoUnitario: r.costo!,
          precioVentaNuevo: r.precioNuevo, vencimiento: r.vencimiento || null,
        })),
      }));
      this.guardado.emit(compra);
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
}
