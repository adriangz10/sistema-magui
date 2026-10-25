import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';
import { FechaISO, Monto, parametros } from './modelos';

export type TipoPeriodo = 'dia' | 'semana' | 'mes' | 'anio' | 'rango';

export interface ConsultaPeriodo {
  periodo: TipoPeriodo;
  desde?: FechaISO;
  hasta?: FechaISO;
}

export interface Resumen {
  periodo: { tipo: TipoPeriodo; desde: string; hasta: string };
  ventas: {
    cantidad: number;
    total: Monto;
    costo: Monto;
    ganancia: Monto;
    margen: number;
    ticketPromedio: Monto;
    anuladas: number;
    renglonesSinCosto: number;
  };
  porMetodo: { metodo: string; monto: Monto; cantidad: number; porcentaje: number }[];
  compras: { cantidad: number; total: Monto };
  gastos: { cantidad: number; total: Monto };
  gananciaNeta: Monto;
  efectivoNeto: Monto;
  anterior: { total: Monto; ganancia: Monto; variacionTotal: number | null; variacionGanancia: number | null };
}

export interface VentaDia {
  fecha: FechaISO;
  cantidad: number;
  total: Monto;
  ganancia: Monto;
}

export interface VentaHora {
  hora: number;
  cantidad: number;
  total: Monto;
}

export interface ProductoVendido {
  productoId: string;
  sku: string;
  nombre: string;
  tipo: string;
  unidades: number;
  total: Monto;
  ganancia: Monto | null;
}

/** Cliente de estadísticas (solo lectura). */
@Injectable({ providedIn: 'root' })
export class EstadisticasApi {
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_BASE_URL);

  resumen(c: ConsultaPeriodo): Observable<Resumen> {
    return this.http.get<Resumen>(`${this.base}/estadisticas/resumen`, { params: parametros({ ...c }) });
  }

  ventasPorDia(c: ConsultaPeriodo): Observable<VentaDia[]> {
    return this.http.get<VentaDia[]>(`${this.base}/estadisticas/ventas-por-dia`, { params: parametros({ ...c }) });
  }

  ventasPorHora(c: ConsultaPeriodo): Observable<VentaHora[]> {
    return this.http.get<VentaHora[]>(`${this.base}/estadisticas/ventas-por-hora`, { params: parametros({ ...c }) });
  }

  masVendidos(c: ConsultaPeriodo, limite = 10, orden: 'unidades' | 'importe' = 'unidades'): Observable<ProductoVendido[]> {
    return this.http.get<ProductoVendido[]>(`${this.base}/estadisticas/mas-vendidos`, { params: parametros({ ...c, limite, orden }) });
  }
}
