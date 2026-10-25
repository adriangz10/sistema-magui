import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';
import { AuthService } from './auth.service';

/**
 * Agrega el token a las llamadas a la API y, si la sesión venció (401 fuera del login),
 * la cierra y manda a ingresar de nuevo.
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const baseUrl = inject(API_BASE_URL);

  const esApi = req.url.startsWith(baseUrl);
  const esLogin = req.url === `${baseUrl}/auth/login`;
  const token = auth.token();

  const conToken = esApi && token && !esLogin ? req.clone({ setHeaders: { Authorization: `Bearer ${token}` } }) : req;

  return next(conToken).pipe(
    catchError((err: unknown) => {
      if (esApi && !esLogin && err instanceof HttpErrorResponse && err.status === 401) {
        auth.cerrarSesion(router.url);
      }
      return throwError(() => err);
    }),
  );
};
