import { ChangeDetectionStrategy, Component, inject, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ApiError } from '../../core/api/api-error';
import { ProductoResumen } from '../../core/api/catalogo.api';
import { InventarioApi, StockProducto } from '../../core/api/inventario.api';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe } from '../../shared/formato/dinero';
import { FechaPipe } from '../../shared/formato/fecha';
import { BuscadorProductoComponent } from '../../shared/ui/buscador-producto.component';
import { CampoComponent } from '../../shared/ui/campo.component';
import { InputDineroComponent } from '../../shared/ui/input-dinero.component';
import { claseEstadoLote, textoEstadoLote } from './estados';

const MOTIVOS = ['Conteo de stock', 'Rotura', 'Vencido', 'Regalo / muestra', 'Uso personal', 'Mercadería encontrada'];

/** Corrección manual de stock: sumar o restar unidades con un motivo. */
@Component({
  selector: 'app-ajuste-stock',
  imports: [FormsModule, DineroPipe, FechaPipe, BuscadorProductoComponent, CampoComponent, InputDineroComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="grid items-start gap-6 lg:grid-cols-2">
      <section class="card space-y-4 p-5" aria-labelledby="titulo-ajuste">
        <h2 id="titulo-ajuste" class="font-semibold">Ajustar stock</h2>
        <app-buscador-producto idCampo="ajuste-buscar" etiqueta="Producto a ajustar" [compacto]="true" tipo="SIMPLE"
                               placeholder="Buscá o escaneá el producto" (elegido)="elegir($event)" />

        @if (stock(); as s) {
          <div class="rounded-xl bg-ink-50 px-4 py-3 dark:bg-ink-800/60">
            <p class="font-medium">{{ s.nombre }}</p>
            <p class="text-sm text-ink-500">{{ s.sku }} · stock actual <strong class="text-ink-900 dark:text-ink-100">{{ s.stock }}</strong></p>
          </div>

          <fieldset>
            <legend class="etiqueta">¿Qué querés hacer?</legend>
            <div class="flex gap-2">
              <label class="flex flex-1 cursor-pointer items-center gap-2 rounded-xl border px-3 py-2.5 text-sm"
                     [class]="sentido() === 'sumar' ? 'border-brand-500 bg-brand-50 dark:bg-brand-950' : 'border-ink-200 dark:border-ink-700'">
                <input type="radio" name="sentido" class="accent-brand-600" value="sumar" [ngModel]="sentido()" (ngModelChange)="sentido.set($event)" /> Sumar unidades
              </label>
              <label class="flex flex-1 cursor-pointer items-center gap-2 rounded-xl border px-3 py-2.5 text-sm"
                     [class]="sentido() === 'restar' ? 'border-brand-500 bg-brand-50 dark:bg-brand-950' : 'border-ink-200 dark:border-ink-700'">
                <input type="radio" name="sentido" class="accent-brand-600" value="restar" [ngModel]="sentido()" (ngModelChange)="sentido.set($event)" /> Quitar unidades
              </label>
            </div>
          </fieldset>

          <div class="grid gap-4 sm:grid-cols-2">
            <app-campo etiqueta="Cantidad" para="aj-cantidad" [error]="errores()['cantidad']">
              <input id="aj-cantidad" type="number" min="1" inputmode="numeric" class="input" [(ngModel)]="cantidad" />
            </app-campo>
            @if (s.lotes.length) {
              <app-campo etiqueta="Lote" para="aj-lote" [opcional]="true" [ayuda]="sentido() === 'restar' ? 'Sin elegir, sale primero lo que vence antes.' : 'Sin elegir, se crea un lote nuevo.'">
                <select id="aj-lote" class="input" [(ngModel)]="loteId">
                  <option value="">{{ sentido() === 'restar' ? 'Automático' : 'Lote nuevo' }}</option>
                  @for (l of s.lotes; track l.id) {
                    <option [value]="l.id">{{ l.vencimiento ? 'Vence ' + (l.vencimiento | fecha) : 'Sin vencimiento' }} · {{ l.stock }} u.</option>
                  }
                </select>
              </app-campo>
            }
            @if (sentido() === 'sumar' && !loteId) {
              <app-campo etiqueta="Vencimiento" para="aj-venc" [opcional]="true">
                <input id="aj-venc" type="date" class="input" [(ngModel)]="vencimiento" />
              </app-campo>
              <app-campo etiqueta="Costo por unidad" para="aj-costo" [opcional]="true">
                <app-input-dinero idCampo="aj-costo" [(ngModel)]="costo" />
              </app-campo>
            }
          </div>

          <app-campo etiqueta="Motivo" para="aj-motivo" [error]="errores()['motivo']">
            <input id="aj-motivo" class="input" maxlength="255" list="motivos-ajuste" [(ngModel)]="motivo" placeholder="Elegí o escribí el motivo" />
            <datalist id="motivos-ajuste">@for (m of motivos; track m) { <option [value]="m"></option> }</datalist>
          </app-campo>

          <div class="flex justify-end">
            <button type="button" class="btn-primario" [disabled]="guardando()" (click)="guardar()">
              {{ sentido() === 'sumar' ? 'Sumar' : 'Quitar' }} {{ cantidad || '' }} {{ cantidad === 1 ? 'unidad' : 'unidades' }}
            </button>
          </div>
        } @else {
          <p class="text-sm text-ink-500">Elegí un producto para ver su stock y corregirlo.</p>
        }
      </section>

      @if (stock(); as s) {
        <section class="card" aria-labelledby="titulo-lotes">
          <h2 id="titulo-lotes" class="border-b border-ink-100 px-5 py-4 font-semibold dark:border-ink-800">Lotes de {{ s.nombre }}</h2>
          @if (s.lotes.length) {
            <ul class="divide-y divide-ink-100 dark:divide-ink-800">
              @for (l of s.lotes; track l.id) {
                <li class="flex items-center justify-between gap-3 px-5 py-3 text-sm">
                  <div>
                    <p>{{ l.vencimiento ? 'Vence el ' + (l.vencimiento | fecha) : 'Sin fecha de vencimiento' }}</p>
                    <p class="text-xs text-ink-500">Costo {{ l.costo ? (l.costo | dinero) : 'sin cargar' }}</p>
                  </div>
                  <div class="text-right">
                    <p class="font-semibold tabular-nums">{{ l.stock }} u.</p>
                    <span [class]="claseEstado(l.estado)">{{ textoEstado(l.estado, l.diasRestantes) }}</span>
                  </div>
                </li>
              }
            </ul>
          } @else {
            <p class="px-5 py-8 text-center text-sm text-ink-500">No tiene unidades en stock.</p>
          }
        </section>
      }
    </div>
  `,
})
export class AjusteStockComponent {
  readonly ajustado = output<void>();

  private readonly api = inject(InventarioApi);
  private readonly toast = inject(ToastService);

  protected readonly motivos = MOTIVOS;
  protected readonly claseEstado = claseEstadoLote;
  protected readonly textoEstado = textoEstadoLote;
  protected readonly stock = signal<StockProducto | null>(null);
  protected readonly sentido = signal<'sumar' | 'restar'>('sumar');
  protected readonly errores = signal<Record<string, string>>({});
  protected readonly guardando = signal(false);
  protected cantidad: number | null = 1;
  protected loteId = '';
  protected vencimiento = '';
  protected costo: string | null = null;
  protected motivo = '';

  protected async elegir(p: ProductoResumen): Promise<void> {
    try {
      this.stock.set(await firstValueFrom(this.api.stock(p.id)));
      this.loteId = '';
      this.errores.set({});
    } catch (e) {
      this.toast.error(e);
    }
  }

  protected async guardar(): Promise<void> {
    const s = this.stock();
    const n = Math.trunc(Number(this.cantidad));
    const locales: Record<string, string> = {};
    if (!n || n < 1) {
      locales['cantidad'] = 'Ingresá una cantidad mayor a cero';
    }
    if (!this.motivo.trim()) {
      locales['motivo'] = 'Contá el motivo del ajuste';
    }
    this.errores.set(locales);
    if (!s || Object.keys(locales).length) {
      return;
    }
    const sumar = this.sentido() === 'sumar';
    this.guardando.set(true);
    try {
      const nuevo = await firstValueFrom(this.api.ajustar({
        productoId: s.productoId, cantidad: sumar ? n : -n, motivo: this.motivo.trim(), loteId: this.loteId || undefined,
        vencimiento: sumar && !this.loteId && this.vencimiento ? this.vencimiento : null,
        costo: sumar && !this.loteId ? this.costo : null,
      }));
      this.stock.set(nuevo);
      this.toast.exito(`Stock de ${nuevo.nombre}: ${nuevo.stock}`);
      this.cantidad = 1;
      this.motivo = '';
      this.vencimiento = '';
      this.costo = null;
      this.loteId = '';
      this.ajustado.emit();
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
