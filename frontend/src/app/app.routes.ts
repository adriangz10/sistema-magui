import { Routes } from '@angular/router';
import { sesionGuard, sinSesionGuard } from './core/auth/auth.guard';

const proximamente = () => import('./features/proximamente/proximamente.page').then((m) => m.ProximamentePage);

/** Sección provisoria hasta la Fase 8. */
function seccion(path: string, titulo: string) {
  return { path, title: titulo, loadComponent: proximamente, data: { titulo } };
}

export const routes: Routes = [
  {
    path: 'ingresar',
    title: 'Ingresar',
    canMatch: [sinSesionGuard],
    loadComponent: () => import('./features/ingresar/ingresar.page').then((m) => m.IngresarPage),
  },
  {
    path: '',
    canMatch: [sesionGuard],
    loadComponent: () => import('./layout/shell.component').then((m) => m.ShellComponent),
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'inicio' },
      { path: 'inicio', title: 'Inicio', loadComponent: () => import('./features/inicio/inicio.page').then((m) => m.InicioPage) },
      seccion('vender', 'Vender'),
      seccion('productos', 'Productos'),
      seccion('categorias', 'Categorías'),
      seccion('inventario', 'Inventario'),
      seccion('compras', 'Compras'),
      seccion('ventas', 'Ventas'),
      seccion('caja', 'Caja'),
      seccion('estadisticas', 'Estadísticas'),
      seccion('configuracion', 'Configuración'),
    ],
  },
  { path: '**', redirectTo: '' },
];
