import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';
import { FechaISO, Monto, Pagina, parametros } from './modelos';

export interface CompraResumen {
  id: string;
  fecha: string;
  empresaId: string | null;
  empresa: string | null;
  total: Monto;
  productos: number;
  unidades: number;
  notas: string | null;
}

export interface CompraDetalle extends CompraResumen {
  items: {
    id: string;
    productoId: string;
    sku: string;
    producto: string;
    cantidad: number;
    costoUnitario: Monto;
    subtotal: Monto;
    precioVentaNuevo: Monto | null;
    vencimiento: FechaISO | null;
  }[];
}

export interface NuevaCompra {
  /** Vacío = ahora. */
  fecha: FechaISO | '';
  empresaId: string;
  notas: string;
  items: { productoId: string; cantidad: number; costoUnitario: Monto; precioVentaNuevo: Monto | null; vencimiento: FechaISO | null }[];
}

/** Cliente de compras de mercadería. */
@Injectable({ providedIn: 'root' })
export class ComprasApi {
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_BASE_URL);

  listar(f: { desde?: FechaISO; hasta?: FechaISO; empresaId?: string; page?: number }): Observable<Pagina<CompraResumen>> {
    return this.http.get<Pagina<CompraResumen>>(`${this.base}/compras`, { params: parametros({ ...f }) });
  }

  obtener(id: string): Observable<CompraDetalle> {
    return this.http.get<CompraDetalle>(`${this.base}/compras/${id}`);
  }

  registrar(c: NuevaCompra): Observable<CompraDetalle> {
    return this.http.post<CompraDetalle>(`${this.base}/compras`, c);
  }
}
