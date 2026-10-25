import { Injectable, signal } from '@angular/core';
import { ApiError } from '../api/api-error';

export type TipoToast = 'exito' | 'error' | 'info';

export interface Toast {
  id: number;
  tipo: TipoToast;
  mensaje: string;
}

/** Avisos breves en la esquina de la pantalla. */
@Injectable({ providedIn: 'root' })
export class ToastService {
  private siguiente = 1;
  readonly toasts = signal<Toast[]>([]);

  exito(mensaje: string): void {
    this.mostrar('exito', mensaje, 3500);
  }

  info(mensaje: string): void {
    this.mostrar('info', mensaje, 4000);
  }

  /** Muestra un error; acepta errores de la API o cualquier otro. */
  error(err: unknown): void {
    const mensaje = typeof err === 'string' ? err : ApiError.desde(err).detalle;
    this.mostrar('error', mensaje, 6000);
  }

  cerrar(id: number): void {
    this.toasts.update((ts) => ts.filter((t) => t.id !== id));
  }

  private mostrar(tipo: TipoToast, mensaje: string, ms: number): void {
    const id = this.siguiente++;
    this.toasts.update((ts) => [...ts.slice(-3), { id, tipo, mensaje }]);
    setTimeout(() => this.cerrar(id), ms);
  }
}
