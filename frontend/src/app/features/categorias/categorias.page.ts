import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ChevronRight, LucideAngularModule, Plus } from 'lucide-angular';
import { ApiError } from '../../core/api/api-error';
import { CatalogoApi, CategoriaNodo } from '../../core/api/catalogo.api';
import { ConfirmacionService } from '../../core/ui/confirmacion.service';
import { ToastService } from '../../core/ui/toast.service';
import { CategoriaPlana, aplanar, conDescendientes, sangria } from '../../shared/catalogo/categorias';
import { CampoComponent } from '../../shared/ui/campo.component';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';
import { ModalComponent } from '../../shared/ui/modal.component';
import { AjustePreciosComponent } from './ajuste-precios.component';
import { EmpresasComponent } from './empresas.component';

interface Edicion {
  categoria: CategoriaPlana | null; // null = nueva
  nombre: string;
  padreId: string;
}

/** Árbol de categorías, cambio de precios en cascada y empresas. */
@Component({
  selector: 'app-categorias',
  imports: [FormsModule, LucideAngularModule, CampoComponent, EncabezadoComponent, ModalComponent, AjustePreciosComponent, EmpresasComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-encabezado titulo="Categorías" descripcion="Organizá los productos y cambiá precios por categoría">
      <button type="button" class="btn-primario" (click)="nueva(null)">
        <lucide-icon [img]="iconos.mas" [size]="18" aria-hidden="true" /> Nueva categoría
      </button>
    </app-encabezado>

    <div class="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <section class="card" aria-label="Categorías">
        @if (cargando() && !planas().length) {
          <div class="space-y-3 p-4">@for (_ of [1, 2, 3, 4]; track $index) { <div class="h-10 animate-pulse rounded-lg bg-ink-100 dark:bg-ink-800"></div> }</div>
        } @else if (!planas().length) {
          <p class="px-6 py-14 text-center text-sm text-muted">Todavía no hay categorías.</p>
        } @else {
          <ul class="divide-y divide-ink-100 dark:divide-ink-800">
            @for (c of planas(); track c.id) {
              <li class="flex flex-wrap items-center gap-2 px-4 py-2.5" [style.padding-left.rem]="1 + c.nivel * 1.5">
                @if (c.nivel > 0) { <lucide-icon [img]="iconos.hijo" [size]="14" class="text-ink-300" aria-hidden="true" /> }
                <span class="min-w-0 flex-1" [class.font-semibold]="c.nivel === 0">
                  {{ c.nombre }} <span class="text-xs font-normal text-muted">· {{ c.productos }} {{ c.productos === 1 ? 'producto' : 'productos' }}</span>
                </span>
                <div class="flex flex-wrap gap-1">
                  <button type="button" class="btn-secundario min-h-8 px-3 text-xs" (click)="ajustando.set(c)" [attr.aria-label]="'Cambiar precios de ' + c.ruta">Cambiar precios</button>
                  <button type="button" class="btn-fantasma min-h-8 px-2.5 text-xs" (click)="nueva(c)" [attr.aria-label]="'Agregar subcategoría a ' + c.ruta">+ Sub</button>
                  <button type="button" class="btn-fantasma min-h-8 px-2.5 text-xs" (click)="editar(c)" [attr.aria-label]="'Editar ' + c.ruta">Editar</button>
                  <button type="button" class="btn-fantasma min-h-8 px-2.5 text-xs text-danger" (click)="eliminar(c)" [attr.aria-label]="'Eliminar ' + c.ruta">Eliminar</button>
                </div>
              </li>
            }
          </ul>
        }
      </section>
      <app-empresas />
    </div>

    <app-modal [abierto]="!!edicion()" [titulo]="edicion()?.categoria ? 'Editar categoría' : 'Nueva categoría'" (cerrar)="edicion.set(null)">
      @if (edicion(); as ed) {
        <form id="form-categoria" class="space-y-4" (ngSubmit)="guardar()">
          <app-campo etiqueta="Nombre" para="cat-nombre" [error]="errorEdicion()">
            <input id="cat-nombre" name="nombre" class="input" maxlength="100" [(ngModel)]="ed.nombre" autofocus
                   [attr.aria-invalid]="errorEdicion() ? true : null" [attr.aria-describedby]="errorEdicion() ? 'cat-nombre-error' : null" />
          </app-campo>
          <app-campo etiqueta="Dentro de" para="cat-padre">
            <select id="cat-padre" name="padre" class="input" [(ngModel)]="ed.padreId">
              <option value="">Ninguna (categoría principal)</option>
              @for (c of padresPosibles(); track c.id) { <option [value]="c.id">{{ sangria(c.nivel) }}{{ c.nombre }}</option> }
            </select>
          </app-campo>
        </form>
      }
      <div pie>
        <button type="button" class="btn-secundario" (click)="edicion.set(null)">Cancelar</button>
        <button type="submit" form="form-categoria" class="btn-primario">Guardar</button>
      </div>
    </app-modal>

    <app-modal [abierto]="!!ajustando()" titulo="Cambiar precios por categoría" ancho="grande" (cerrar)="ajustando.set(null)">
      @if (ajustando(); as c) { <app-ajuste-precios [categoria]="c" (cerrar)="finAjuste($event)" /> }
    </app-modal>
  `,
})
export class CategoriasPage {
  private readonly api = inject(CatalogoApi);
  private readonly toast = inject(ToastService);
  private readonly confirmacion = inject(ConfirmacionService);

  protected readonly iconos = { mas: Plus, hijo: ChevronRight };
  protected readonly sangria = sangria;
  protected readonly arbol = signal<CategoriaNodo[]>([]);
  protected readonly planas = computed(() => aplanar(this.arbol()));
  protected readonly cargando = signal(false);
  protected readonly edicion = signal<Edicion | null>(null);
  protected readonly errorEdicion = signal('');
  protected readonly ajustando = signal<CategoriaPlana | null>(null);

  /** Al mover una categoría, no puede quedar dentro de sí misma ni de sus subcategorías. */
  protected readonly padresPosibles = computed(() => {
    const ed = this.edicion();
    const excluir = ed?.categoria ? conDescendientes(this.arbol(), ed.categoria.id) : new Set<string>();
    return this.planas().filter((c) => !excluir.has(c.id));
  });

  constructor() {
    void this.cargar();
  }

  private async cargar(): Promise<void> {
    this.cargando.set(true);
    try {
      this.arbol.set(await firstValueFrom(this.api.categorias()));
    } catch (e) {
      this.toast.error(e);
    } finally {
      this.cargando.set(false);
    }
  }

  protected nueva(padre: CategoriaPlana | null): void {
    this.errorEdicion.set('');
    this.edicion.set({ categoria: null, nombre: '', padreId: padre?.id ?? '' });
  }

  protected editar(c: CategoriaPlana): void {
    this.errorEdicion.set('');
    this.edicion.set({ categoria: c, nombre: c.nombre, padreId: c.padreId ?? '' });
  }

  protected async guardar(): Promise<void> {
    const ed = this.edicion();
    if (!ed) {
      return;
    }
    try {
      const padre = ed.padreId || null;
      if (ed.categoria) {
        await firstValueFrom(this.api.actualizarCategoria(ed.categoria.id, ed.nombre, padre, ed.categoria.version));
      } else {
        await firstValueFrom(this.api.crearCategoria(ed.nombre, padre));
      }
      this.edicion.set(null);
      this.toast.exito('Categoría guardada');
      void this.cargar();
    } catch (e) {
      const err = ApiError.desde(e);
      this.errorEdicion.set(err.campos['nombre'] ?? err.message);
    }
  }

  protected async eliminar(c: CategoriaPlana): Promise<void> {
    const ok = await this.confirmacion.preguntar({ titulo: 'Eliminar categoría', mensaje: `¿Eliminar "${c.ruta}"?`, confirmar: 'Eliminar', peligro: true });
    if (!ok) {
      return;
    }
    try {
      await firstValueFrom(this.api.eliminarCategoria(c.id));
      this.toast.exito('Categoría eliminada');
      void this.cargar();
    } catch (e) {
      this.toast.error(e);
    }
  }

  protected finAjuste(aplicado: boolean): void {
    this.ajustando.set(null);
    if (aplicado) {
      void this.cargar();
    }
  }
}
