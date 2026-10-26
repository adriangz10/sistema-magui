import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ApiError } from '../../core/api/api-error';
import { InventarioApi } from '../../core/api/inventario.api';
import { AuthService } from '../../core/auth/auth.service';
import { API_BASE_URL } from '../../core/config/api.config';
import { ES_ESCRITORIO } from '../../core/config/plataforma';
import { InstalacionService } from '../../core/pwa/instalacion.service';
import { TemaService } from '../../core/ui/tema.service';
import { ToastService } from '../../core/ui/toast.service';
import { CampoComponent } from '../../shared/ui/campo.component';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';

/** Contraseña, aviso de vencimientos, apariencia y datos del sistema. */
@Component({
  selector: 'app-configuracion',
  imports: [FormsModule, CampoComponent, EncabezadoComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-encabezado titulo="Configuración" />

    <div class="grid items-start gap-6 lg:grid-cols-2">
      <section class="card p-5" aria-labelledby="titulo-clave">
        <h2 id="titulo-clave" class="font-semibold">Cambiar contraseña</h2>
        <p class="mb-4 text-sm text-ink-500">Al cambiarla se cierra la sesión en los demás dispositivos.</p>
        <form class="space-y-4" (ngSubmit)="cambiarClave()">
          <app-campo etiqueta="Contraseña actual" para="cl-actual" [error]="errores()['actual']">
            <input id="cl-actual" name="actual" type="password" class="input" autocomplete="current-password" [(ngModel)]="actual"
                   [attr.aria-invalid]="errores()['actual'] ? true : null" [attr.aria-describedby]="errores()['actual'] ? 'cl-actual-error' : null" />
          </app-campo>
          <app-campo etiqueta="Contraseña nueva" para="cl-nueva" [error]="errores()['nueva']" ayuda="Al menos 8 caracteres.">
            <input id="cl-nueva" name="nueva" type="password" class="input" autocomplete="new-password" [(ngModel)]="nueva"
                   [attr.aria-invalid]="errores()['nueva'] ? true : null" [attr.aria-describedby]="errores()['nueva'] ? 'cl-nueva-error' : 'cl-nueva-ayuda'" />
          </app-campo>
          <app-campo etiqueta="Repetí la contraseña nueva" para="cl-repetir" [error]="errores()['repetir']">
            <input id="cl-repetir" name="repetir" type="password" class="input" autocomplete="new-password" [(ngModel)]="repetir"
                   [attr.aria-invalid]="errores()['repetir'] ? true : null" [attr.aria-describedby]="errores()['repetir'] ? 'cl-repetir-error' : null" />
          </app-campo>
          <div class="flex justify-end"><button type="submit" class="btn-primario" [disabled]="guardandoClave()">Cambiar contraseña</button></div>
        </form>
      </section>

      <div class="space-y-6">
        @if (!esEscritorio) {
        <section class="card p-5" aria-labelledby="titulo-instalar">
          <h2 id="titulo-instalar" class="font-semibold">App en este dispositivo</h2>
          @if (instalacion.instalada()) {
            <p class="mt-1 text-sm text-ok dark:text-green-400">✓ Ya la estás usando como app instalada.</p>
          } @else if (instalacion.puedeInstalar()) {
            <p class="mt-1 mb-3 text-sm text-ink-500">Instalala para abrirla desde un ícono, en pantalla completa, como cualquier app.</p>
            <button type="button" class="btn-primario" (click)="instalar()">Instalar Magui</button>
          } @else if (instalacion.esIOS) {
            <p class="mt-1 text-sm text-ink-500">En iPhone o iPad: tocá <strong>Compartir</strong> (el cuadrado con la flecha) y elegí <strong>Agregar a inicio</strong>.</p>
          } @else {
            <p class="mt-1 text-sm text-ink-500">Desde el celular, abrí el menú del navegador y elegí <strong>Instalar app</strong> o <strong>Agregar a la pantalla de inicio</strong>.</p>
          }
        </section>
        }

        <section class="card p-5" aria-labelledby="titulo-venc">
          <h2 id="titulo-venc" class="font-semibold">Aviso de vencimientos</h2>
          <p class="mb-4 text-sm text-ink-500">Con cuántos días de anticipación se marca un producto como "por vencer".</p>
          <form class="flex items-end gap-2" (ngSubmit)="guardarDias()">
            <app-campo class="flex-1" etiqueta="Días de anticipación" para="cf-dias" [error]="errorDias()">
              <input id="cf-dias" name="dias" type="number" min="1" max="365" inputmode="numeric" class="input" [(ngModel)]="dias"
                     [attr.aria-invalid]="errorDias() ? true : null" [attr.aria-describedby]="errorDias() ? 'cf-dias-error' : null" />
            </app-campo>
            <button type="submit" class="btn-primario" [disabled]="guardandoDias()">Guardar</button>
          </form>
        </section>

        <section class="card p-5" aria-labelledby="titulo-apariencia">
          <h2 id="titulo-apariencia" class="font-semibold">Apariencia</h2>
          <div class="mt-3 flex gap-2" role="group" aria-label="Tema">
            @for (t of temas; track t.id) {
              <button type="button" class="btn flex-1" [attr.aria-pressed]="tema.tema() === t.id"
                      [class]="tema.tema() === t.id ? 'bg-ink-900 text-white dark:bg-ink-100 dark:text-ink-900' : 'ring-1 ring-ink-200 dark:ring-ink-700'"
                      (click)="tema.tema() !== t.id && tema.alternar()">{{ t.etiqueta }}</button>
            }
          </div>
        </section>

        <section class="card p-5 text-sm" aria-labelledby="titulo-sistema">
          <h2 id="titulo-sistema" class="mb-2 font-semibold">Sistema</h2>
          <dl class="space-y-1 text-ink-600 dark:text-ink-300">
            <div class="flex justify-between"><dt>Servidor</dt><dd>{{ estadoServidor() }}</dd></div>
            <div class="flex justify-between"><dt>Versión</dt><dd>{{ version() || '—' }}</dd></div>
          </dl>
        </section>
      </div>
    </div>
  `,
})
export class ConfiguracionPage {
  private readonly auth = inject(AuthService);
  private readonly inventario = inject(InventarioApi);
  private readonly toast = inject(ToastService);
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_BASE_URL);
  protected readonly tema = inject(TemaService);
  protected readonly instalacion = inject(InstalacionService);
  /** En el escritorio la app ya está instalada: no se ofrece instalar la versión web. */
  protected readonly esEscritorio = ES_ESCRITORIO;

  protected readonly temas = [{ id: 'claro' as const, etiqueta: 'Claro' }, { id: 'oscuro' as const, etiqueta: 'Oscuro' }];
  protected actual = '';
  protected nueva = '';
  protected repetir = '';
  protected dias: number | null = null;
  protected readonly errores = signal<Record<string, string>>({});
  protected readonly errorDias = signal('');
  protected readonly guardandoClave = signal(false);
  protected readonly guardandoDias = signal(false);
  protected readonly estadoServidor = signal('Consultando…');
  protected readonly version = signal('');

  constructor() {
    void firstValueFrom(this.inventario.configuracion()).then((c) => (this.dias = c.diasAlertaVencimiento)).catch((e) => this.toast.error(e));
    void firstValueFrom(this.http.get<{ status: string; version: string }>(`${this.base}/health`))
      .then((h) => {
        this.estadoServidor.set(h.status === 'ok' ? 'En línea' : 'Con problemas');
        this.version.set(h.version);
      })
      .catch(() => this.estadoServidor.set('Sin conexión'));
  }

  protected async cambiarClave(): Promise<void> {
    const locales: Record<string, string> = {};
    if (!this.actual) {
      locales['actual'] = 'Ingresá tu contraseña actual';
    }
    if (this.nueva.length < 8) {
      locales['nueva'] = 'Debe tener al menos 8 caracteres';
    }
    if (this.nueva !== this.repetir) {
      locales['repetir'] = 'Las contraseñas no coinciden';
    }
    this.errores.set(locales);
    if (Object.keys(locales).length) {
      return;
    }
    this.guardandoClave.set(true);
    try {
      await this.auth.cambiarClave(this.actual, this.nueva);
      this.actual = this.nueva = this.repetir = '';
      this.toast.exito('Contraseña cambiada. Las otras sesiones se cerraron.');
    } catch (e) {
      const err = ApiError.desde(e);
      if (err.codigo === 'clave_incorrecta') {
        this.errores.set({ actual: err.message });
      } else if (Object.keys(err.campos).length) {
        this.errores.set({ ...err.campos });
      } else {
        this.toast.error(err);
      }
    } finally {
      this.guardandoClave.set(false);
    }
  }

  protected async instalar(): Promise<void> {
    if (await this.instalacion.instalar()) {
      this.toast.exito('¡Listo! Magui quedó instalada en este dispositivo.');
    }
  }

  protected async guardarDias(): Promise<void> {
    this.errorDias.set('');
    const n = Number(this.dias);
    if (!Number.isInteger(n) || n < 1 || n > 365) {
      this.errorDias.set('Entre 1 y 365 días');
      return;
    }
    this.guardandoDias.set(true);
    try {
      await firstValueFrom(this.inventario.guardarConfiguracion(n));
      this.toast.exito(`Los productos se marcarán como "por vencer" ${n} días antes`);
    } catch (e) {
      const err = ApiError.desde(e);
      this.errorDias.set(err.campos['diasAlertaVencimiento'] ?? err.message);
    } finally {
      this.guardandoDias.set(false);
    }
  }
}
