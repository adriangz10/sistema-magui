import { DestroyRef, Injectable, inject, signal } from '@angular/core';

/** Indica si el dispositivo tiene conexión (base del indicador "En línea / Sin conexión"). */
@Injectable({ providedIn: 'root' })
export class ConexionService {
  readonly enLinea = signal(globalThis.navigator?.onLine ?? true);

  constructor() {
    const actualizar = () => this.enLinea.set(globalThis.navigator?.onLine ?? true);
    globalThis.addEventListener?.('online', actualizar);
    globalThis.addEventListener?.('offline', actualizar);
    inject(DestroyRef).onDestroy(() => {
      globalThis.removeEventListener?.('online', actualizar);
      globalThis.removeEventListener?.('offline', actualizar);
    });
  }
}
