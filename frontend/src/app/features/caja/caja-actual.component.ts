import { ChangeDetectionStrategy, Component, computed, inject, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { LockKeyhole, LockKeyholeOpen, LucideAngularModule } from 'lucide-angular';
import { ApiError } from '../../core/api/api-error';
import { CajaApi, SesionCaja } from '../../core/api/caja.api';
import { Monto } from '../../core/api/modelos';
import { MetodoPago } from '../../core/api/ventas.api';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe } from '../../shared/formato/dinero';
import { FechaPipe } from '../../shared/formato/fecha';
import { CampoComponent } from '../../shared/ui/campo.component';
import { InputDineroComponent } from '../../shared/ui/input-dinero.component';
import { ModalComponent } from '../../shared/ui/modal.component';
import { ArqueoTablaComponent } from './arqueo-tabla.component';
import { claseDiferencia, diferenciaEnVivo, NOMBRE_MEDIO, textoDiferencia, totalesFormulario, validarContado } from './arqueo';
import { EditarCajaComponent } from './editar-caja.component';
import { VentasCajaComponent } from './ventas-caja.component';

/** Caja del momento: abrirla, ver el arqueo en vivo y cerrarla contando cada medio (US-002). */
@Component({
  selector: 'app-caja-actual',
  imports: [FormsModule, LucideAngularModule, DineroPipe, FechaPipe, CampoComponent, InputDineroComponent, ModalComponent,
    ArqueoTablaComponent, EditarCajaComponent, VentasCajaComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (cargando() && !cargado()) {
      <div class="card h-40 animate-pulse" aria-label="Cargando"></div>
    } @else if (sesion(); as s) {
      <section class="card p-5" aria-labelledby="caja-abierta-titulo">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h2 id="caja-abierta-titulo" class="flex items-center gap-2 text-lg font-semibold">
              <lucide-icon [img]="iconoAbierta" [size]="20" class="text-ok" aria-hidden="true" /> Caja abierta
            </h2>
            <p class="mt-1 text-sm text-muted">
              Desde {{ s.abiertaEn | fecha }} · Inicial {{ s.montoInicial | dinero }} ·
              {{ s.cantidadVentas }} {{ s.cantidadVentas === 1 ? 'venta' : 'ventas' }}
            </p>
            @if (s.notas) { <p class="mt-1 text-sm">{{ s.notas }}</p> }
          </div>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn-fantasma" (click)="viendoVentas.set(s)">Ver ventas</button>
            <button type="button" class="btn-secundario" (click)="editando.set(s)">Editar</button>
            <button type="button" class="btn-primario" (click)="abrirCierre(s)">
              <lucide-icon [img]="iconoCerrada" [size]="18" aria-hidden="true" /> Cerrar caja
            </button>
          </div>
        </div>
        <h3 class="mt-5 mb-2 text-sm font-semibold">Lo que debería haber según el sistema</h3>
        <app-arqueo-tabla [sesion]="s" />
        <button type="button" class="btn-fantasma mt-2 min-h-8 px-3 text-sm" (click)="cargar()">Actualizar</button>
      </section>
    } @else {
      <section class="card mx-auto max-w-md p-6" aria-labelledby="caja-cerrada-titulo">
        <h2 id="caja-cerrada-titulo" class="flex items-center gap-2 text-lg font-semibold">
          <lucide-icon [img]="iconoCerrada" [size]="20" aria-hidden="true" /> La caja está cerrada
        </h2>
        <p class="mt-1 mb-4 text-sm text-muted">Abrila para empezar a vender. Ingresá con cuánto efectivo arrancás.</p>
        <form class="grid gap-4" (ngSubmit)="abrir()">
          <app-campo etiqueta="Efectivo inicial" para="ca-inicial" [error]="errores()['montoInicial']">
            <app-input-dinero idCampo="ca-inicial" name="montoInicial" [(ngModel)]="montoInicial" [invalido]="!!errores()['montoInicial']"
                              [describedBy]="errores()['montoInicial'] ? 'ca-inicial-error' : ''" />
          </app-campo>
          <app-campo etiqueta="Notas" para="ca-notas" [opcional]="true" [error]="errores()['notas']">
            <input id="ca-notas" name="notas" class="input" maxlength="500" [(ngModel)]="notasApertura" />
          </app-campo>
          <button type="submit" class="btn-primario" [disabled]="guardando()">
            <lucide-icon [img]="iconoAbierta" [size]="18" aria-hidden="true" /> Abrir caja
          </button>
        </form>
      </section>
    }

    <app-modal [abierto]="!!cerrando()" titulo="Cerrar caja" ancho="grande" (cerrar)="cerrando.set(null)">
      @if (cerrando(); as s) {
        <form id="form-cierre" (ngSubmit)="cerrar(s)">
          <p class="mb-4 text-sm text-muted">Contá lo que hay en cada medio (el cajón, la cuenta, el QR y el posnet) y anotalo. Si no hubo nada, poné 0.</p>
          <div class="overflow-x-auto">
            <table class="tabla">
              <caption class="sr-only">Conteo por medio de pago</caption>
              <thead>
                <tr><th scope="col">Medio</th><th scope="col" class="text-right">Según el sistema</th>
                  <th scope="col">Contado</th><th scope="col" class="text-right">Diferencia</th></tr>
              </thead>
              <tbody>
                @for (a of s.arqueo; track a.metodo) {
                  <tr>
                    <th scope="row" class="font-medium">{{ nombre[a.metodo] }}</th>
                    <td class="text-right tabular-nums">{{ a.esperado | dinero }}</td>
                    <td class="min-w-36">
                      <label [for]="'cc-' + a.metodo" class="sr-only">Contado en {{ nombre[a.metodo] }}</label>
                      <app-input-dinero [idCampo]="'cc-' + a.metodo" [ngModelOptions]="{ standalone: true }" [ngModel]="contado()[a.metodo]"
                                        (ngModelChange)="cambiarContado(a.metodo, $event)" [invalido]="!!errores()['contado.' + a.metodo]"
                                        [describedBy]="errores()['contado.' + a.metodo] ? 'cc-' + a.metodo + '-error' : ''" />
                      @if (errores()['contado.' + a.metodo]; as e) {
                        <p [id]="'cc-' + a.metodo + '-error'" class="mt-1 text-xs text-danger" role="alert">{{ e }}</p>
                      }
                    </td>
                    <td class="text-right">
                      @let dif = diferencia(a.esperado, a.metodo);
                      <span [class]="clase(dif)">{{ texto(dif) }}</span>
                    </td>
                  </tr>
                }
              </tbody>
              <tfoot>
                <tr class="border-t border-ink-200 font-semibold dark:border-ink-700">
                  <th scope="row">Total</th>
                  <td class="text-right tabular-nums">{{ totales().esperado | dinero }}</td>
                  <td class="text-right tabular-nums">{{ totales().contado | dinero }}</td>
                  <td class="text-right"><span [class]="clase(totales().diferencia)">{{ texto(totales().diferencia) }}</span></td>
                </tr>
              </tfoot>
            </table>
          </div>
          <app-campo class="mt-4" etiqueta="Notas" para="cc-notas" [opcional]="true" [error]="errores()['notas']">
            <textarea id="cc-notas" name="notas" class="input min-h-16 py-2" maxlength="500" [(ngModel)]="notasCierre"
                      placeholder="Ej.: faltó cambio, se pagó un flete sin cargar"></textarea>
          </app-campo>
        </form>
      }
      <div pie>
        <button type="button" class="btn-secundario" (click)="cerrando.set(null)">Cancelar</button>
        <button type="submit" form="form-cierre" class="btn-primario" [disabled]="guardando()">Cerrar caja</button>
      </div>
    </app-modal>

    <app-editar-caja [sesion]="editando()" (guardado)="editando.set(null); cargar()" (cerrar)="editando.set(null)" />
    <app-ventas-caja [sesion]="viendoVentas()" (cerrar)="viendoVentas.set(null)" />
  `,
})
export class CajaActualComponent {
  private readonly api = inject(CajaApi);
  private readonly toast = inject(ToastService);

  /** Avisa cuando se abre o se cierra la caja (para refrescar el historial). */
  readonly cambio = output<void>();

  protected readonly iconoAbierta = LockKeyholeOpen;
  protected readonly iconoCerrada = LockKeyhole;
  protected readonly nombre = NOMBRE_MEDIO;
  protected readonly texto = textoDiferencia;
  protected readonly clase = claseDiferencia;

  protected readonly sesion = signal<SesionCaja | null>(null);
  protected readonly cargando = signal(false);
  protected readonly cargado = signal(false);
  protected readonly guardando = signal(false);
  protected readonly errores = signal<Record<string, string>>({});
  protected readonly cerrando = signal<SesionCaja | null>(null);
  protected readonly editando = signal<SesionCaja | null>(null);
  protected readonly viendoVentas = signal<SesionCaja | null>(null);
  protected readonly contado = signal<Partial<Record<MetodoPago, Monto | null>>>({});
  protected readonly totales = computed(() => totalesFormulario(this.cerrando()?.arqueo ?? [], this.contado()));
  protected montoInicial: Monto | null = null;
  protected notasApertura = '';
  protected notasCierre = '';

  constructor() {
    void this.cargar();
  }

  async cargar(): Promise<void> {
    this.cargando.set(true);
    try {
      this.sesion.set(await firstValueFrom(this.api.actual()));
      this.cargado.set(true);
    } catch (e) {
      this.toast.error(e);
    } finally {
      this.cargando.set(false);
    }
  }

  protected diferencia(esperado: Monto, m: MetodoPago): Monto | null {
    return diferenciaEnVivo(esperado, this.contado()[m]);
  }

  protected cambiarContado(m: MetodoPago, v: Monto | null): void {
    this.contado.update((c) => ({ ...c, [m]: v }));
  }

  private mostrarError(e: unknown): void {
    const err = ApiError.desde(e);
    if (Object.keys(err.campos).length) {
      this.errores.set({ ...err.campos });
    } else {
      this.toast.error(err);
    }
  }

  protected async abrir(): Promise<void> {
    if (this.montoInicial === null || this.montoInicial.startsWith('-')) {
      this.errores.set({ montoInicial: 'Ingresá con cuánto efectivo abrís (puede ser 0)' });
      return;
    }
    this.errores.set({});
    this.guardando.set(true);
    try {
      this.sesion.set(await firstValueFrom(this.api.abrir(this.montoInicial, this.notasApertura)));
      this.montoInicial = null;
      this.notasApertura = '';
      this.toast.exito('Caja abierta');
      this.cambio.emit();
    } catch (e) {
      this.mostrarError(e);
      if (ApiError.desde(e).codigo === 'caja_ya_abierta') {
        void this.cargar();
      }
    } finally {
      this.guardando.set(false);
    }
  }

  protected async abrirCierre(s: SesionCaja): Promise<void> {
    // Arqueo recién calculado: incluye las ventas hechas desde que se abrió la pantalla.
    await this.cargar();
    const actual = this.sesion();
    if (!actual) {
      return;
    }
    this.contado.set({});
    this.notasCierre = actual.notas ?? s.notas ?? '';
    this.errores.set({});
    this.cerrando.set(actual);
  }

  protected async cerrar(s: SesionCaja): Promise<void> {
    const v = validarContado(this.contado());
    if (v.faltan) {
      this.errores.set(Object.fromEntries(v.faltan.map((m) => ['contado.' + m, 'Ingresá cuánto hay (puede ser 0)'])));
      return;
    }
    this.errores.set({});
    this.guardando.set(true);
    try {
      const cerrada = await firstValueFrom(this.api.cerrar(s.id, s.version, v.contado, this.notasCierre));
      this.cerrando.set(null);
      this.sesion.set(null);
      const dif = cerrada.diferencia;
      this.toast.exito(`Caja cerrada · ${textoDiferencia(dif)}`);
      this.cambio.emit();
    } catch (e) {
      this.mostrarError(e);
    } finally {
      this.guardando.set(false);
    }
  }
}
