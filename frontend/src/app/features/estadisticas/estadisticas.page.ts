import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ConsultaPeriodo, EstadisticasApi, ProductoVendido, Resumen, TipoPeriodo, VentaDia, VentaHora } from '../../core/api/estadisticas.api';
import { ToastService } from '../../core/ui/toast.service';
import { aCentavos, formatearDinero } from '../../shared/formato/dinero';
import { hoyISO } from '../../shared/formato/fecha';
import { columnasPorDia, columnasPorHora, filasMasVendidos } from '../../shared/graficos/adaptadores';
import { BarrasHorizontalesComponent, FilaBarra } from '../../shared/graficos/barras-horizontales.component';
import { GraficoColumnasComponent } from '../../shared/graficos/grafico-columnas.component';
import { TarjetaDatoComponent } from '../../shared/graficos/tarjeta-dato.component';
import { CampoComponent } from '../../shared/ui/campo.component';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';

const PERIODOS: { id: TipoPeriodo; etiqueta: string; comparacion: string }[] = [
  { id: 'dia', etiqueta: 'Hoy', comparacion: 'vs. ayer a esta hora' },
  { id: 'semana', etiqueta: 'Esta semana', comparacion: 'vs. el mismo tramo de la semana pasada' },
  { id: 'mes', etiqueta: 'Este mes', comparacion: 'vs. el mismo tramo del mes pasado' },
  { id: 'anio', etiqueta: 'Este año', comparacion: 'vs. el mismo tramo del año pasado' },
  { id: 'rango', etiqueta: 'Elegir fechas', comparacion: 'vs. el período anterior' },
];

const NOMBRE_METODO: Record<string, string> = { EFECTIVO: 'Efectivo', TRANSFERENCIA: 'Transferencia', QR: 'QR', TARJETA: 'Tarjeta' };

