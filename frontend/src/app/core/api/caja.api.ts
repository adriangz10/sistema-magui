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

/** Cliente de gastos y retiros de caja. */
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
}
