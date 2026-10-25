import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { HealthService } from '../../core/api/health.service';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';

type Estado = 'cargando' | 'ok' | 'error';

/** Inicio provisorio: el tablero con estadísticas y alertas llega en la Fase 8. */
@Component({
  selector: 'app-inicio',
  imports: [EncabezadoComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-encabezado titulo="¡Hola!" descripcion="Bienvenida al Sistema Magui" />
    <div class="card p-5">
      <p class="text-sm font-medium text-ink-500">Estado del servidor</p>
      @switch (estado()) {
        @case ('cargando') {
          <p class="mt-1 text-ink-700 dark:text-ink-300">Conectando…</p>
        }
        @case ('ok') {
          <p class="mt-1 font-medium text-ok dark:text-green-400" role="status">● En línea · versión {{ version() }}</p>
        }
        @case ('error') {
          <p class="mt-1 font-medium text-danger dark:text-red-400" role="alert">● Sin conexión con el servidor</p>
        }
      }
    </div>
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
