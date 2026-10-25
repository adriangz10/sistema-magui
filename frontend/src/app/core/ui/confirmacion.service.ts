import { Injectable, signal } from '@angular/core';

export interface PedidoConfirmacion {
  titulo: string;
  mensaje: string;
  confirmar?: string;
  cancelar?: string;
  peligro?: boolean;
}

interface PedidoActivo extends PedidoConfirmacion {
  responder: (ok: boolean) => void;
}

/** Diálogo "¿Estás segura?" que se usa con await: `if (await confirmacion.preguntar(...))`. */
@Injectable({ providedIn: 'root' })
export class ConfirmacionService {
  readonly activo = signal<PedidoActivo | null>(null);

  preguntar(pedido: PedidoConfirmacion): Promise<boolean> {
    this.activo()?.responder(false);
    return new Promise((resolve) => {
      this.activo.set({
        ...pedido,
        responder: (ok) => {
          this.activo.set(null);
          resolve(ok);
        },
      });
    });
  }
}
