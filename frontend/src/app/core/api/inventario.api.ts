import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';
import { FechaISO, Monto, Pagina, parametros } from './modelos';

export type EstadoLote = 'VENCIDO' | 'POR_VENCER' | 'VIGENTE' | 'SIN_FECHA';
export type TipoMovimiento = 'INICIAL' | 'COMPRA' | 'VENTA' | 'ANULACION' | 'AJUSTE';

export const TIPOS_MOVIMIENTO: Record<TipoMovimiento, string> = {
  INICIAL: 'Stock inicial', COMPRA: 'Compra', VENTA: 'Venta', ANULACION: 'Anulación', AJUSTE: 'Ajuste',
};

export interface Lote {
  id: string;
  vencimiento: FechaISO | null;
  costo: Monto | null;
  stock: number;
  estado: EstadoLote;
  diasRestantes: number | null;
}

export interface StockProducto {
  productoId: string;
  sku: string;
  nombre: string;
  stock: number;
  lotes: Lote[];
}

export interface Movimiento {
  id: string;
  fecha: string;
  productoId: string;
  sku: string;
  producto: string;
  cantidad: number;
  tipo: TipoMovimiento;
  referenciaId: string | null;
  nota: string;
  loteVencimiento: FechaISO | null;
}

export interface Vencimiento {
  loteId: string;
  productoId: string;
  sku: string;
  producto: string;
  vencimiento: FechaISO;
  stock: number;
  estado: EstadoLote;
  diasRestantes: number;
}

export interface Alertas {
  vencidos: number;
  porVencer: number;
  sinStock: number;
  stockBajo: number;
  diasAlerta: number;
}

export interface AjusteStock {
  productoId: string;
  cantidad: number;
  motivo: string;
  loteId?: string;
  vencimiento?: FechaISO | null;
  costo?: Monto | null;
}

/** Cliente de inventario: stock por lote, vencimientos, ajustes y movimientos. */
@Injectable({ providedIn: 'root' })
export class InventarioApi {
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_BASE_URL);

  stock(productoId: string): Observable<StockProducto> {
    return this.http.get<StockProducto>(`${this.base}/inventario/productos/${productoId}/stock`);
  }

  movimientos(f: { productoId?: string; tipo?: TipoMovimiento | ''; desde?: FechaISO; hasta?: FechaISO; page?: number }): Observable<Pagina<Movimiento>> {
    return this.http.get<Pagina<Movimiento>>(`${this.base}/inventario/movimientos`, { params: parametros({ ...f }) });
  }

  ajustar(a: AjusteStock): Observable<StockProducto> {
    return this.http.post<StockProducto>(`${this.base}/inventario/ajustes`, a);
  }

  vencimientos(estado: 'vencidos' | 'por_vencer' | 'todos', page = 1): Observable<Pagina<Vencimiento>> {
    return this.http.get<Pagina<Vencimiento>>(`${this.base}/inventario/vencimientos`, { params: { estado, page } });
  }

  alertas(): Observable<Alertas> {
    return this.http.get<Alertas>(`${this.base}/inventario/alertas`);
  }

  configuracion(): Observable<{ diasAlertaVencimiento: number }> {
    return this.http.get<{ diasAlertaVencimiento: number }>(`${this.base}/inventario/configuracion`);
  }

  guardarConfiguracion(diasAlertaVencimiento: number): Observable<{ diasAlertaVencimiento: number }> {
    return this.http.put<{ diasAlertaVencimiento: number }>(`${this.base}/inventario/configuracion`, { diasAlertaVencimiento });
  }
}
