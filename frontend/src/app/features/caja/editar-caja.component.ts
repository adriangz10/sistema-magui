import { ChangeDetectionStrategy, Component, effect, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ApiError } from '../../core/api/api-error';
import { CajaApi, SesionCaja, SesionCajaDetalle } from '../../core/api/caja.api';
import { Monto } from '../../core/api/modelos';
import { MetodoPago } from '../../core/api/ventas.api';
import { ToastService } from '../../core/ui/toast.service';
import { DineroPipe } from '../../shared/formato/dinero';
import { CampoComponent } from '../../shared/ui/campo.component';
import { InputDineroComponent } from '../../shared/ui/input-dinero.component';
import { ModalComponent } from '../../shared/ui/modal.component';
import { NOMBRE_MEDIO, validarContado } from './arqueo';

/** Edita monto inicial, contado y notas de una caja; pide motivo y queda en el historial (US-002, D-3). */
@Component({
  selector: 'app-editar-caja',
  imports: [FormsModule, DineroPipe, CampoComponent, InputDineroComponent, ModalComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-modal [abierto]="!!sesion()" titulo="Editar caja" (cerrar)="cerrar.emit()">
      @if (sesion(); as s) {
        <form id="form-editar-caja" class="grid gap-4" (ngSubmit)="guardar(s)">
          <p class="text-sm text-muted">
            Los horarios de apertura y cierre no se cambian. El cambio queda en el historial de la caja con el motivo.
          </p>
          <app-campo etiqueta="Monto inicial en efectivo" para="ec-inicial" [error]="errores()['montoInicial']">
            <app-input-dinero idCampo="ec-inicial" name="montoInicial" [(ngModel)]="montoInicial"
                              [invalido]="!!errores()['montoInicial']" [describedBy]="errores()['montoInicial'] ? 'ec-inicial-error' : ''" />
          </app-campo>

          @if (!s.abierta) {
            <fieldset class="grid gap-3 sm:grid-cols-2">
              <legend class="etiqueta">Lo que se contó</legend>
              @for (a of s.arqueo; track a.metodo) {
                <app-campo [etiqueta]="nombre[a.metodo]" [para]="'ec-' + a.metodo" [error]="errores()['contado.' + a.metodo]">
                  <app-input-dinero [idCampo]="'ec-' + a.metodo" [ngModelOptions]="{ standalone: true }" [ngModel]="contado()[a.metodo]"
                                    (ngModelChange)="cambiarContado(a.metodo, $event)" [invalido]="!!errores()['contado.' + a.metodo]"
                                    [describedBy]="errores()['contado.' + a.metodo] ? 'ec-' + a.metodo + '-error' : 'ec-' + a.metodo + '-ayuda'" />
                  <p [id]="'ec-' + a.metodo + '-ayuda'" class="mt-1 text-xs text-muted">Sistema al cierre: {{ a.esperado | dinero }}</p>
                </app-campo>
              }
            </fieldset>
          }

          <app-campo etiqueta="Notas" para="ec-notas" [opcional]="true" [error]="errores()['notas']">
            <textarea id="ec-notas" name="notas" class="input min-h-16 py-2" maxlength="500" [(ngModel)]="notas"></textarea>
          </app-campo>
          <app-campo etiqueta="Motivo del cambio" para="ec-motivo" [error]="errores()['motivo']">
            <input id="ec-motivo" name="motivo" class="input" maxlength="255" [(ngModel)]="motivo" placeholder="Ej.: me equivoqué al contar"
                   [attr.aria-invalid]="errores()['motivo'] ? true : null" [attr.aria-describedby]="errores()['motivo'] ? 'ec-motivo-error' : null" />
          </app-campo>
        </form>
      }
      <div pie>
        <button type="button" class="btn-secundario" (click)="cerrar.emit()">Cancelar</button>
        <button type="submit" form="form-editar-caja" class="btn-primario" [disabled]="guardando()">Guardar cambios</button>
      </div>
    </app-modal>
  `,
})
export class EditarCajaComponent {
  private readonly api = inject(CajaApi);
  private readonly toast = inject(ToastService);

  /** Caja a editar; null = cerrado. */
  readonly sesion = input<SesionCaja | null>(null);
  readonly guardado = output<SesionCajaDetalle>();
  readonly cerrar = output<void>();

  protected readonly nombre = NOMBRE_MEDIO;
  protected readonly guardando = signal(false);
  protected readonly errores = signal<Record<string, string>>({});
  protected readonly contado = signal<Partial<Record<MetodoPago, Monto | null>>>({});
  protected montoInicial: Monto | null = null;
  protected notas = '';
  protected motivo = '';

  constructor() {
    effect(() => {
      const s = this.sesion();
      if (!s) {
        return;
      }
      this.montoInicial = s.montoInicial;
      this.notas = s.notas ?? '';
      this.motivo = '';
      this.contado.set(Object.fromEntries(s.arqueo.map((a) => [a.metodo, a.contado])));
      this.errores.set({});
    });
  }

  protected cambiarContado(m: MetodoPago, v: Monto | null): void {
    this.contado.update((c) => ({ ...c, [m]: v }));
  }

  protected async guardar(s: SesionCaja): Promise<void> {
    const locales: Record<string, string> = {};
    if (this.montoInicial === null || this.montoInicial.startsWith('-')) {
      locales['montoInicial'] = 'Ingresá un monto (puede ser 0)';
    }
    if (!this.motivo.trim()) {
      locales['motivo'] = 'Contá por qué se edita la caja';
    }
    const v = s.abierta ? null : validarContado(this.contado());
    for (const m of v?.faltan ?? []) {
      locales['contado.' + m] = 'Ingresá cuánto hay (puede ser 0)';
    }
    this.errores.set(locales);
    if (Object.keys(locales).length) {
      return;
    }
    this.guardando.set(true);
    try {
      const det = await firstValueFrom(this.api.editar(s.id, {
        version: s.version, motivo: this.motivo, montoInicial: this.montoInicial!, notas: this.notas,
        contado: v?.contado,
      }));
      this.toast.exito('Caja actualizada');
      this.guardado.emit(det);
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
