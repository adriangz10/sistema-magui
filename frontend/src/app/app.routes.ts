import { Routes } from '@angular/router';

export const routes: Routes = [
  { path: '', loadComponent: () => import('./features/inicio/inicio.page').then((m) => m.InicioPage) },
  { path: '**', redirectTo: '' },
];
