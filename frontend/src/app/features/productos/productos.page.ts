import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { LucideAngularModule, Plus, Search } from 'lucide-angular';
import { CatalogoApi, CategoriaNodo, ConsultaProductos, Empresa, ProductoDetalle, ProductoResumen } from '../../core/api/catalogo.api';
import { Pagina } from '../../core/api/modelos';
import { ConfirmacionService } from '../../core/ui/confirmacion.service';
import { ToastService } from '../../core/ui/toast.service';
import { aplanar, sangria } from '../../shared/catalogo/categorias';
import { aCentavos, DineroPipe } from '../../shared/formato/dinero';
import { CampoComponent } from '../../shared/ui/campo.component';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';
import { ModalComponent } from '../../shared/ui/modal.component';
import { TablaComponent } from '../../shared/ui/tabla.component';
import { HistorialPreciosComponent } from './historial-precios.component';
import { ProductoFormComponent } from './producto-form.component';

type Filtro = Required<Pick<ConsultaProductos, 'q' | 'categoriaId' | 'empresaId' | 'tipo' | 'estado' | 'orden' | 'page'>> & { stockBajo: boolean };

const FILTRO_INICIAL: Filtro = { q: '', categoriaId: '', empresaId: '', tipo: '', estado: 'activos', orden: 'nombre', stockBajo: false, page: 1 };

