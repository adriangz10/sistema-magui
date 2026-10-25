import { ChangeDetectionStrategy, Component, OnInit, computed, inject, input, output, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { LucideAngularModule, Minus, Plus, Trash2 } from 'lucide-angular';
import { ApiError } from '../../core/api/api-error';
import { CatalogoApi, CategoriaNodo, Empresa, ProductoDetalle, ProductoGuardar, ProductoResumen, TipoProducto } from '../../core/api/catalogo.api';
import { ToastService } from '../../core/ui/toast.service';
import { aplanar } from '../../shared/catalogo/categorias';
import { aCentavos, DineroPipe, deCentavos } from '../../shared/formato/dinero';
import { BuscadorProductoComponent } from '../../shared/ui/buscador-producto.component';
import { CampoComponent } from '../../shared/ui/campo.component';
import { InputDineroComponent } from '../../shared/ui/input-dinero.component';

interface Componente {
  productoId: string;
  nombre: string;
  sku: string;
  precio: string;
  cantidad: number;
}

/** Alta y edición de productos simples y combos. */
@Component({
  selector: 'app-producto-form',
  imports: [ReactiveFormsModule, LucideAngularModule, DineroPipe, BuscadorProductoComponent, CampoComponent, InputDineroComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form [formGroup]="form" (ngSubmit)="guardar()" novalidate class="space-y-5" id="form-producto">
      <div class="grid gap-4 sm:grid-cols-2">
        <app-campo class="sm:col-span-2" etiqueta="Nombre" para="p-nombre" [error]="error('nombre')">
          <input id="p-nombre" class="input" formControlName="nombre" maxlength="255" autocomplete="off"
                 [attr.aria-invalid]="error('nombre') ? true : null" [attr.aria-describedby]="error('nombre') ? 'p-nombre-error' : null" />
        </app-campo>
        <app-campo etiqueta="SKU" para="p-sku" [opcional]="true" [error]="error('sku')" ayuda="Si lo dejás vacío se genera uno.">
          <input id="p-sku" class="input" formControlName="sku" maxlength="40" autocomplete="off"
                 [attr.aria-invalid]="error('sku') ? true : null" [attr.aria-describedby]="error('sku') ? 'p-sku-error' : 'p-sku-ayuda'" />
        </app-campo>
        <app-campo etiqueta="Código de barras" para="p-codigo" [opcional]="true" [error]="error('codigoBarras')" ayuda="Podés escanearlo con el lector.">
          <input id="p-codigo" class="input" formControlName="codigoBarras" maxlength="64" autocomplete="off" (keydown.enter)="$event.preventDefault()"
                 [attr.aria-invalid]="error('codigoBarras') ? true : null" [attr.aria-describedby]="error('codigoBarras') ? 'p-codigo-error' : 'p-codigo-ayuda'" />
        </app-campo>
        <app-campo etiqueta="Empresa" para="p-empresa" [opcional]="true" [error]="error('empresaId')">
          <select id="p-empresa" class="input" formControlName="empresaId">
            <option value="">Sin empresa</option>
            @for (e of empresas(); track e.id) { <option [value]="e.id">{{ e.nombre }}{{ e.activo ? '' : ' (inactiva)' }}</option> }
          </select>
        </app-campo>
        @if (tipo() === 'SIMPLE') {
          <app-campo etiqueta="Stock mínimo" para="p-minimo" [error]="error('stockMinimo')" ayuda="Avisa cuando el stock llega a este número.">
            <input id="p-minimo" type="number" min="0" inputmode="numeric" class="input" formControlName="stockMinimo"
                   [attr.aria-describedby]="error('stockMinimo') ? 'p-minimo-error' : 'p-minimo-ayuda'" />
          </app-campo>
        }
      </div>

      @if (tipo() === 'COMBO') {
        <fieldset class="rounded-2xl border border-ink-200 p-4 dark:border-ink-700">
          <legend class="px-1 text-sm font-medium">Productos del combo</legend>
          <app-buscador-producto idCampo="combo-buscar" etiqueta="Agregar producto al combo" [compacto]="true" tipo="SIMPLE"
                                 placeholder="Buscá o escaneá un producto para agregarlo" (elegido)="agregarComponente($event)" />
          @if (componentes().length) {
            <ul class="mt-3 divide-y divide-ink-100 dark:divide-ink-800">
              @for (c of componentes(); track c.productoId) {
                <li class="flex items-center gap-3 py-2">
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-sm font-medium">{{ c.nombre }}</p>
                    <p class="text-xs text-ink-500">{{ c.sku }} · {{ c.precio | dinero }} c/u</p>
                  </div>
                  <div class="flex items-center gap-1">
                    <button type="button" class="btn-secundario btn-icono min-h-8 w-8" (click)="cambiarCantidad(c, -1)" [attr.aria-label]="'Una unidad menos de ' + c.nombre">
                      <lucide-icon [img]="iconos.menos" [size]="14" aria-hidden="true" />
                    </button>
                    <span class="w-8 text-center tabular-nums" [attr.aria-label]="c.cantidad + ' unidades'">{{ c.cantidad }}</span>
                    <button type="button" class="btn-secundario btn-icono min-h-8 w-8" (click)="cambiarCantidad(c, 1)" [attr.aria-label]="'Una unidad más de ' + c.nombre">
                      <lucide-icon [img]="iconos.mas" [size]="14" aria-hidden="true" />
                    </button>
                  </div>
                  <button type="button" class="btn-fantasma btn-icono min-h-8 w-8 text-ink-400 hover:text-danger" (click)="quitarComponente(c)" [attr.aria-label]="'Quitar ' + c.nombre">
                    <lucide-icon [img]="iconos.quitar" [size]="16" aria-hidden="true" />
                  </button>
                </li>
              }
            </ul>
            <p class="mt-2 text-sm text-ink-500">
              Por separado: <strong class="tabular-nums">{{ precioSeparado() | dinero }}</strong>
              @if (ahorro() > 0) { · el cliente ahorra <strong class="text-ok tabular-nums dark:text-green-400">{{ ahorroTexto() | dinero }}</strong> }
            </p>
          } @else {
            <p class="mt-3 text-sm text-ink-500">Agregá al menos 2 unidades en total (pueden ser del mismo producto).</p>
          }
          @if (error('items')) { <p class="mt-2 text-sm text-danger dark:text-red-400" role="alert">{{ error('items') }}</p> }
        </fieldset>
      }

      <div class="grid gap-4 sm:grid-cols-2">
        <app-campo etiqueta="Precio de venta" para="p-precio" [error]="error('precioVenta')">
          <app-input-dinero idCampo="p-precio" formControlName="precioVenta" [invalido]="!!error('precioVenta')"
                            [describedBy]="error('precioVenta') ? 'p-precio-error' : ''" />
        </app-campo>
        @if (tipo() === 'SIMPLE') {
          <app-campo etiqueta="Costo" para="p-costo" [opcional]="true" [error]="error('costo')" [ayuda]="ayudaCosto()">
            <app-input-dinero idCampo="p-costo" formControlName="costo" [describedBy]="error('costo') ? 'p-costo-error' : 'p-costo-ayuda'" />
          </app-campo>
        }
      </div>

      <fieldset>
        <legend class="etiqueta">Categorías <span class="font-normal text-ink-400">(opcional)</span></legend>
        <div class="max-h-48 overflow-y-auto rounded-xl border border-ink-200 p-2 dark:border-ink-700">
          @for (c of categoriasPlanas(); track c.id) {
            <label class="flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1.5 text-sm hover:bg-ink-50 dark:hover:bg-ink-800" [style.padding-left.rem]="0.5 + c.nivel * 1.25">
              <input type="checkbox" class="size-4 rounded accent-brand-600" [checked]="seleccionadas().has(c.id)" (change)="alternarCategoria(c.id)" />
              {{ c.nombre }}
            </label>
          } @empty {
            <p class="p-2 text-sm text-ink-500">Todavía no hay categorías. Creálas en la sección Categorías.</p>
          }
        </div>
      </fieldset>

      <app-campo etiqueta="Descripción" para="p-desc" [opcional]="true">
        <textarea id="p-desc" class="input min-h-24 py-2" formControlName="descripcion"></textarea>
      </app-campo>

      <label class="inline-flex items-center gap-2 text-sm">
        <input type="checkbox" class="size-4 rounded accent-brand-600" formControlName="mostrarEnTienda" /> Mostrar en tienda online
      </label>

      <div class="flex justify-end gap-2 border-t border-ink-100 pt-4 dark:border-ink-800">
        <button type="button" class="btn-secundario" (click)="cancelado.emit()">Cancelar</button>
        <button type="submit" class="btn-primario" [disabled]="guardando()">{{ producto() ? 'Guardar cambios' : 'Crear' }}</button>
      </div>
    </form>
  `,
})
export class ProductoFormComponent implements OnInit {
  readonly producto = input<ProductoDetalle | null>(null);
  readonly tipo = input<TipoProducto>('SIMPLE');
  readonly categorias = input<CategoriaNodo[]>([]);
  readonly empresas = input<Empresa[]>([]);
  readonly guardado = output<ProductoDetalle>();
  readonly cancelado = output<void>();

  private readonly api = inject(CatalogoApi);
  private readonly toast = inject(ToastService);

  protected readonly iconos = { mas: Plus, menos: Minus, quitar: Trash2 };
  protected readonly form = inject(FormBuilder).nonNullable.group({
    nombre: ['', Validators.required],
    sku: [''],
    codigoBarras: [''],
    empresaId: [''],
    precioVenta: inject(FormBuilder).control<string | null>(null),
    costo: inject(FormBuilder).control<string | null>(null),
    stockMinimo: [0],
    descripcion: [''],
    mostrarEnTienda: [true],
  });
  private readonly valores = toSignal(this.form.valueChanges, { initialValue: this.form.getRawValue() });

  protected readonly categoriasPlanas = computed(() => aplanar(this.categorias()));
  protected readonly seleccionadas = signal(new Set<string>());
  protected readonly componentes = signal<Componente[]>([]);
  protected readonly errores = signal<Record<string, string>>({});
  protected readonly guardando = signal(false);

  protected readonly precioSeparado = computed(() =>
    deCentavos(this.componentes().reduce((t, c) => t + aCentavos(c.precio) * c.cantidad, 0)));
  protected readonly ahorro = computed(() => aCentavos(this.precioSeparado()) - aCentavos(this.valores().precioVenta ?? null));
  protected readonly ahorroTexto = computed(() => deCentavos(this.ahorro()));
  protected readonly ayudaCosto = computed(() => {
    const precio = aCentavos(this.valores().precioVenta ?? null);
    const costo = aCentavos(this.valores().costo ?? null);
    if (!precio || !costo) {
      return 'Se actualiza solo con cada compra.';
    }
    return `Margen: ${Math.round(((precio - costo) / precio) * 100)} % · ganancia ${new Intl.NumberFormat('es-AR', { style: 'currency', currency: 'ARS' }).format((precio - costo) / 100)} por unidad`;
  });

  ngOnInit(): void {
    const p = this.producto();
    if (!p) {
      return;
    }
    this.form.reset({
      nombre: p.nombre, sku: p.sku, codigoBarras: p.codigoBarras ?? '', empresaId: p.empresaId ?? '',
      precioVenta: p.precioVenta, costo: p.costo, stockMinimo: p.stockMinimo, descripcion: p.descripcion ?? '',
      mostrarEnTienda: p.mostrarEnTienda,
    });
    this.seleccionadas.set(new Set(p.categorias.map((c) => c.id)));
    this.componentes.set(p.items.map((i) => ({ productoId: i.productoId, nombre: i.nombre, sku: i.sku, precio: i.precioVenta, cantidad: i.cantidad })));
  }

  protected error(campo: string): string | null {
    return this.errores()[campo] ?? null;
  }

  protected alternarCategoria(id: string): void {
    this.seleccionadas.update((s) => {
      const n = new Set(s);
      if (n.has(id)) {
        n.delete(id);
      } else {
        n.add(id);
      }
      return n;
    });
  }

  protected agregarComponente(p: ProductoResumen): void {
    if (p.id === this.producto()?.id) {
      return;
    }
    this.componentes.update((cs) => {
      const i = cs.findIndex((c) => c.productoId === p.id);
      if (i >= 0) {
        return cs.map((c, j) => (j === i ? { ...c, cantidad: c.cantidad + 1 } : c));
      }
      return [...cs, { productoId: p.id, nombre: p.nombre, sku: p.sku, precio: p.precioVenta, cantidad: 1 }];
    });
  }

  protected cambiarCantidad(c: Componente, delta: number): void {
    const n = c.cantidad + delta;
    if (n <= 0) {
      this.quitarComponente(c);
      return;
    }
    this.componentes.update((cs) => cs.map((x) => (x.productoId === c.productoId ? { ...x, cantidad: n } : x)));
  }

  protected quitarComponente(c: Componente): void {
    this.componentes.update((cs) => cs.filter((x) => x.productoId !== c.productoId));
  }

  protected async guardar(): Promise<void> {
    const v = this.form.getRawValue();
    const locales: Record<string, string> = {};
    if (!v.nombre.trim()) {
      locales['nombre'] = 'El nombre es obligatorio';
    }
    if (!v.precioVenta || aCentavos(v.precioVenta) <= 0) {
      locales['precioVenta'] = 'Ingresá un precio mayor a cero';
    }
    if (this.tipo() === 'COMBO' && this.componentes().reduce((t, c) => t + c.cantidad, 0) < 2) {
      locales['items'] = 'El combo necesita al menos 2 unidades en total';
    }
    this.errores.set(locales);
    if (Object.keys(locales).length) {
      return;
    }

    const datos: ProductoGuardar = {
      tipo: this.tipo(), sku: v.sku.trim(), codigoBarras: v.codigoBarras.trim(), nombre: v.nombre, descripcion: v.descripcion,
      empresaId: v.empresaId, precioVenta: v.precioVenta!, costo: this.tipo() === 'SIMPLE' ? v.costo : null,
      stockMinimo: Number(v.stockMinimo) || 0, mostrarEnTienda: v.mostrarEnTienda, categoriaIds: [...this.seleccionadas()],
      items: this.tipo() === 'COMBO' ? this.componentes().map((c) => ({ productoId: c.productoId, cantidad: c.cantidad })) : [],
    };
    const actual = this.producto();
    this.guardando.set(true);
    try {
      const res = actual
        ? await firstValueFrom(this.api.actualizarProducto(actual.id, { ...datos, version: actual.version }))
        : await firstValueFrom(this.api.crearProducto(datos));
      this.guardado.emit(res);
    } catch (e) {
      const err = ApiError.desde(e);
      const campo = { sku_duplicado: 'sku', codigo_barras_duplicado: 'codigoBarras', combo_invalido: 'items' }[err.codigo];
      if (Object.keys(err.campos).length) {
        this.errores.set({ ...err.campos });
      } else if (campo) {
        this.errores.set({ [campo]: err.message });
      } else {
        this.toast.error(err);
      }
    } finally {
      this.guardando.set(false);
    }
  }
}
