import { ChangeDetectionStrategy, Component, inject, input, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { Eye, EyeOff, LoaderCircle, LucideAngularModule } from 'lucide-angular';
import { ApiError } from '../../core/api/api-error';
import { AuthService } from '../../core/auth/auth.service';

/** Pantalla de ingreso con la contraseña de la dueña. */
@Component({
  selector: 'app-ingresar',
  imports: [ReactiveFormsModule, LucideAngularModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <main class="flex min-h-full items-center justify-center bg-ink-50 p-4 dark:bg-ink-950">
      <div class="w-full max-w-sm">
        <div class="mb-8 text-center">
          <div class="mx-auto flex size-12 items-center justify-center rounded-xl bg-ink-900 text-xl font-semibold text-white dark:bg-ink-100 dark:text-ink-900">M</div>
          <h1 class="mt-4 text-2xl font-semibold tracking-tight">Sistema Magui</h1>
          <p class="mt-1 text-sm text-muted">Ingresá para continuar</p>
        </div>

        <form class="card space-y-4 p-6 sm:p-8" [formGroup]="form" (ngSubmit)="ingresar()" novalidate>
          <div>
            <label for="clave" class="etiqueta">Contraseña</label>
            <div class="relative">
              <input id="clave" [type]="mostrar() ? 'text' : 'password'" class="input pr-11" formControlName="clave"
                     autocomplete="current-password" autofocus
                     [attr.aria-invalid]="error() ? true : null" [attr.aria-describedby]="error() ? 'clave-error' : null" />
              <button type="button" class="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-muted hover:text-ink-900 dark:hover:text-ink-100"
                      (click)="mostrar.set(!mostrar())" [attr.aria-label]="mostrar() ? 'Ocultar contraseña' : 'Mostrar contraseña'">
                <lucide-icon [img]="mostrar() ? iconos.ocultar : iconos.ver" [size]="18" aria-hidden="true" />
              </button>
            </div>
            @if (error()) {
              <p id="clave-error" class="mt-2 text-sm text-danger" role="alert">{{ error() }}</p>
            }
          </div>
          <button type="submit" class="btn-primario w-full" [disabled]="enviando()">
            @if (enviando()) {
              <lucide-icon [img]="iconos.cargando" [size]="18" class="animate-spin" aria-hidden="true" />
            }
            Ingresar
          </button>
        </form>
      </div>
    </main>
  `,
})
export class IngresarPage {
  /** Ruta a la que volver después de ingresar (?volver=). */
  readonly volver = input<string>();

  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  protected readonly form = inject(FormBuilder).nonNullable.group({ clave: ['', Validators.required] });
  protected readonly enviando = signal(false);
  protected readonly error = signal('');
  protected readonly mostrar = signal(false);
  protected readonly iconos = { ver: Eye, ocultar: EyeOff, cargando: LoaderCircle };

  protected async ingresar(): Promise<void> {
    if (this.form.invalid || this.enviando()) {
      return;
    }
    this.enviando.set(true);
    this.error.set('');
    try {
      await this.auth.iniciarSesion(this.form.getRawValue().clave);
      const destino = this.volver();
      await this.router.navigateByUrl(destino && destino.startsWith('/') && !destino.startsWith('//') ? destino : '/');
    } catch (e) {
      this.error.set(ApiError.desde(e).message);
      this.form.controls.clave.reset();
    } finally {
      this.enviando.set(false);
    }
  }
}
