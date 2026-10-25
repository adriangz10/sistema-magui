import { Boxes, ChartColumn, House, LucideIconData, Package, Receipt, Settings, ShoppingCart, Tags, Truck, Wallet } from 'lucide-angular';

export interface ItemMenu {
  ruta: string;
  etiqueta: string;
  icono: LucideIconData;
  /** Se muestra en la barra inferior del celular. */
  movil?: boolean;
}

/** Secciones de la app, en el orden del menú. */
export const MENU: readonly ItemMenu[] = [
  { ruta: '/inicio', etiqueta: 'Inicio', icono: House, movil: true },
  { ruta: '/vender', etiqueta: 'Vender', icono: ShoppingCart, movil: true },
  { ruta: '/productos', etiqueta: 'Productos', icono: Package, movil: true },
  { ruta: '/categorias', etiqueta: 'Categorías', icono: Tags },
  { ruta: '/inventario', etiqueta: 'Inventario', icono: Boxes },
  { ruta: '/compras', etiqueta: 'Compras', icono: Truck },
  { ruta: '/ventas', etiqueta: 'Ventas', icono: Receipt, movil: true },
  { ruta: '/caja', etiqueta: 'Caja', icono: Wallet },
  { ruta: '/estadisticas', etiqueta: 'Estadísticas', icono: ChartColumn },
  { ruta: '/configuracion', etiqueta: 'Configuración', icono: Settings },
];
