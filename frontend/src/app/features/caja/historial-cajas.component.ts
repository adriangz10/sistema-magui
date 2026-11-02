import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { CajaApi, CambioCaja, SesionCaja, SesionCajaDetalle } from '../../core/api/caja.api';
import { Pagina } from '../../core/api/modelos';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe } from '../../shared/formato/dinero';
import { FechaPipe } from '../../shared/formato/fecha';
import { ModalComponent } from '../../shared/ui/modal.component';
import { TablaComponent } from '../../shared/ui/tabla.component';
import { ArqueoTablaComponent } from './arqueo-tabla.component';
import { claseDiferencia, etiquetaCampo, textoDiferencia, valorCambio } from './arqueo';
import { EditarCajaComponent } from './editar-caja.component';
import { VentasCajaComponent } from './ventas-caja.component';

/** Historial de cajas con su conciliación, detalle, edición y ventas de cada una (US-002). */
@Component({
  selector: 'app-historial-cajas',
  imports: [DineroPipe, FechaPipe, ModalComponent, TablaComponent, ArqueoTablaComponent, EditarCajaComponent, VentasCajaComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-tabla [pagina]="pagina()" [cargando]="cargando()" mensajeVacio="Todavía no hay cajas"
               ayudaVacio="Cada vez que abrís y cerrás la caja queda un registro acá." (paginaCambio)="cargar($event)">
      <table class="tabla">
        <caption class="sr-only">Historial de cajas</caption>
        <thead>
          <tr>
            <th scope="col">Apertura</th><th scope="col">Cierre</th><th scope="col" class="text-right">Ventas</th>
            <th scope="col" class="text-right">Según el sistema</th><th scope="col" class="text-right">Contado</th>
            <th scope="col">Resultado</th><th scope="col"><span class="sr-only">Acciones</span></th>
          </tr>
        </thead>
        <tbody>
          @for (s of pagina()?.items; track s.id) {
            <tr>
              <td class="whitespace-nowrap">{{ s.abiertaEn | fecha }}</td>
              <td class="whitespace-nowrap">
                @if (s.cerradaEn) { {{ s.cerradaEn | fecha }} } @else { <span class="badge-ok">Abierta</span> }
              </td>
              <td class="text-right tabular-nums">{{ s.cantidadVentas }}</td>
              <td class="text-right tabular-nums">{{ s.totalEsperado | dinero }}</td>
              <td class="text-right tabular-nums">{{ s.totalContado | dinero }}</td>
              <td>
                @if (!s.abierta) {
                  <span [class]="clase(s.diferencia)">{{ texto(s.diferencia) }}</span>
                  @if (s.editada) { <span class="ml-1 text-xs text-muted">(editada)</span> }
                }
              </td>
              <td class="whitespace-nowrap text-right">
                <button type="button" class="btn-fantasma min-h-8 px-3" (click)="viendoVentas.set(s)"
                        [attr.aria-label]="'Ver ventas de la caja del ' + (s.abiertaEn | fecha)">Ventas</button>
                <button type="button" class="btn-fantasma min-h-8 px-3" (click)="verDetalle(s)"
                        [attr.aria-label]="'Ver detalle de la caja del ' + (s.abiertaEn | fecha)">Detalle</button>
              </td>
            </tr>
          }
        </tbody>
      </table>
    </app-tabla>

    <app-modal [abierto]="!!detalle()" titulo="Detalle de la caja" ancho="grande" (cerrar)="detalle.set(null)">
      @if (detalle(); as d) {
        <dl class="mb-4 grid gap-2 text-sm sm:grid-cols-2">
          <div><dt class="text-muted">Apertura</dt><dd class="font-medium">{{ d.abiertaEn | fecha }}</dd></div>
          <div><dt class="text-muted">Cierre</dt><dd class="font-medium">{{ d.cerradaEn ? (d.cerradaEn | fecha) : 'Sigue abierta' }}</dd></div>
          <div><dt class="text-muted">Monto inicial</dt><dd class="font-medium tabular-nums">{{ d.montoInicial | dinero }}</dd></div>
          <div><dt class="text-muted">Ventas confirmadas</dt><dd class="font-medium tabular-nums">{{ d.cantidadVentas }}</dd></div>
          @if (d.notas) { <div class="sm:col-span-2"><dt class="text-muted">Notas</dt><dd>{{ d.notas }}</dd></div> }
        </dl>
        <app-arqueo-tabla [sesion]="d" />

        <section aria-labelledby="hist-caja-titulo" class="mt-6">
          <h3 id="hist-caja-titulo" class="mb-2 text-sm font-semibold">Historial de ediciones</h3>
          @if (d.ediciones.length) {
            <ul class="divide-y divide-ink-100 dark:divide-ink-800">
              @for (e of d.ediciones; track e.id) {
                <li class="py-3 text-sm">
                  <p><strong class="tabular-nums">{{ e.fecha | fecha }}</strong></p>
                  <p class="text-muted">Motivo: {{ e.motivo }}</p>
                  <ul class="mt-1 space-y-0.5">
                    @for (c of e.cambios; track c.campo) {
                      <li>{{ etiqueta(c.campo) }}: <span class="text-muted">{{ valor(c, c.antes) }}</span> → <strong>{{ valor(c, c.despues) }}</strong></li>
                    }
                  </ul>
                </li>
              }
            </ul>
          } @else {
            <p class="text-sm text-muted">Sin ediciones</p>
          }
        </section>
      }
      <div pie>
        @if (detalle(); as d) {
          <button type="button" class="btn-secundario" (click)="viendoVentas.set(d)">Ver ventas</button>
          <button type="button" class="btn-primario" (click)="editando.set(d)">Editar</button>
        }
      </div>
    </app-modal>

    <app-editar-caja [sesion]="editando()" (guardado)="alEditar($event)" (cerrar)="editando.set(null)" />
    <app-ventas-caja [sesion]="viendoVentas()" (cerrar)="viendoVentas.set(null)" />
  `,
})
export class HistorialCajasComponent {
  private readonly api = inject(CajaApi);
  private readonly toast = inject(ToastService);

  protected readonly pagina = signal<Pagina<SesionCaja> | null>(null);
  protected readonly cargando = signal(false);
  protected readonly detalle = signal<SesionCajaDetalle | null>(null);
  protected readonly editando = signal<SesionCaja | null>(null);
  protected readonly viendoVentas = signal<SesionCaja | null>(null);
  protected readonly texto = textoDiferencia;
  protected readonly clase = claseDiferencia;
  protected readonly etiqueta = etiquetaCampo;
  private paginaActual = 1;

  constructor() {
    void this.cargar(1);
  }

  async cargar(page = this.paginaActual): Promise<void> {
    this.paginaActual = page;
    this.cargando.set(true);
    try {
      this.pagina.set(await firstValueFrom(this.api.sesiones(page)));
    } catch (e) {
      this.toast.error(e);
    } finally {
      this.cargando.set(false);
    }
  }

  protected valor(c: CambioCaja, v: string | null): string {
    return valorCambio(c, v);
  }

  protected async verDetalle(s: SesionCaja): Promise<void> {
    try {
      this.detalle.set(await firstValueFrom(this.api.sesion(s.id)));
    } catch (e) {
      this.toast.error(e);
    }
  }

  protected alEditar(d: SesionCajaDetalle): void {
    this.editando.set(null);
    if (this.detalle()) {
      this.detalle.set(d);
    }
    void this.cargar();
  }
}
