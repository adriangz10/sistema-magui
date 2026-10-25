import { ChangeDetectionStrategy, Component, computed, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ApiError } from '../../core/api/api-error';
import { AjustePrecios, CatalogoApi, ResultadoAjuste } from '../../core/api/catalogo.api';
import { ConfirmacionService } from '../../core/ui/confirmacion.service';
import { ToastService } from '../../core/ui/toast.service';
import { CategoriaPlana } from '../../shared/catalogo/categorias';
import { DineroPipe, interpretarDinero } from '../../shared/formato/dinero';
import { CampoComponent } from '../../shared/ui/campo.component';
import { PaginadorComponent } from '../../shared/ui/paginador.component';

const REDONDEOS = [
  { valor: '', etiqueta: 'Sin redondeo' },
  { valor: '10', etiqueta: 'A $10' },
  { valor: '50', etiqueta: 'A $50' },
  { valor: '100', etiqueta: 'A $100' },
  { valor: '500', etiqueta: 'A $500' },
  { valor: '1000', etiqueta: 'A $1.000' },
];

/** Cambio de precios en cascada para una categoría: vista previa y aplicación. */
@Component({
  selector: 'app-ajuste-precios',
  imports: [FormsModule, DineroPipe, CampoComponent, PaginadorComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <p class="mb-4 text-sm text-ink-500">Categoría: <strong class="text-ink-800 dark:text-ink-200">{{ categoria().ruta }}</strong>. Solo cambia productos activos.</p>

    <div class="grid gap-4 sm:grid-cols-2">
      <fieldset class="sm:col-span-2">
        <legend class="etiqueta">Tipo de cambio</legend>
        <div class="flex gap-2">
          @for (t of tipos; track t.valor) {
            <label class="flex flex-1 cursor-pointer items-center gap-2 rounded-xl border px-3 py-2.5 text-sm"
                   [class]="tipo() === t.valor ? 'border-brand-500 bg-brand-50 dark:bg-brand-950' : 'border-ink-200 dark:border-ink-700'">
              <input type="radio" name="tipo-ajuste" class="accent-brand-600" [value]="t.valor" [ngModel]="tipo()" (ngModelChange)="cambiar(() => tipo.set($event))" />
              {{ t.etiqueta }}
            </label>
          }
        </div>
      </fieldset>
      <app-campo [etiqueta]="tipo() === 'PORCENTAJE' ? 'Porcentaje (%)' : 'Monto ($)'" para="aj-valor" [error]="errorValor()"
                 ayuda="Usá un número negativo para bajar (ej. -5).">
        <input id="aj-valor" class="input" inputmode="decimal" [ngModel]="valor()" (ngModelChange)="cambiar(() => valor.set($event))"
               [placeholder]="tipo() === 'PORCENTAJE' ? 'Ej.: 10' : 'Ej.: 500'"
               [attr.aria-describedby]="errorValor() ? 'aj-valor-error' : 'aj-valor-ayuda'" />
      </app-campo>
      <app-campo etiqueta="Redondear el resultado" para="aj-redondeo">
        <select id="aj-redondeo" class="input" [ngModel]="redondeo()" (ngModelChange)="cambiar(() => redondeo.set($event))">
          @for (r of redondeos; track r.valor) { <option [value]="r.valor">{{ r.etiqueta }}</option> }
        </select>
      </app-campo>
      <label class="inline-flex items-center gap-2 text-sm sm:col-span-2">
        <input type="checkbox" class="size-4 rounded accent-brand-600" [ngModel]="incluirSub()" (ngModelChange)="cambiar(() => incluirSub.set($event))" />
        Incluir las subcategorías
      </label>
      <app-campo class="sm:col-span-2" etiqueta="Motivo" para="aj-motivo" [opcional]="true">
        <input id="aj-motivo" class="input" maxlength="255" [(ngModel)]="motivo" placeholder="Ej.: aumento de Natura ciclo 15" />
      </app-campo>
    </div>

    <div class="mt-4 flex justify-end">
      <button type="button" class="btn-secundario" [disabled]="cargando()" (click)="previsualizar()">Ver cómo quedan los precios</button>
    </div>

    @if (previa(); as r) {
      <div class="mt-4 rounded-2xl border border-ink-200 dark:border-ink-700">
        @if (r.cantidad === 0) {
          <p class="p-6 text-center text-sm text-ink-500">Ningún producto cambia de precio con este ajuste.</p>
        } @else {
          <p class="border-b border-ink-100 px-4 py-3 text-sm font-medium dark:border-ink-800">{{ r.cantidad }} productos van a cambiar</p>
          <ul class="divide-y divide-ink-100 dark:divide-ink-800">
            @for (p of paginaPrevia(); track p.productoId) {
              <li class="flex items-center justify-between gap-3 px-4 py-2 text-sm">
                <span class="min-w-0 truncate">{{ p.nombre }}</span>
                <span class="shrink-0 tabular-nums"><span class="text-ink-500 line-through">{{ p.precioAnterior | dinero }}</span> → <strong>{{ p.precioNuevo | dinero }}</strong></span>
              </li>
            }
          </ul>
          <app-paginador [pagina]="paginaActual()" [totalPaginas]="totalPaginas()" [total]="r.cantidad" (cambio)="paginaActual.set($event)" />
        }
      </div>
    }

    <div class="mt-5 flex justify-end gap-2 border-t border-ink-100 pt-4 dark:border-ink-800">
      <button type="button" class="btn-secundario" (click)="cerrar.emit(false)">Cancelar</button>
      <button type="button" class="btn-primario" [disabled]="!previa()?.cantidad || cargando()" (click)="aplicar()">
        Aplicar a {{ previa()?.cantidad ?? 0 }} productos
      </button>
    </div>
  `,
})
export class AjustePreciosComponent {
  readonly categoria = input.required<CategoriaPlana>();
  /** true si se aplicaron cambios. */
  readonly cerrar = output<boolean>();

  private readonly api = inject(CatalogoApi);
  private readonly toast = inject(ToastService);
  private readonly confirmacion = inject(ConfirmacionService);

  protected readonly tipos = [{ valor: 'PORCENTAJE', etiqueta: 'Porcentaje' }, { valor: 'MONTO', etiqueta: 'Monto fijo' }] as const;
  protected readonly redondeos = REDONDEOS;
  protected readonly tipo = signal<AjustePrecios['tipo']>('PORCENTAJE');
  protected readonly valor = signal('');
  protected readonly redondeo = signal('100');
  protected readonly incluirSub = signal(true);
  protected motivo = '';
  protected readonly previa = signal<ResultadoAjuste | null>(null);
  protected readonly cargando = signal(false);
  protected readonly errorValor = signal('');
  protected readonly paginaActual = signal(1);
  protected readonly totalPaginas = computed(() => Math.ceil((this.previa()?.cantidad ?? 0) / 10));
  protected readonly paginaPrevia = computed(() => (this.previa()?.productos ?? []).slice((this.paginaActual() - 1) * 10, this.paginaActual() * 10));

  /** Cualquier cambio invalida la vista previa (hay que volver a calcularla antes de aplicar). */
  protected cambiar(fn: () => void): void {
    fn();
    this.previa.set(null);
    this.errorValor.set('');
  }

  private pedido(): AjustePrecios | null {
    const v = interpretarDinero(this.valor().replace('%', ''));
    if (!v || v === '0.00' || v === '-0.00') {
      this.errorValor.set('Ingresá un número distinto de cero');
      return null;
    }
    return { tipo: this.tipo(), valor: v, redondeo: this.redondeo() || null, incluirSubcategorias: this.incluirSub(), motivo: this.motivo };
  }

  protected async previsualizar(): Promise<void> {
    const p = this.pedido();
    if (!p) {
      return;
    }
    this.cargando.set(true);
    try {
      this.previa.set(await firstValueFrom(this.api.ajustarPrecios(this.categoria().id, p, false)));
      this.paginaActual.set(1);
    } catch (e) {
      this.mostrarError(e);
    } finally {
      this.cargando.set(false);
    }
  }

  protected async aplicar(): Promise<void> {
    const p = this.pedido();
    const cantidad = this.previa()?.cantidad ?? 0;
    if (!p || !cantidad) {
      return;
    }
    const ok = await this.confirmacion.preguntar({
      titulo: 'Aplicar cambio de precios',
      mensaje: `Se van a cambiar los precios de ${cantidad} productos. Queda registrado en el historial de cada uno.`,
      confirmar: 'Aplicar',
    });
    if (!ok) {
      return;
    }
    this.cargando.set(true);
    try {
      const r = await firstValueFrom(this.api.ajustarPrecios(this.categoria().id, p, true));
      this.toast.exito(`Precios actualizados en ${r.cantidad} productos`);
      this.cerrar.emit(true);
    } catch (e) {
      this.mostrarError(e);
    } finally {
      this.cargando.set(false);
    }
  }

  private mostrarError(e: unknown): void {
    const err = ApiError.desde(e);
    if (err.campos['valor']) {
      this.errorValor.set(err.campos['valor']);
    } else {
      this.toast.error(err);
    }
  }
}
