import { HttpClient } from '@angular/common/http';
import { DestroyRef, Injectable, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';
import { ES_ESCRITORIO } from '../config/plataforma';
import { ConexionService } from '../red/conexion.service';

/** Estado de la sincronización del escritorio (GET /api/escritorio/estado). */
export interface EstadoEscritorio {
  enLinea: boolean;
  pendientes: number;
  rechazadas: number;
  necesitaIngresar: boolean;
  ultimaSincronizacion: string | null;
  copiaDelCatalogo: string | null;
  ultimoError?: string;
}

/** Cada cuánto se consulta el estado (es local: no gasta internet). */
export const INTERVALO_ESTADO_MS = 5000;

/**
 * Ventas cobradas en esta PC que faltan subir al servidor. Solo en el escritorio:
 * en la web y el celular no hace nada.
 */
@Injectable({ providedIn: 'root' })
export class EscritorioService {
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_BASE_URL);
  private readonly conexion = inject(ConexionService);

  readonly activo = ES_ESCRITORIO;
  readonly estado = signal<EstadoEscritorio | null>(null);
  readonly pendientes = computed(() => this.estado()?.pendientes ?? 0);
  readonly rechazadas = computed(() => this.estado()?.rechazadas ?? 0);
  readonly necesitaIngresar = computed(() => this.estado()?.necesitaIngresar ?? false);

  constructor() {
    if (!this.activo) {
      return;
    }
    void this.actualizar();
    const id = setInterval(() => void this.actualizar(), INTERVALO_ESTADO_MS);
    inject(DestroyRef).onDestroy(() => clearInterval(id));
  }

  /** Consulta el estado ya (por ejemplo, después de cobrar). */
  async actualizar(): Promise<void> {
    try {
      const e = await firstValueFrom(this.http.get<EstadoEscritorio>(`${this.base}/escritorio/estado`));
      this.estado.set(e);
      this.conexion.informar(e.enLinea); // el núcleo sabe si el servidor responde
    } catch {
      // es local: si falla, se mantiene el último estado conocido
    }
  }
}
