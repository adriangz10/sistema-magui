import { DOCUMENT } from '@angular/common';
import { Injectable, inject, signal } from '@angular/core';
import { VentaDetalle } from '../api/ventas.api';
import { NegocioService } from '../negocio/negocio.service';

const CLASE = 'imprimiendo-ticket';

/** Espera a que el navegador dibuje el ticket antes de imprimir. */
const dosCuadros = () => new Promise<void>((ok) => requestAnimationFrame(() => requestAnimationFrame(() => ok())));

/**
 * Imprime el ticket de una venta (US-003) con la impresora del sistema: una ticketera térmica
 * instalada en Windows (58 u 80 mm) o cualquier otra. Mientras imprime, en la hoja solo aparece el ticket.
 */
@Injectable({ providedIn: 'root' })
export class ImpresionTicketService {
  private readonly documento = inject(DOCUMENT);
  private readonly negocio = inject(NegocioService);

  /** Venta que se está imprimiendo (la dibuja <app-ticket-impresion> en la raíz de la app). */
  readonly venta = signal<VentaDetalle | null>(null);

  /** Si los tickets están habilitados en Configuración. */
  get habilitado(): boolean {
    return this.negocio.ticket().habilitado;
  }

  async imprimir(venta: VentaDetalle): Promise<void> {
    this.venta.set(venta);
    await this.precargarLogo();
    await dosCuadros();
    const raiz = this.documento.documentElement;
    raiz.classList.add(CLASE);
    try {
      this.documento.defaultView?.print(); // bloquea hasta que se imprime o se cancela
    } finally {
      raiz.classList.remove(CLASE);
      this.venta.set(null);
    }
  }

  /** Si el logo no está en caché, el ticket saldría sin logo: se espera (como mucho 2 s). */
  private async precargarLogo(): Promise<void> {
    const url = this.negocio.logo();
    if (!url || !this.negocio.ticket().mostrarLogo) {
      return;
    }
    const img = new Image();
    img.src = url;
    await Promise.race([img.decode().catch(() => undefined), new Promise((ok) => setTimeout(ok, 2000))]);
  }
}