/** Catálogo de productos y combos. */
@Component({
  selector: 'app-productos',
  imports: [FormsModule, LucideAngularModule, DineroPipe, CampoComponent, EncabezadoComponent, ModalComponent, TablaComponent,
    ProductoFormComponent, HistorialPreciosComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-encabezado titulo="Productos" descripcion="Productos y combos, con su precio, costo y stock">
      <button type="button" class="btn-secundario" (click)="nuevo('COMBO')">
        <lucide-icon [img]="iconos.mas" [size]="18" aria-hidden="true" /> Nuevo combo
      </button>
      <button type="button" class="btn-primario" (click)="nuevo('SIMPLE')">
        <lucide-icon [img]="iconos.mas" [size]="18" aria-hidden="true" /> Nuevo producto
      </button>
    </app-encabezado>

    <div class="card mb-4 space-y-3 p-4">
      <div class="relative">
        <lucide-icon [img]="iconos.buscar" [size]="18" class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-ink-400" aria-hidden="true" />
        <label for="f-q" class="sr-only">Buscar productos</label>
        <input id="f-q" type="search" class="input pl-10" placeholder="Buscar por nombre, SKU o código de barras"
               [ngModel]="filtro().q" (ngModelChange)="buscar($event)" />
      </div>
      <button type="button" class="btn-secundario w-full lg:hidden" (click)="filtrosAbiertos.set(!filtrosAbiertos())"
              [attr.aria-expanded]="filtrosAbiertos()" aria-controls="filtros-productos">
        {{ filtrosAbiertos() ? 'Ocultar filtros' : 'Filtros' }}{{ filtrosActivos() ? ' (' + filtrosActivos() + ')' : '' }}
      </button>
      <div id="filtros-productos" class="gap-3 sm:grid-cols-2 lg:grid lg:grid-cols-5" [class]="filtrosAbiertos() ? 'grid' : 'hidden'">
        <app-campo etiqueta="Categoría" para="f-cat">
          <select id="f-cat" class="input" [ngModel]="filtro().categoriaId" (ngModelChange)="filtrar({ categoriaId: $event })">
            <option value="">Todas</option>
            @for (c of categorias(); track c.id) { <option [value]="c.id">{{ sangria(c.nivel) }}{{ c.nombre }}</option> }
          </select>
        </app-campo>
        <app-campo etiqueta="Empresa" para="f-emp">
          <select id="f-emp" class="input" [ngModel]="filtro().empresaId" (ngModelChange)="filtrar({ empresaId: $event })">
            <option value="">Todas</option>
            @for (e of empresas(); track e.id) { <option [value]="e.id">{{ e.nombre }}</option> }
          </select>
        </app-campo>
        <app-campo etiqueta="Tipo" para="f-tipo">
          <select id="f-tipo" class="input" [ngModel]="filtro().tipo" (ngModelChange)="filtrar({ tipo: $event })">
            <option value="">Todos</option><option value="SIMPLE">Productos</option><option value="COMBO">Combos</option>
          </select>
        </app-campo>
        <app-campo etiqueta="Estado" para="f-estado">
          <select id="f-estado" class="input" [ngModel]="filtro().estado" (ngModelChange)="filtrar({ estado: $event })">
            <option value="activos">Activos</option><option value="inactivos">Desactivados</option><option value="todos">Todos</option>
          </select>
        </app-campo>
        <app-campo etiqueta="Ordenar por" para="f-orden">
          <select id="f-orden" class="input" [ngModel]="filtro().orden" (ngModelChange)="filtrar({ orden: $event })">
            <option value="nombre">Nombre</option><option value="recientes">Más nuevos</option>
            <option value="precio">Precio</option><option value="stock">Menos stock</option>
          </select>
        </app-campo>
      </div>
      <label class="items-center gap-2 text-sm lg:inline-flex" [class]="filtrosAbiertos() ? 'inline-flex' : 'hidden'">
        <input type="checkbox" class="size-4 rounded accent-brand-600" [ngModel]="filtro().stockBajo" (ngModelChange)="filtrar({ stockBajo: $event })" />
        Solo stock bajo (en o por debajo del mínimo)
      </label>
    </div>

    <app-tabla [pagina]="pagina()" [cargando]="cargando()" mensajeVacio="No hay productos con estos filtros"
               (paginaCambio)="filtrar({ page: $event }, false)">
      <table class="tabla">
        <caption class="sr-only">Productos</caption>
        <thead>
          <tr>
            <th scope="col">Producto</th><th scope="col">Empresa</th><th scope="col" class="text-right">Precio</th>
            <th scope="col" class="text-right">Costo</th><th scope="col" class="text-right">Margen</th>
            <th scope="col" class="text-right">Stock</th><th scope="col"><span class="sr-only">Acciones</span></th>
          </tr>
        </thead>
        <tbody>
          @for (p of pagina()?.items; track p.id) {
            <tr [class.opacity-60]="!p.activo">
              <td class="min-w-72">
                <p class="font-medium">
                  {{ p.nombre }}
                  @if (p.tipo === 'COMBO') { <span class="badge-neutro ml-1">Combo</span> }
                  @if (!p.activo) { <span class="badge-danger ml-1">Desactivado</span> }
                </p>
                <p class="text-xs text-ink-500">{{ p.sku }}{{ p.codigoBarras ? ' · ' + p.codigoBarras : '' }}</p>
              </td>
              <td class="text-ink-500">{{ p.empresa ?? '—' }}</td>
              <td class="text-right font-medium tabular-nums">{{ p.precioVenta | dinero }}</td>
              <td class="text-right text-ink-500 tabular-nums">{{ p.costo ? (p.costo | dinero) : '—' }}</td>
              <td class="text-right tabular-nums">{{ margen(p) }}</td>
              <td class="text-right">
                <span [class]="claseStock(p)">{{ p.stock }}</span>
              </td>
              <td class="text-right whitespace-nowrap">
                <button type="button" class="btn-fantasma min-h-8 px-3" (click)="editar(p)" [attr.aria-label]="'Editar ' + p.nombre">Editar</button>
                <button type="button" class="btn-fantasma min-h-8 px-3" (click)="historial.set(p)" [attr.aria-label]="'Historial de precios de ' + p.nombre">Precios</button>
                <button type="button" class="btn-fantasma min-h-8 px-3" (click)="alternarEstado(p)"
                        [attr.aria-label]="(p.activo ? 'Desactivar ' : 'Activar ') + p.nombre">{{ p.activo ? 'Desactivar' : 'Activar' }}</button>
              </td>
            </tr>
          }
        </tbody>
      </table>
    </app-tabla>

    <app-modal [abierto]="!!formulario()" [titulo]="tituloFormulario()" ancho="grande" (cerrar)="formulario.set(null)">
      @if (formulario(); as f) {
        <app-producto-form [producto]="f.producto" [tipo]="f.tipo" [categorias]="arbol()" [empresas]="empresas()"
                           (guardado)="guardado($event)" (cancelado)="formulario.set(null)" />
      }
    </app-modal>

    <app-modal [abierto]="!!historial()" [titulo]="'Historial de precios'" (cerrar)="historial.set(null)">
      @if (historial(); as h) { <app-historial-precios [producto]="h" /> }
    </app-modal>
  `,
})
export class ProductosPage {
  private readonly api = inject(CatalogoApi);
  private readonly toast = inject(ToastService);
  private readonly confirmacion = inject(ConfirmacionService);

  protected readonly iconos = { mas: Plus, buscar: Search };
  protected readonly sangria = sangria;
  protected readonly filtro = signal<Filtro>({ ...FILTRO_INICIAL });
  protected readonly pagina = signal<Pagina<ProductoResumen> | null>(null);
  protected readonly cargando = signal(false);
  protected readonly arbol = signal<CategoriaNodo[]>([]);
  protected readonly categorias = computed(() => aplanar(this.arbol()));
  protected readonly empresas = signal<Empresa[]>([]);
  protected readonly formulario = signal<{ producto: ProductoDetalle | null; tipo: 'SIMPLE' | 'COMBO' } | null>(null);
  protected readonly historial = signal<ProductoResumen | null>(null);
  protected readonly filtrosAbiertos = signal(false);
  protected readonly filtrosActivos = computed(() => {
    const f = this.filtro();
    return [f.categoriaId, f.empresaId, f.tipo, f.estado !== 'activos', f.orden !== 'nombre', f.stockBajo].filter(Boolean).length;
  });
  protected readonly tituloFormulario = computed(() => {
    const f = this.formulario();
    const que = f?.tipo === 'COMBO' ? 'combo' : 'producto';
    return f?.producto ? `Editar ${que}` : `Nuevo ${que}`;
  });

  private temporizador: ReturnType<typeof setTimeout> | undefined;

  constructor() {
    void this.cargar();
    void this.cargarListas();
  }

  private async cargarListas(): Promise<void> {
    try {
      const [arbol, empresas] = await Promise.all([firstValueFrom(this.api.categorias()), this.api.todasLasEmpresas()]);
      this.arbol.set(arbol);
      this.empresas.set(empresas);
    } catch (e) {
      this.toast.error(e);
    }
  }

  protected buscar(q: string): void {
    clearTimeout(this.temporizador);
    this.temporizador = setTimeout(() => this.filtrar({ q }), 300);
  }

  protected filtrar(cambio: Partial<Filtro>, reiniciarPagina = true): void {
    this.filtro.update((f) => ({ ...f, ...cambio, page: reiniciarPagina ? 1 : (cambio.page ?? f.page) }));
    void this.cargar();
  }

  private async cargar(): Promise<void> {
    this.cargando.set(true);
    try {
      this.pagina.set(await firstValueFrom(this.api.listarProductos(this.filtro())));
    } catch (e) {
      this.toast.error(e);
    } finally {
      this.cargando.set(false);
    }
  }

  protected margen(p: ProductoResumen): string {
    if (!p.costo) {
      return '—';
    }
    const precio = aCentavos(p.precioVenta);
    return precio ? `${Math.round(((precio - aCentavos(p.costo)) / precio) * 100)} %` : '—';
  }

  protected claseStock(p: ProductoResumen): string {
    if (p.stock <= 0) {
      return 'badge-danger';
    }
    return p.tipo === 'SIMPLE' && p.stock <= p.stockMinimo ? 'badge-warn' : 'badge-neutro';
  }

  protected nuevo(tipo: 'SIMPLE' | 'COMBO'): void {
    this.formulario.set({ producto: null, tipo });
  }

  protected async editar(p: ProductoResumen): Promise<void> {
    try {
      const detalle = await firstValueFrom(this.api.obtenerProducto(p.id));
      this.formulario.set({ producto: detalle, tipo: detalle.tipo });
    } catch (e) {
      this.toast.error(e);
    }
  }

  protected guardado(p: ProductoDetalle): void {
    const nuevo = !this.formulario()?.producto;
    this.formulario.set(null);
    this.toast.exito(nuevo ? `${p.nombre} creado. Cargá su stock desde Compras o Inventario.` : `${p.nombre} actualizado`);
    void this.cargar();
  }

  protected async alternarEstado(p: ProductoResumen): Promise<void> {
    if (p.activo) {
      const ok = await this.confirmacion.preguntar({
        titulo: 'Desactivar producto',
        mensaje: `${p.nombre} dejará de aparecer en Vender. Sus ventas anteriores se conservan y podés volver a activarlo.`,
        confirmar: 'Desactivar', peligro: true,
      });
      if (!ok) {
        return;
      }
    }
    try {
      await firstValueFrom(this.api.cambiarEstado(p.id, !p.activo));
      this.toast.exito(p.activo ? `${p.nombre} desactivado` : `${p.nombre} activado`);
      void this.cargar();
    } catch (e) {
      this.toast.error(e);
    }
  }
}
