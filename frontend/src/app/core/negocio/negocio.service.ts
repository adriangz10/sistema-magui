import { Injectable, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { ConfigTicket, Marca, Negocio, NegocioApi } from '../api/negocio.api';
import { almacenamiento } from '../almacenamiento';

const CLAVE = 'magui.negocio';

export const TICKET_POR_DEFECTO: ConfigTicket = {
  habilitado: false, imprimirAlCobrar: false, anchoPapel: 80, mostrarLogo: true, datos: null, pie: '¡Gracias por su compra!',
};

const POR_DEFECTO: Negocio = { nombre: 'Magui', tieneLogo: false, logoVersion: 0, ticket: TICKET_POR_DEFECTO };

function leerCopia(): Negocio {
  try {
    const v = JSON.parse(almacenamiento.leer(CLAVE) ?? 'null') as Partial<Negocio> | null;
    return v?.nombre ? { ...POR_DEFECTO, ...v, ticket: { ...TICKET_POR_DEFECTO, ...v.ticket } } : POR_DEFECTO;
  } catch {
    return POR_DEFECTO;
  }
}

/**
 * Nombre, logo y configuración del ticket (US-003). Guarda una copia local para mostrarlos
 * e imprimir tickets aunque no haya conexión (app de escritorio).
 */
@Injectable({ providedIn: 'root' })
export class NegocioService {
  private readonly api = inject(NegocioApi);
  private readonly estado = signal<Negocio>(leerCopia());

  readonly negocio = this.estado.asReadonly();
  readonly nombre = computed(() => this.estado().nombre);
  readonly ticket = computed(() => this.estado().ticket);
  /** URL del logo, o null si no hay. */
  readonly logo = computed(() => (this.estado().tieneLogo ? this.api.urlLogo(this.estado().logoVersion) : null));
  /** Inicial para cuando no hay logo. */
  readonly inicial = computed(() => this.estado().nombre.trim().charAt(0).toUpperCase() || 'M');

  /** Lo público (sirve antes de ingresar). Sin conexión se queda con la copia local. */
  async cargarMarca(): Promise<void> {
    try {
      this.actualizar(await firstValueFrom(this.api.marca()));
    } catch {
      // sin conexión: copia local
    }
  }

  /** Todo, incluido el ticket (requiere sesión). */
  async cargar(): Promise<void> {
    try {
      this.actualizar(await firstValueFrom(this.api.obtener()));
    } catch {
      // sin conexión: copia local
    }
  }

  actualizar(cambio: Marca | Negocio): void {
    this.estado.update((n) => ({ ...n, ...cambio }));
    almacenamiento.guardar(CLAVE, JSON.stringify(this.estado()));
  }
}
