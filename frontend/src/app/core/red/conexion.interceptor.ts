import { HttpErrorResponse, HttpInterceptorFn, HttpResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { tap } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';
import { ConexionService } from './conexion.service';

/** Estados que significan "no se llegó al servidor" (sin red, o el service worker sin respuesta). */
export function sinRespuesta(status: number): boolean {
  return status === 0 || status === 502 || status === 503 || status === 504;
}

/**
 * Consultas que el service worker puede responder desde su copia sin conexión
 * (ver dataGroups en ngsw-config.json): una respuesta de estas no prueba que haya servidor.
 */
const DESDE_COPIA = /\/api\/(productos|categorias|empresas)(\/|\?|$)/;

/** Actualiza el indicador de conexión según lo que pasa con cada llamada a la API. */
export const conexionInterceptor: HttpInterceptorFn = (req, next) => {
  if (!req.url.startsWith(inject(API_BASE_URL))) {
    return next(req);
  }
  const conexion = inject(ConexionService);
  return next(req).pipe(
    tap({
      next: (e) => e instanceof HttpResponse && !(req.method === 'GET' && DESDE_COPIA.test(req.url)) && conexion.informar(true),
      error: (e: unknown) => e instanceof HttpErrorResponse && conexion.informar(!sinRespuesta(e.status)),
    }),
  );
};