/** Estadísticas de ventas, ganancias, compras y gastos por período. */
@Component({
  selector: 'app-estadisticas',
  imports: [FormsModule, BarrasHorizontalesComponent, GraficoColumnasComponent, TarjetaDatoComponent, CampoComponent, EncabezadoComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-encabezado titulo="Estadísticas" descripcion="Cuánto se vende, cuánto se gana y en qué se gasta" />

    <!-- Filtro de período: una sola fila arriba de todo -->
    <div class="mb-6 flex flex-wrap items-end gap-2" role="group" aria-label="Período">
      @for (p of periodos; track p.id) {
        <button type="button" class="btn min-h-9 rounded-full px-4" [attr.aria-pressed]="periodo() === p.id"
                [class]="periodo() === p.id ? 'bg-ink-900 text-white dark:bg-ink-100 dark:text-ink-900' : 'bg-white text-ink-700 ring-1 ring-ink-200 dark:bg-ink-900 dark:text-ink-200 dark:ring-ink-700'"
                (click)="elegirPeriodo(p.id)">{{ p.etiqueta }}</button>
      }
      @if (periodo() === 'rango') {
        <app-campo etiqueta="Desde" para="e-desde">
          <input id="e-desde" type="date" class="input min-h-9" [max]="hoy" [ngModel]="desde()" (ngModelChange)="desde.set($event); cargar()" />
        </app-campo>
        <app-campo etiqueta="Hasta" para="e-hasta">
          <input id="e-hasta" type="date" class="input min-h-9" [max]="hoy" [ngModel]="hasta()" (ngModelChange)="hasta.set($event); cargar()" />
        </app-campo>
      }
    </div>

    @if (resumen(); as r) {
      <div [class.opacity-60]="cargando()" class="transition-opacity">
        <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <app-tarjeta-dato etiqueta="Vendido" [valor]="dinero(r.ventas.total)" [variacion]="r.anterior.variacionTotal" [comparacion]="comparacion()"
                            [detalle]="r.ventas.cantidad + ' ventas'" />
          <app-tarjeta-dato etiqueta="Ganancia" [valor]="dinero(r.ventas.ganancia)" [variacion]="r.anterior.variacionGanancia" [comparacion]="comparacion()" />
          <app-tarjeta-dato etiqueta="Margen de ganancia" [valor]="porcentaje(r.ventas.margen)" detalle="Ganancia sobre lo vendido" />
          <app-tarjeta-dato etiqueta="Ganancia neta" [valor]="dinero(r.gananciaNeta)" detalle="Ganancia menos gastos de caja" />
          <app-tarjeta-dato etiqueta="Ventas" [valor]="r.ventas.cantidad.toString()" [detalle]="'Ticket promedio ' + dinero(r.ventas.ticketPromedio)" />
          <app-tarjeta-dato etiqueta="Compras de mercadería" [valor]="dinero(r.compras.total)" [detalle]="r.compras.cantidad + ' ' + (r.compras.cantidad === 1 ? 'compra' : 'compras')" />
          <app-tarjeta-dato etiqueta="Gastos de caja" [valor]="dinero(r.gastos.total)" [detalle]="r.gastos.cantidad + ' ' + (r.gastos.cantidad === 1 ? 'gasto' : 'gastos')" />
          <app-tarjeta-dato etiqueta="Efectivo neto" [valor]="dinero(r.efectivoNeto)" detalle="Cobrado en efectivo menos gastos en efectivo" />
        </div>

        @if (r.ventas.renglonesSinCosto) {
          <p class="mt-4 rounded-xl bg-amber-50 px-4 py-3 text-sm text-warn dark:bg-amber-950 dark:text-amber-300" role="note">
            {{ r.ventas.renglonesSinCosto }} {{ r.ventas.renglonesSinCosto === 1 ? 'producto vendido no tenía' : 'productos vendidos no tenían' }}
            costo cargado: la ganancia real es menor que la que se muestra. Se completa sola al registrar compras de esos productos.
          </p>
        }
        @if (r.ventas.anuladas) {
          <p class="mt-2 text-sm text-ink-500">{{ r.ventas.anuladas }} {{ r.ventas.anuladas === 1 ? 'venta anulada' : 'ventas anuladas' }} en el período (no se cuentan).</p>
        }

        <div class="mt-6 grid gap-6 lg:grid-cols-2">
          <section class="card p-5 lg:col-span-2" aria-labelledby="titulo-dias">
            <h2 id="titulo-dias" class="font-semibold">Ventas por día</h2>
            @if (periodo() === 'anio') {
              <p class="mt-2 text-sm text-ink-500">Para ver el detalle día por día elegí un mes o un rango de fechas.</p>
            } @else {
              <p class="mb-4 text-sm text-ink-500">Total vendido cada día del período</p>
              <app-grafico-columnas [datos]="columnasDias()" titulo="Ventas por día" etiquetaX="Día" prefijo="$" />
            }
          </section>

          <section class="card p-5" aria-labelledby="titulo-horas">
            <h2 id="titulo-horas" class="font-semibold">¿A qué hora se vende más?</h2>
            <p class="mb-4 text-sm text-ink-500">Total vendido por hora del día</p>
            <app-grafico-columnas [datos]="columnasHoras()" titulo="Ventas por hora" etiquetaX="Hora" prefijo="$" />
          </section>

          <section class="card p-5" aria-labelledby="titulo-metodos">
            <h2 id="titulo-metodos" class="font-semibold">Formas de pago</h2>
            <p class="mb-4 text-sm text-ink-500">Cuánto entró por cada medio</p>
            <app-barras-horizontales [filas]="filasMetodos()" titulo="Formas de pago" />
          </section>

          <section class="card p-5 lg:col-span-2" aria-labelledby="titulo-top">
            <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
              <div>
                <h2 id="titulo-top" class="font-semibold">Productos más vendidos</h2>
                <p class="text-sm text-ink-500">Los 10 primeros del período</p>
              </div>
              <div class="flex rounded-xl ring-1 ring-ink-200 dark:ring-ink-700" role="group" aria-label="Ordenar ranking por">
                @for (o of ordenes; track o.id) {
                  <button type="button" class="btn min-h-9 px-3" [attr.aria-pressed]="orden() === o.id"
                          [class]="orden() === o.id ? 'bg-ink-900 text-white dark:bg-ink-100 dark:text-ink-900' : 'text-ink-600 dark:text-ink-300'"
                          (click)="cambiarOrden(o.id)">{{ o.etiqueta }}</button>
                }
              </div>
            </div>
            <app-barras-horizontales [filas]="filasTop()" titulo="Productos más vendidos" [numerada]="true" />
          </section>
        </div>
      </div>
    } @else if (cargando()) {
      <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">@for (_ of [1, 2, 3, 4]; track $index) { <div class="card h-28 animate-pulse"></div> }</div>
    } @else if (periodo() === 'rango') {
      <p class="card p-6 text-center text-sm text-ink-500">Elegí desde y hasta qué fecha.</p>
    }
  `,
})
export class EstadisticasPage {
  private readonly api = inject(EstadisticasApi);
  private readonly toast = inject(ToastService);

  protected readonly periodos = PERIODOS;
  protected readonly ordenes = [{ id: 'unidades' as const, etiqueta: 'Unidades' }, { id: 'importe' as const, etiqueta: 'Importe' }];
  protected readonly hoy = hoyISO();
  protected readonly periodo = signal<TipoPeriodo>('mes');
  protected readonly desde = signal('');
  protected readonly hasta = signal('');
  protected readonly orden = signal<'unidades' | 'importe'>('unidades');
  protected readonly resumen = signal<Resumen | null>(null);
  protected readonly dias = signal<VentaDia[]>([]);
  protected readonly horas = signal<VentaHora[]>([]);
  protected readonly top = signal<ProductoVendido[]>([]);
  protected readonly cargando = signal(false);

  protected readonly comparacion = computed(() => PERIODOS.find((p) => p.id === this.periodo())?.comparacion ?? '');
  protected readonly columnasDias = computed(() => columnasPorDia(this.dias()));
  protected readonly columnasHoras = computed(() => columnasPorHora(this.horas()));
  protected readonly filasTop = computed(() => filasMasVendidos(this.top(), this.orden()));
  protected readonly filasMetodos = computed<FilaBarra[]>(() =>
    (this.resumen()?.porMetodo ?? []).map((m) => ({
      etiqueta: NOMBRE_METODO[m.metodo] ?? m.metodo,
      valor: aCentavos(m.monto),
      texto: formatearDinero(m.monto),
      detalle: `${this.porcentaje(m.porcentaje)} del total · ${m.cantidad} ${m.cantidad === 1 ? 'venta' : 'ventas'}`,
    })));

  constructor() {
    void this.cargar();
  }

  private consulta(): ConsultaPeriodo | null {
    if (this.periodo() !== 'rango') {
      return { periodo: this.periodo() };
    }
    if (!this.desde() || !this.hasta()) {
      return null;
    }
    return { periodo: 'rango', desde: this.desde(), hasta: this.hasta() };
  }

  protected elegirPeriodo(p: TipoPeriodo): void {
    this.periodo.set(p);
    if (p === 'rango' && !this.desde()) {
      this.desde.set(this.hoy.slice(0, 8) + '01');
      this.hasta.set(this.hoy);
    }
    void this.cargar();
  }

  protected cambiarOrden(o: 'unidades' | 'importe'): void {
    this.orden.set(o);
    const c = this.consulta();
    if (c) {
      void firstValueFrom(this.api.masVendidos(c, 10, o)).then((t) => this.top.set(t)).catch((e) => this.toast.error(e));
    }
  }

  async cargar(): Promise<void> {
    const c = this.consulta();
    if (!c) {
      this.resumen.set(null);
      return;
    }
    this.cargando.set(true);
    try {
      const [resumen, horas, top, dias] = await Promise.all([
        firstValueFrom(this.api.resumen(c)),
        firstValueFrom(this.api.ventasPorHora(c)),
        firstValueFrom(this.api.masVendidos(c, 10, this.orden())),
        c.periodo === 'anio' ? Promise.resolve([] as VentaDia[]) : firstValueFrom(this.api.ventasPorDia(c)),
      ]);
      this.resumen.set(resumen);
      this.horas.set(horas);
      this.top.set(top);
      this.dias.set(dias);
    } catch (e) {
      this.toast.error(e);
    } finally {
      this.cargando.set(false);
    }
  }

  protected dinero(m: string): string {
    return formatearDinero(m);
  }

  protected porcentaje(n: number): string {
    return `${new Intl.NumberFormat('es-AR', { maximumFractionDigits: 1 }).format(n)} %`;
  }
}
