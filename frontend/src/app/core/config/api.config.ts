import { InjectionToken } from '@angular/core';

/**
 * URL base de la API. Por defecto '/api' (mismo origen):
 * - web/móvil: la API Go sirve el frontend.
 * - desarrollo: proxy.conf.json redirige a localhost:8080.
 * - escritorio: Wails sirve la app y el backend local.
 */
export const API_BASE_URL = new InjectionToken<string>('API_BASE_URL', {
  providedIn: 'root',
  factory: () => '/api',
});
