import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';

export type AnchoPapel = 58 | 80;

/** Configuración del ticket de cada venta (no es factura). */
export interface ConfigTicket {
  habilitado: boolean;
  /** Imprime apenas se cobra; si no, queda el botón "Imprimir ticket". */
  imprimirAlCobrar: boolean;
  anchoPapel: AnchoPapel;
  mostrarLogo: boolean;
  /** Dirección, teléfono, CUIT... una por línea. */
  datos: string | null;
  pie: string | null;
}

/** Lo público: se ve también en la pantalla de ingreso. */
export interface Marca {
  nombre: string;
  tieneLogo: boolean;
  logoVersion: number;
}

export interface Negocio extends Marca {
  ticket: ConfigTicket;
}

/** Cliente de los datos del negocio (US-003). */
@Injectable({ providedIn: 'root' })
export class NegocioApi {
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_BASE_URL);

  marca(): Observable<Marca> {
    return this.http.get<Marca>(`${this.base}/negocio/marca`);
  }

  obtener(): Observable<Negocio> {
    return this.http.get<Negocio>(`${this.base}/negocio`);
  }

  guardar(n: { nombre: string; ticket: ConfigTicket }): Observable<Negocio> {
    return this.http.put<Negocio>(`${this.base}/negocio`, n);
  }

  /** Sube el archivo tal cual (el servidor valida que sea PNG, JPG o WEBP de hasta 512 KB). */
  subirLogo(archivo: Blob): Observable<Marca> {
    return this.http.put<Marca>(`${this.base}/negocio/logo`, archivo, { headers: { 'Content-Type': archivo.type || 'application/octet-stream' } });
  }

  quitarLogo(): Observable<Marca> {
    return this.http.delete<Marca>(`${this.base}/negocio/logo`);
  }

  urlLogo(version: number): string {
    return `${this.base}/negocio/logo?v=${version}`;
  }
}
