import { DestroyRef, Injectable, effect, inject, signal } from '@angular/core';
import { API_BASE_URL } from '../config/api.config';

/**
 * Indica si hay conexión con el servidor (indicador "En línea / Sin conexión").
 * Combina el aviso del navegador (poco confiable: dice "en línea" con un wifi sin internet)
 * con el resultado real de las llamadas a la API, que informa conexionInterceptor.
 */
@Injectable({ providedIn: 'root' })
export class ConexionService {
  readonly enLinea = signal(globalThis.navigator?.onLine ?? true);

  constructor() {
    const actualizar = () => this.enLinea.set(globalThis.navigator?.onLine ?? true);
    globalThis.addEventListener?.('online', actualizar);
    globalThis.addEventListener?.('offline', actualizar);
    // /health nunca sale de la copia del service worker: sirve para saber si hay servidor.
    const base = inject(API_BASE_URL);
    const probar = () =>
      fetch(`${base}/health`, { cache: 'no-store' }).then((r) => this.informar(r.ok)).catch(() => this.informar(false));
    void probar(); // al abrir la app (puede abrir desde la copia, sin señal)
    // Sin conexión: se vuelve a probar cada 15 s para pasar a "En línea" apenas vuelva.
    let reintento: ReturnType<typeof setInterval> | undefined;
    effect(() => {
      clearInterval(reintento);
      if (!this.enLinea()) {
        reintento = setInterval(() => void probar(), 15000);
      }
    });
    inject(DestroyRef).onDestroy(() => {
      clearInterval(reintento);
      globalThis.removeEventListener?.('online', actualizar);
      globalThis.removeEventListener?.('offline', actualizar);
    });
  }

  /** Lo llama el interceptor: el servidor respondió (true) o no se pudo llegar (false). */
  informar(hayConexion: boolean): void {
    if (this.enLinea() !== hayConexion) {
      this.enLinea.set(hayConexion);
    }
  }
}
