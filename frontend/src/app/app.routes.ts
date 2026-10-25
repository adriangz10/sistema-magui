import { Routes } from '@angular/router';
import { sesionGuard, sinSesionGuard } from './core/auth/auth.guard';

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
      { path: 'vender', title: 'Vender', loadComponent: () => import('./features/vender/vender.page').then((m) => m.VenderPage) },
      { path: 'productos', title: 'Productos', loadComponent: () => import('./features/productos/productos.page').then((m) => m.ProductosPage) },
      { path: 'categorias', title: 'Categorías', loadComponent: () => import('./features/categorias/categorias.page').then((m) => m.CategoriasPage) },
      { path: 'inventario', title: 'Inventario', loadComponent: () => import('./features/inventario/inventario.page').then((m) => m.InventarioPage) },
      { path: 'compras', title: 'Compras', loadComponent: () => import('./features/compras/compras.page').then((m) => m.ComprasPage) },
      { path: 'ventas', title: 'Ventas', loadComponent: () => import('./features/ventas/ventas.page').then((m) => m.VentasPage) },
      { path: 'caja', title: 'Caja', loadComponent: () => import('./features/caja/caja.page').then((m) => m.CajaPage) },
      { path: 'estadisticas', title: 'Estadísticas', loadComponent: () => import('./features/estadisticas/estadisticas.page').then((m) => m.EstadisticasPage) },
      { path: 'configuracion', title: 'Configuración', loadComponent: () => import('./features/configuracion/configuracion.page').then((m) => m.ConfiguracionPage) },
    ],
  },
  { path: '**', redirectTo: '' },
];
