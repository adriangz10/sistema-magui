import { Injectable, computed, signal } from '@angular/core';

/** Evento que Chrome/Edge/Android disparan cuando la app se puede instalar. */
interface EventoInstalacion extends Event {
  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

/**
 * Instalación de la app en el dispositivo (ícono en la pantalla de inicio, pantalla completa).
 * En Android/Chrome/Edge se ofrece un botón; en iPhone hay que hacerlo desde "Compartir".
 */
@Injectable({ providedIn: 'root' })
export class InstalacionService {
  private evento: EventoInstalacion | null = null;
  private readonly disponible = signal(false);

  /** Ya se está usando como app instalada. */
  readonly instalada = signal(this.modoApp());
  /** Se puede mostrar el botón "Instalar". */
  readonly puedeInstalar = computed(() => this.disponible() && !this.instalada());
  /** iPhone/iPad en Safari: no hay botón, se muestran instrucciones. */
  readonly esIOS = /iphone|ipad|ipod/i.test(globalThis.navigator?.userAgent ?? '') && !this.modoApp();

  constructor() {
    globalThis.addEventListener?.('beforeinstallprompt', (e) => {
      e.preventDefault(); // se ofrece desde nuestro botón, no con el cartel del navegador
      this.evento = e as EventoInstalacion;
      this.disponible.set(true);
    });
    globalThis.addEventListener?.('appinstalled', () => {
      this.instalada.set(true);
      this.disponible.set(false);
    });
  }

  /** Abre el diálogo de instalación del navegador. Devuelve true si se aceptó. */
  async instalar(): Promise<boolean> {
    if (!this.evento) {
      return false;
    }
    await this.evento.prompt();
    const { outcome } = await this.evento.userChoice;
    this.evento = null;
    this.disponible.set(false);
    return outcome === 'accepted';
  }

  private modoApp(): boolean {
    const nav = globalThis.navigator as Navigator & { standalone?: boolean };
    return !!globalThis.matchMedia?.('(display-mode: standalone)').matches || nav?.standalone === true;
  }
}
