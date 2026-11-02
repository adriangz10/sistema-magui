import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';
import { FechaISO, Monto, Pagina, parametros } from './modelos';

export type MetodoPago = 'EFECTIVO' | 'TRANSFERENCIA' | 'QR' | 'TARJETA';
export type EstadoVenta = 'CONFIRMADA' | 'ANULADA';
export type OrigenVenta = 'WEB' | 'MOVIL' | 'DESKTOP';

export const METODOS: readonly { valor: MetodoPago; etiqueta: string }[] = [
  { valor: 'EFECTIVO', etiqueta: 'Efectivo' },
  { valor: 'TRANSFERENCIA', etiqueta: 'Transferencia' },
  { valor: 'QR', etiqueta: 'QR' },
  { valor: 'TARJETA', etiqueta: 'Tarjeta' },
];

export interface VentaResumen {
  id: string;
  fecha: string;
  total: Monto;
  estado: EstadoVenta;
  origen: OrigenVenta;
  productos: number;
  unidades: number;
  metodos: MetodoPago[];
}

export interface VentaDetalle extends VentaResumen {
  costoTotal: Monto;
  ganancia: Monto;
  /** false si algún producto no tenía costo cargado: la ganancia es aproximada. */
  costoCompleto: boolean;
  vuelto: Monto;
  anuladaEn: string | null;
  motivoAnulacion: string | null;
  items: { productoId: string; sku: string; producto: string; tipo: string; cantidad: number; precioUnitario: Monto; subtotal: Monto }[];
  pagos: { metodo: MetodoPago; monto: Monto; montoRecibido: Monto | null; vuelto: Monto | null }[];
}

export interface NuevaVenta {
  items: { productoId: string; cantidad: number }[];
  pagos: { metodo: MetodoPago; monto: Monto }[];
  origen: OrigenVenta;
  permitirSinStock: boolean;
}

export interface FiltroVentas {
  /** Día (AAAA-MM-DD) o instante ISO (ej. la apertura de una caja). */
  desde?: FechaISO;
  /** Día inclusivo (AAAA-MM-DD) o instante ISO exclusivo (ej. el cierre de una caja). */
  hasta?: FechaISO;
  estado?: EstadoVenta | '';
  metodo?: MetodoPago | '';
  page?: number;
}

/** Cliente de ventas. */
@Injectable({ providedIn: 'root' })
export class VentasApi {
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_BASE_URL);

  registrar(v: NuevaVenta): Observable<VentaDetalle> {
    return this.http.post<VentaDetalle>(`${this.base}/ventas`, v);
  }

  listar(f: FiltroVentas = {}): Observable<Pagina<VentaResumen>> {
    return this.http.get<Pagina<VentaResumen>>(`${this.base}/ventas`, { params: parametros({ ...f }) });
  }

  obtener(id: string): Observable<VentaDetalle> {
    return this.http.get<VentaDetalle>(`${this.base}/ventas/${id}`);
  }

  anular(id: string, motivo: string): Observable<VentaDetalle> {
    return this.http.post<VentaDetalle>(`${this.base}/ventas/${id}/anular`, { motivo });
  }
}
