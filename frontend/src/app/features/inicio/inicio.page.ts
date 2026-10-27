import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { CircleAlert, LucideAngularModule, ShoppingCart } from 'lucide-angular';
import { EstadisticasApi, ProductoVendido, Resumen, VentaDia } from '../../core/api/estadisticas.api';
import { Alertas, InventarioApi } from '../../core/api/inventario.api';
import { ToastService } from '../../core/ui/toast.service';
import { formatearDinero } from '../../shared/formato/dinero';
import { columnasPorDia, filasMasVendidos } from '../../shared/graficos/adaptadores';
import { BarrasHorizontalesComponent } from '../../shared/graficos/barras-horizontales.component';
import { GraficoColumnasComponent } from '../../shared/graficos/grafico-columnas.component';
import { TarjetaDatoComponent } from '../../shared/graficos/tarjeta-dato.component';

/** Tablero de inicio: cómo va el día, la semana y qué requiere atención. */
@Component({
  selector: 'app-inicio',
  imports: [RouterLink, LucideAngularModule, BarrasHorizontalesComponent, GraficoColumnasComponent, TarjetaDatoComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h1 class="titulo-pagina">{{ saludo() }}</h1>
        <p class="mt-1 text-sm text-muted">Así va el día</p>
      </div>
      <a routerLink="/vender" class="btn-primario h-12 px-6 text-base">
        <lucide-icon [img]="iconos.vender" [size]="20" aria-hidden="true" /> Vender
      </a>
    </div>

    @if (resumen(); as r) {
      <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" [class.opacity-60]="cargando()">
        <app-tarjeta-dato class="sm:col-span-2" etiqueta="Vendido hoy" [valor]="dinero(r.ventas.total)" [destacado]="true"
                          [variacion]="r.anterior.variacionTotal" comparacion="vs. ayer a esta hora"
                          [detalle]="r.anterior.variacionTotal === null ? r.ventas.cantidad + ' ' + (r.ventas.cantidad === 1 ? 'venta' : 'ventas') : ''" />
        <app-tarjeta-dato etiqueta="Ganancia de hoy" [valor]="dinero(r.ventas.ganancia)"
                          [detalle]="'Margen ' + margen(r.ventas.margen) + (r.ventas.renglonesSinCosto ? ' (aproximado)' : '')" />
        <app-tarjeta-dato etiqueta="Efectivo de hoy" [valor]="dinero(r.efectivoNeto)"
                          [detalle]="r.efectivoNeto.startsWith('-') ? 'Salió más efectivo del que entró (gastos en efectivo)' : 'Cobrado en efectivo menos gastos en efectivo'" />
      </div>
    } @else if (cargando()) {
      <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        @for (_ of [1, 2, 3]; track $index) { <div class="card h-32 animate-pulse first:sm:col-span-2"></div> }
      </div>
    }

    @if (alertasVisibles(); as a) {
      <a routerLink="/inventario" class="card mt-4 flex items-center gap-3 border-warn-line bg-warn-soft p-4 text-sm transition-colors duration-150 hover:border-ink-300">
        <lucide-icon [img]="iconos.alerta" [size]="20" class="shrink-0 text-warn" aria-hidden="true" />
        <span class="flex-1">
          <strong>Para revisar:</strong> {{ a }}
        </span>
        <span class="font-medium text-warn">Ver inventario →</span>
      </a>
    }

    <div class="mt-6 grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
      <section class="card p-5" aria-labelledby="titulo-semana">
        <h2 id="titulo-semana" class="font-semibold">Ventas de la semana</h2>
        <p class="mb-4 text-sm text-muted">Total vendido por día, de lunes a domingo</p>
        @if (semana().length) {
          <app-grafico-columnas [datos]="columnasSemana()" titulo="Ventas de la semana por día" etiquetaX="Día" prefijo="$" />
        }
      </section>
      <section class="card p-5" aria-labelledby="titulo-top">
        <h2 id="titulo-top" class="font-semibold">Más vendidos de la semana</h2>
        <p class="mb-4 text-sm text-muted">Por unidades</p>
        <app-barras-horizontales [filas]="filasTop()" titulo="Más vendidos de la semana" [numerada]="true"
                                 vacio="Todavía no hay ventas esta semana" />
        <a routerLink="/estadisticas" class="mt-4 inline-block text-sm font-medium text-brand-600 hover:underline dark:text-brand-300">Ver estadísticas completas →</a>
      </section>
    </div>
  `,
})
export class InicioPage {
  private readonly api = inject(EstadisticasApi);
  private readonly inventario = inject(InventarioApi);
  private readonly toast = inject(ToastService);

  protected readonly iconos = { vender: ShoppingCart, alerta: CircleAlert };
  protected readonly resumen = signal<Resumen | null>(null);
  protected readonly semana = signal<VentaDia[]>([]);
  protected readonly top = signal<ProductoVendido[]>([]);
  protected readonly alertas = signal<Alertas | null>(null);
  protected readonly cargando = signal(true);

  protected readonly columnasSemana = computed(() => columnasPorDia(this.semana()));
  protected readonly filasTop = computed(() => filasMasVendidos(this.top(), 'unidades'));
  protected readonly alertasVisibles = computed(() => {
    const a = this.alertas();
    if (!a) {
      return '';
    }
    const partes = [
      a.vencidos && `${a.vencidos} ${a.vencidos === 1 ? 'lote vencido' : 'lotes vencidos'}`,
      a.porVencer && `${a.porVencer} por vencer`,
      a.sinStock && `${a.sinStock} ${a.sinStock === 1 ? 'producto' : 'productos'} sin stock`,
      a.stockBajo && `${a.stockBajo} con stock bajo`,
    ].filter(Boolean);
    return partes.join(' · ');
  });

  protected readonly saludo = computed(() => {
    const h = Number(new Intl.DateTimeFormat('es-AR', { hour: 'numeric', hourCycle: 'h23', timeZone: 'America/Argentina/Buenos_Aires' }).format(new Date()));
    return h < 12 ? '¡Buen día!' : h < 20 ? '¡Buenas tardes!' : '¡Buenas noches!';
  });

  constructor() {
    void this.cargar();
  }

  private async cargar(): Promise<void> {
    try {
      const [resumen, semana, top, alertas] = await Promise.all([
        firstValueFrom(this.api.resumen({ periodo: 'dia' })),
        firstValueFrom(this.api.ventasPorDia({ periodo: 'semana' })),
        firstValueFrom(this.api.masVendidos({ periodo: 'semana' }, 5)),
        firstValueFrom(this.inventario.alertas()),
      ]);
      this.resumen.set(resumen);
      this.semana.set(semana);
      this.top.set(top);
      this.alertas.set(alertas);
    } catch (e) {
      this.toast.error(e);
    } finally {
      this.cargando.set(false);
    }
  }

  protected dinero(m: string): string {
    return formatearDinero(m);
  }

  protected margen(m: number): string {
    return `${new Intl.NumberFormat('es-AR', { maximumFractionDigits: 1 }).format(m)} %`;
  }
}
