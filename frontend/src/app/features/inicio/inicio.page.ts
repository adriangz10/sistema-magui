import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { HealthService } from '../../core/api/health.service';

type Estado = 'cargando' | 'ok' | 'error';

/** Pantalla provisoria de la Fase 0: confirma que el frontend habla con el backend. */
@Component({
  selector: 'app-inicio',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <main class="mx-auto flex min-h-full max-w-xl flex-col items-center justify-center gap-6 p-6 text-center">
      <div class="flex size-16 items-center justify-center rounded-2xl bg-brand-600 text-2xl font-bold text-white shadow-lg">
        M
      </div>
      <div>
        <h1 class="text-3xl font-semibold tracking-tight">Sistema Magui</h1>
        <p class="mt-2 text-ink-500">Punto de venta, stock y estadísticas</p>
      </div>

      <div class="w-full rounded-xl border border-ink-200 bg-white p-4 shadow-sm dark:border-ink-700 dark:bg-ink-900">
        <p class="text-sm font-medium text-ink-500">Estado del servidor</p>
        @switch (estado()) {
          @case ('cargando') {
            <p class="mt-1 text-ink-700">Conectando…</p>
          }
          @case ('ok') {
            <p class="mt-1 font-medium text-ok" role="status">● En línea · versión {{ version() }}</p>
          }
          @case ('error') {
            <p class="mt-1 font-medium text-danger" role="alert">● Sin conexión con el servidor</p>
          }
        }
      </div>
    </main>
  `,
})
export class InicioPage {
  private readonly health = inject(HealthService);

  protected readonly estado = signal<Estado>('cargando');
  protected readonly version = signal('');

  constructor() {
    this.health.check().subscribe({
      next: (h) => {
        this.version.set(h.version);
        this.estado.set('ok');
      },
      error: () => this.estado.set('error'),
    });
  }
}
