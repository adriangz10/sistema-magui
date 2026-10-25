import { inject } from '@angular/core';
import { CanMatchFn, Router } from '@angular/router';
import { AuthService } from './auth.service';

/** Solo deja entrar con sesión; si no, manda a la pantalla de ingreso. */
export const sesionGuard: CanMatchFn = (_route, segmentos) => {
  if (inject(AuthService).autenticada()) {
    return true;
  }
  const volver = '/' + segmentos.map((s) => s.path).join('/');
  return inject(Router).createUrlTree(['/ingresar'], { queryParams: volver !== '/' ? { volver } : {} });
};

/** La pantalla de ingreso no se muestra si ya hay sesión. */
export const sinSesionGuard: CanMatchFn = () =>
  inject(AuthService).autenticada() ? inject(Router).createUrlTree(['/']) : true;
