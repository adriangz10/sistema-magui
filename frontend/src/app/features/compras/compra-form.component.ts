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
import { hoyISO } from '../../shared/formato/fecha';
import { BuscadorProductoComponent } from '../../shared/ui/buscador-producto.component';
import { CampoComponent } from '../../shared/ui/campo.component';
import { InputDineroComponent } from '../../shared/ui/input-dinero.component';
import {
  armarEdicion, armarItems, cabeceraDe, hayCambios, hayCambiosRenglones, mapearErroresServidor, minimoCantidad, motivoNoQuitar,
  RenglonEditable, renglonesDe, renglonNuevo, resumenEfectos, seAplicoEdicion, totalRenglones, validarRenglones,
} from './edicion-compra';

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
  imports: [FormsModule, LucideAngularModule, DineroPipe, BuscadorProductoComponent, CampoComponent, InputDineroComponent],
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

        @if (concurrente()) {
          <div class="aviso-warn" role="alert">Otra operación se cruzó, intentá de nuevo.</div>
        }

        <app-buscador-producto idCampo="compra-buscar-ed" etiqueta="Agregar producto a la compra" [compacto]="true" tipo="SIMPLE"
                               placeholder="Escaneá o buscá un producto para agregarlo" (elegido)="agregarEd($event)" />

        <ul class="space-y-3" aria-label="Renglones de la compra">
          @for (r of renglonesEd(); track r.id; let i = $index) {
            <li class="rounded-xl border border-ink-200 p-4 dark:border-ink-700" [class.border-dashed]="r.quitado">
              <div class="mb-3 flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <p class="font-medium" [class.line-through]="r.quitado">{{ r.producto }}
                    @if (r.nuevo) { <span class="badge-ok">Nuevo</span> }
                    @if (r.quitado) { <span class="badge-danger">Se quitará</span> }
                  </p>
                  @if (r.nuevo) {
                    <p class="text-xs text-muted">{{ r.sku }}@if (r.precioActual) { · precio actual {{ r.precioActual | dinero }} }</p>
                  } @else {
                    <p class="text-xs text-muted">{{ r.sku }} · ya salieron {{ r.consumido }} de {{ r.cantidadOriginal }}</p>
                  }
                </div>
                <button type="button" [class]="r.quitado ? 'btn-secundario' : 'btn-fantasma btn-icono -mt-1 -mr-2 text-muted hover:text-danger'"
                        (click)="alternarQuitar(r)" [attr.aria-label]="(r.quitado ? 'Deshacer quitar ' : 'Quitar ') + r.producto"
                        [attr.aria-disabled]="!r.quitado && motivoNoQuitar(r) ? 'true' : null"
                        [attr.aria-describedby]="!r.quitado && motivoNoQuitar(r) ? 'e-motivo-' + r.id : null">
                  @if (r.quitado) { Deshacer } @else { <lucide-icon [img]="iconoQuitar" [size]="18" aria-hidden="true" /> }
                </button>
              </div>
              @if (!r.quitado && motivoNoQuitar(r); as motivoQ) {
                <p class="mb-3 text-xs text-muted" [id]="'e-motivo-' + r.id">{{ motivoQ }}</p>
              }
              @if (errores()['quitados[' + r.id + ']']; as errQ) {
                <p class="mb-3 text-sm text-danger" role="alert">{{ errQ }}</p>
              }
              @for (campo of ['id', 'productoId']; track campo) {
                @if (errorRenglon(i, campo); as errC) {
                  <p class="mb-3 text-sm text-danger" role="alert">{{ errC }}</p>
                }
              }
              @if (r.quitado) {
                <p class="text-sm text-muted">Se quitarán {{ r.cantidadOriginal }} unidades del stock al guardar.</p>
              } @else {
              <div class="grid gap-3 sm:grid-cols-4">
                <app-campo etiqueta="Cantidad" [para]="'e-cant-' + r.id" [error]="errorRenglon(i, 'cantidad')">
                  <input [id]="'e-cant-' + r.id" type="number" [min]="minimo(r)" inputmode="numeric" class="input" [(ngModel)]="r.cantidad" (ngModelChange)="refrescar()" />
                </app-campo>
                <app-campo etiqueta="Costo c/u" [para]="'e-costo-' + r.id" [error]="errorRenglon(i, 'costoUnitario')">
                  <app-input-dinero [idCampo]="'e-costo-' + r.id" [(ngModel)]="r.costo" (ngModelChange)="refrescar()" />
                </app-campo>
                <app-campo etiqueta="Nuevo precio" [para]="'e-precio-' + r.id" [opcional]="true" [error]="errorRenglon(i, 'precioVentaNuevo')">
                  <app-input-dinero [idCampo]="'e-precio-' + r.id" [(ngModel)]="r.precioNuevo" (ngModelChange)="refrescar()" placeholder="Sin cambio" />
                </app-campo>
                <app-campo etiqueta="Vencimiento" [para]="'e-venc-' + r.id" [opcional]="true" [error]="errorRenglon(i, 'vencimiento')">
                  <input [id]="'e-venc-' + r.id" type="date" class="input" [(ngModel)]="r.vencimiento" (ngModelChange)="refrescar()" />
                </app-campo>
              </div>
              <p class="mt-2 text-right text-sm text-muted">Subtotal <strong class="text-ink-900 tabular-nums dark:text-ink-100">{{ subtotal(r) | dinero }}</strong></p>
              }
            </li>
          }
        </ul>
        @if (errores()['items']) { <p class="text-sm text-danger" role="alert">{{ errores()['items'] }}</p> }
        @if (efectos().length) {
          <ul class="aviso-info list-disc pl-6" aria-label="Efectos de los cambios">
            @for (ef of efectos(); track ef) { <li>{{ ef }}</li> }
          </ul>
        }
        <div class="flex flex-wrap items-center justify-between gap-3 border-t border-ink-100 pt-4 dark:border-ink-800">
          <p class="text-lg">Total de la compra <strong class="tabular-nums">{{ totalEdicion() | dinero }}</strong></p>
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
  /** Renglones existentes en edición (US-001b). */
  protected readonly renglonesEd = signal<RenglonEditable[]>([]);
  protected readonly concurrente = signal(false);
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
        this.concurrente.set(false);
        this.renglonesEd.set(renglonesDe(c));
      }
    });
  }

  protected readonly total = computed(() => {
    this.version();
    return deCentavos(this.renglones().reduce((t, r) => t + aCentavos(r.costo) * (Number(r.cantidad) || 0), 0));
  });
  protected readonly totalEdicion = computed(() => {
    this.version();
    return totalRenglones(this.renglonesEd());
  });
  protected readonly efectos = computed(() => {
    this.version();
    return resumenEfectos(this.renglonesEd());
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

  /** Edición: agrega un renglón nuevo al final (se envía sin `id`). */
  protected agregarEd(p: ProductoResumen): void {
    this.renglonesEd.update((rs) => [...rs, renglonNuevo(this.siguienteClave++, p)]);
  }

  protected motivoNoQuitar(r: RenglonEditable): string | null {
    return motivoNoQuitar(r, this.renglonesEd());
  }

  /** Un renglón existente queda marcado "se quitará" (se puede deshacer); uno nuevo se descarta sin rastro. */
  protected quitarEd(r: RenglonEditable): void {
    if (this.motivoNoQuitar(r)) {
      return;
    }
    if (r.nuevo) {
      this.renglonesEd.update((rs) => rs.filter((x) => x.id !== r.id));
      // Las posiciones se corrieron: los errores por posición ya no valen.
      this.errores.update((e) => Object.fromEntries(Object.entries(e).filter(([k]) => !k.startsWith('items['))));
    } else {
      this.renglonesEd.update((rs) => rs.map((x) => (x.id === r.id ? { ...x, quitado: true } : x)));
    }
  }

  /** Un único botón: quita o deshace (así el foco no se pierde al alternar). */
  protected alternarQuitar(r: RenglonEditable): void {
    if (r.quitado) {
      this.deshacerQuitar(r);
    } else {
      this.quitarEd(r);
    }
  }

  protected deshacerQuitar(r: RenglonEditable): void {
    this.renglonesEd.update((rs) => rs.map((x) => (x.id === r.id ? { ...x, quitado: false } : x)));
    this.errores.update(({ [`quitados[${r.id}]`]: _, ...resto }) => resto);
  }

  protected minimo(r: RenglonEditable): number {
    return minimoCantidad(r.consumido);
  }

  protected subtotal(r: { costo: string | null; cantidad: number | string | null }): string {
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
    Object.assign(locales, validarRenglones(this.renglonesEd()));
    this.errores.set(locales);
    this.concurrente.set(false);
    if (Object.keys(locales).length) {
      return;
    }
    const cambioRenglones = hayCambiosRenglones(original, this.renglonesEd());
    if (!cambioRenglones && !hayCambios(original, actual)) {
      this.sinCambios.emit();
      return;
    }
    this.guardando.set(true);
    try {
      const compra = await firstValueFrom(this.api.editar(original.id, armarEdicion(original, actual, origenDispositivo(), cambioRenglones ? armarItems(this.renglonesEd()) : undefined)));
      if (seAplicoEdicion(original.version, compra)) {
        this.guardado.emit(compra);
      } else {
        this.sinCambios.emit();
      }
    } catch (e) {
      const err = ApiError.desde(e);
      if (err.codigo === 'version_desactualizada') {
        this.desactualizada.set(true);
      } else if (err.codigo === 'compra_consumida') {
        this.errores.set(mapearErroresServidor(this.renglonesEd(), err.campos));
        await this.refrescarConsumido(original.id);
      } else if (err.codigo === 'operacion_concurrente') {
        this.concurrente.set(true);
      } else if (err.codigo === 'compra_no_encontrada') {
        this.toast.error(new ApiError(404, err.codigo, 'La compra ya no existe'));
        this.cancelado.emit();
      } else if (err.codigo === 'empresa_no_encontrada') {
        this.errores.set({ empresaId: 'La empresa elegida ya no existe' });
      } else if (err.codigo === 'producto_no_comprable') {
        this.errores.set({ items: 'Un producto agregado es un combo: no se puede comprar, comprá sus productos' });
      } else if (err.codigo === 'producto_inactivo') {
        this.errores.set({ items: 'Un producto agregado está inactivo: quitalo o activalo en Productos' });
      } else if (err.codigo === 'producto_no_encontrado') {
        this.errores.set({ items: 'Un producto agregado ya no existe, quitalo e intentá de nuevo' });
      } else if (err.status === 422 && Object.keys(err.campos).length) {
        this.errores.set(mapearErroresServidor(this.renglonesEd(), err.campos));
      } else {
        this.toast.error(err);
      }
    } finally {
      this.guardando.set(false);
    }
  }

  /** Tras un 409 compra_consumida: trae el consumo actual sin pisar lo que se tipeó y revalida los mínimos. */
  private async refrescarConsumido(id: string): Promise<void> {
    try {
      const fresca = await firstValueFrom(this.api.obtener(id));
      this.renglonesEd.update((rs) => rs.map((r) => {
        const it = fresca.items.find((x) => x.id === r.id);
        if (!it) {
          return r;
        }
        const consumido = it.consumido ?? 0;
        // Si lo que se quería quitar ya tiene consumo, vuelve a quedar activo (el error queda en el renglón).
        return { ...r, consumido, quitado: consumido > 0 ? false : r.quitado };
      }));
      this.errores.update((e) => ({ ...validarRenglones(this.renglonesEd()), ...e }));
    } catch {
      // Se conservan los errores del servidor; el usuario puede recargar.
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
