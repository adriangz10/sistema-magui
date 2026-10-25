import { Injectable, inject } from '@angular/core';
import { SwUpdate, VersionReadyEvent } from '@angular/service-worker';
import { filter } from 'rxjs';
import { ConfirmacionService } from '../ui/confirmacion.service';

const CADA_HORA = 60 * 60 * 1000;

/**
 * Cuando se publica una versión nueva del sistema, la app instalada la descarga sola y
 * pregunta si se quiere actualizar ahora (no interrumpe una venta a mitad de camino).
 */
@Injectable({ providedIn: 'root' })
export class ActualizacionService {
  private readonly sw = inject(SwUpdate);
  private readonly confirmacion = inject(ConfirmacionService);

  iniciar(): void {
    if (!this.sw.isEnabled) {
      return;
    }
    this.sw.versionUpdates
      .pipe(filter((e): e is VersionReadyEvent => e.type === 'VERSION_READY'))
      .subscribe(() => void this.ofrecer());
    this.sw.unrecoverable.subscribe(() => document.location.reload());
    setInterval(() => void this.sw.checkForUpdate().catch(() => undefined), CADA_HORA);
  }

  private async ofrecer(): Promise<void> {
    const ok = await this.confirmacion.preguntar({
      titulo: 'Hay una versión nueva',
      mensaje: 'Se descargó una actualización del sistema. ¿La aplicamos ahora? La pantalla se va a recargar.',
      confirmar: 'Actualizar',
      cancelar: 'Más tarde',
    });
    if (ok) {
      await this.sw.activateUpdate();
      document.location.reload();
    }
  }
}
