import { ChangeDetectionStrategy, Component, effect, inject, input, output, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { SesionCaja } from '../../core/api/caja.api';
import { Pagina } from '../../core/api/modelos';
import { MetodoPago, VentaResumen, VentasApi } from '../../core/api/ventas.api';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe } from '../../shared/formato/dinero';
import { FechaPipe } from '../../shared/formato/fecha';
import { ModalComponent } from '../../shared/ui/modal.component';
import { TablaComponent } from '../../shared/ui/tabla.component';
import { NOMBRE_MEDIO } from './arqueo';

/** Ventas hechas durante una caja: entre su apertura y su cierre (o hasta ahora si sigue abierta). */
@Component({
  selector: 'app-ventas-caja',
  imports: [DineroPipe, FechaPipe, ModalComponent, TablaComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-modal [abierto]="!!sesion()" titulo="Ventas de la caja" ancho="grande" (cerrar)="cerrar.emit()">
      @if (sesion(); as s) {
        <p class="mb-3 text-sm text-muted">
          Desde {{ s.abiertaEn | fecha }} {{ s.cerradaEn ? 'hasta ' + (s.cerradaEn | fecha) : 'hasta ahora (caja abierta)' }}.
          Las anuladas se muestran pero no suman en el arqueo.
        </p>
        <app-tabla [pagina]="pagina()" [cargando]="cargando()" mensajeVacio="No hubo ventas en esta caja"
                   (paginaCambio)="cargar(s, $event)">
          <table class="tabla">
            <caption class="sr-only">Ventas de la caja</caption>
            <thead>
              <tr><th scope="col">Hora</th><th scope="col">Detalle</th><th scope="col">Forma de pago</th>
                <th scope="col" class="text-right">Total</th><th scope="col">Estado</th></tr>
            </thead>
            <tbody>
              @for (v of pagina()?.items; track v.id) {
                <tr>
                  <td class="whitespace-nowrap">{{ v.fecha | fecha }}</td>
                  <td class="text-muted">{{ v.unidades }} {{ v.unidades === 1 ? 'unidad' : 'unidades' }}</td>
                  <td>
                    <div class="flex flex-wrap gap-1">
                      @for (m of v.metodos; track m) { <span class="badge-neutro">{{ nombre(m) }}</span> }
                    </div>
                  </td>
                  <td class="text-right font-semibold tabular-nums" [class.line-through]="v.estado === 'ANULADA'">{{ v.total | dinero }}</td>
                  <td><span [class]="v.estado === 'ANULADA' ? 'badge-danger' : 'badge-neutro'">{{ v.estado === 'ANULADA' ? 'Anulada' : 'Confirmada' }}</span></td>
                </tr>
              }
            </tbody>
          </table>
        </app-tabla>
      }
      <div pie>
        <button type="button" class="btn-secundario" (click)="cerrar.emit()">Cerrar</button>
      </div>
    </app-modal>
  `,
})
export class VentasCajaComponent {
  private readonly api = inject(VentasApi);
  private readonly toast = inject(ToastService);

  readonly sesion = input<SesionCaja | null>(null);
  readonly cerrar = output<void>();

  protected readonly pagina = signal<Pagina<VentaResumen> | null>(null);
  protected readonly cargando = signal(false);

  constructor() {
    effect(() => {
      const s = this.sesion();
      this.pagina.set(null);
      if (s) {
        void this.cargar(s, 1);
      }
    });
  }

  protected nombre(m: MetodoPago): string {
    return NOMBRE_MEDIO[m] ?? m;
  }

  protected async cargar(s: SesionCaja, page: number): Promise<void> {
    this.cargando.set(true);
    try {
      // Instantes exactos: hasta es exclusivo, igual que el período que usa el arqueo.
      this.pagina.set(await firstValueFrom(this.api.listar({ desde: s.abiertaEn, hasta: s.cerradaEn ?? undefined, page })));
    } catch (e) {
      this.toast.error(e);
    } finally {
      this.cargando.set(false);
    }
  }
}
