import { HttpErrorResponse, HttpInterceptorFn, HttpResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { tap } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';
import { ES_ESCRITORIO } from '../config/plataforma';
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

/**
 * En el escritorio, rutas que responde el núcleo local aunque no haya servidor: no prueban nada.
 * Ahí el estado de la conexión lo informa EscritorioService (lo sabe el núcleo).
 */
const LOCALES_ESCRITORIO = /\/api\/(ventas|escritorio\/.*|auth\/login)$/;

/** true si la respuesta a este pedido no dice nada sobre si hay servidor. */
export function noIndicaConexion(metodo: string, url: string, escritorio = ES_ESCRITORIO): boolean {
  const ruta = url.split('?')[0];
  if (metodo === 'GET' && DESDE_COPIA.test(url)) {
    return true;
  }
  return escritorio && (LOCALES_ESCRITORIO.test(ruta) && (metodo === 'POST' || ruta.includes('/escritorio/')));
}

/** Actualiza el indicador de conexión según lo que pasa con cada llamada a la API. */
export const conexionInterceptor: HttpInterceptorFn = (req, next) => {
  if (!req.url.startsWith(inject(API_BASE_URL))) {
    return next(req);
  }
  const conexion = inject(ConexionService);
  return next(req).pipe(
    tap({
      next: (e) => e instanceof HttpResponse && !noIndicaConexion(req.method, req.url) && conexion.informar(true),
      error: (e: unknown) => {
        if (e instanceof HttpErrorResponse) {
          if (sinRespuesta(e.status)) {
            conexion.informar(false);
          } else if (!noIndicaConexion(req.method, req.url)) {
            conexion.informar(true);
          }
        }
      },
    }),
  );
};
