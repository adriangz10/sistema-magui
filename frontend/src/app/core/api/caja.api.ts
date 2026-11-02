import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';
import { FechaISO, Monto, Pagina, parametros } from './modelos';
import { MetodoPago } from './ventas.api';

export interface Gasto {
  id: string;
  fecha: string;
  monto: Monto;
  motivo: string;
  categoria: string | null;
  metodo: MetodoPago;
  anulado: boolean;
  anuladoEn: string | null;
  motivoAnulacion: string | null;
}

export interface ListadoGastos extends Pagina<Gasto> {
  /** Suma de todos los gastos filtrados (sin anulados), no solo de la página. */
  totalMonto: Monto;
}

export interface NuevoGasto {
  /** Vacío = ahora. */
  fecha: FechaISO | '';
  monto: Monto;
  motivo: string;
  categoria: string;
  metodo: MetodoPago;
}

// --- Apertura y cierre de caja (US-002) ---

/** Lo que dice el sistema contra lo contado, para un medio de pago. */
export interface Arqueo {
  metodo: MetodoPago;
  ventas: Monto;
  gastos: Monto;
  /** Ventas − gastos (en efectivo, más el monto inicial). */
  esperado: Monto;
  /** null mientras la caja está abierta. */
  contado: Monto | null;
  /** contado − esperado: positiva sobra, negativa falta. */
  diferencia: Monto | null;
}

export interface SesionCaja {
  id: string;
  abiertaEn: string;
  cerradaEn: string | null;
  abierta: boolean;
  montoInicial: Monto;
  notas: string | null;
  version: number;
  cantidadVentas: number;
  /** Abierta: en vivo y sin contado. Cerrada: la foto del cierre (o de la última edición). */
  arqueo: Arqueo[];
  totalEsperado: Monto;
  totalContado: Monto | null;
  diferencia: Monto | null;
  editada: boolean;
}

export interface CambioCaja {
  /** montoInicial, notas, contado.EFECTIVO, esperado.QR, ... */
  campo: string;
  antes: string | null;
  despues: string | null;
}

export interface EdicionCaja {
  id: string;
  fecha: string;
  motivo: string;
  version: number;
  cambios: CambioCaja[];
}

export interface SesionCajaDetalle extends SesionCaja {
  ediciones: EdicionCaja[];
}

export type ContadoPorMedio = Record<MetodoPago, Monto>;

export interface EdicionCajaCmd {
  version: number;
  motivo: string;
  montoInicial: Monto;
  /** Solo si la caja está cerrada. */
  contado?: ContadoPorMedio;
  notas: string;
}

/** Cliente de gastos, retiros y apertura/cierre de caja. */
@Injectable({ providedIn: 'root' })
export class CajaApi {
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_BASE_URL);

  listar(f: { desde?: FechaISO; hasta?: FechaISO; categoria?: string; incluirAnulados?: boolean; page?: number }): Observable<ListadoGastos> {
    return this.http.get<ListadoGastos>(`${this.base}/caja/gastos`, { params: parametros({ ...f }) });
  }

  registrar(g: NuevoGasto): Observable<Gasto> {
    return this.http.post<Gasto>(`${this.base}/caja/gastos`, g);
  }

  anular(id: string, motivo: string): Observable<Gasto> {
    return this.http.post<Gasto>(`${this.base}/caja/gastos/${id}/anular`, { motivo });
  }

  categorias(): Observable<string[]> {
    return this.http.get<string[]>(`${this.base}/caja/categorias`);
  }

  /** Caja abierta con el arqueo en vivo, o null si está cerrada. */
  actual(): Observable<SesionCaja | null> {
    return this.http.get<SesionCaja | null>(`${this.base}/caja/sesiones/actual`);
  }

  sesiones(page = 1): Observable<Pagina<SesionCaja>> {
    return this.http.get<Pagina<SesionCaja>>(`${this.base}/caja/sesiones`, { params: parametros({ page }) });
  }

  sesion(id: string): Observable<SesionCajaDetalle> {
    return this.http.get<SesionCajaDetalle>(`${this.base}/caja/sesiones/${id}`);
  }

  abrir(montoInicial: Monto, notas: string): Observable<SesionCaja> {
    return this.http.post<SesionCaja>(`${this.base}/caja/sesiones`, { montoInicial, notas });
  }

  cerrar(id: string, version: number, contado: ContadoPorMedio, notas: string): Observable<SesionCaja> {
    return this.http.post<SesionCaja>(`${this.base}/caja/sesiones/${id}/cerrar`, { version, contado, notas });
  }

  editar(id: string, cmd: EdicionCajaCmd): Observable<SesionCajaDetalle> {
    return this.http.patch<SesionCajaDetalle>(`${this.base}/caja/sesiones/${id}`, cmd);
  }
}
