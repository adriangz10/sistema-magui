import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';
import { OrigenOperacion } from '../config/origen';
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
  /** Sube 1 con cada edición; se envía al editar (control de concurrencia). */
  version: number;
  /** Instante de la última edición; null si nunca se editó. */
  editadaEn: string | null;
}

/** Cambio de un campo de la cabecera o de un renglón (valores ya formateables según `campo`). */
export interface CambioCampo {
  campo: string;
  antes: string | number | null;
  despues: string | number | null;
  antesId?: string | null;
  despuesId?: string | null;
}

export interface CambioRenglon {
  accion: 'MODIFICADO' | 'AGREGADO' | 'QUITADO';
  itemId?: string;
  productoId: string;
  producto: string;
  cambios: CambioCampo[];
}

export interface EdicionCompra {
  id: string;
  fecha: string;
  origen: OrigenOperacion;
  motivo: string | null;
  version: number;
  cabecera: CambioCampo[];
  renglones: CambioRenglon[];
}

export interface CompraDetalle extends CompraResumen {
  ediciones: EdicionCompra[];
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
    /** Unidades del lote que ya salieron (ventas, ajustes, combos). Mínimo editable de la cantidad. */
    consumido: number;
    /** Precio de venta vigente hoy del producto; null si no tiene. Opcional: respuestas viejas no lo traen. */
    precioVentaActual?: Monto | null;
  }[];
}

/** Renglón existente dentro del PUT (US-001b): `id` obligatorio, mismo `productoId`. */
export interface ItemEditado {
  id: string;
  productoId: string;
  cantidad: number;
  costoUnitario: Monto;
  precioVentaNuevo: Monto | null;
  vencimiento: FechaISO | null;
}

/** Cuerpo de PUT /compras/{id}. Sin `items` los renglones no cambian; con `items` es la lista completa (US-001b). */
export interface EditarCompra {
  fecha: FechaISO;
  empresaId: string;
  notas: string;
  version: number;
  origen: OrigenOperacion;
  motivo?: string;
  items?: ItemEditado[];
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

  editar(id: string, c: EditarCompra): Observable<CompraDetalle> {
    return this.http.put<CompraDetalle>(`${this.base}/compras/${id}`, c);
  }
}
