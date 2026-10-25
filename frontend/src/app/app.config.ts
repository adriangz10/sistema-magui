import { provideHttpClient, withFetch, withInterceptors } from '@angular/common/http';
import { ApplicationConfig, LOCALE_ID, isDevMode, provideBrowserGlobalErrorListeners } from '@angular/core';
import { registerLocaleData } from '@angular/common';
import localeEsAr from '@angular/common/locales/es-AR';
import { TitleStrategy, provideRouter, withComponentInputBinding } from '@angular/router';

import { routes } from './app.routes';
import { authInterceptor } from './core/auth/auth.interceptor';
import { conexionInterceptor } from './core/red/conexion.interceptor';
import { TituloMagui } from './core/titulo';
import { provideServiceWorker } from '@angular/service-worker';


registerLocaleData(localeEsAr);

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes, withComponentInputBinding()),
    provideHttpClient(withFetch(), withInterceptors([conexionInterceptor, authInterceptor])),
    { provide: TitleStrategy, useClass: TituloMagui },
    { provide: LOCALE_ID, useValue: 'es-AR' },
    // App instalable y que abre sin señal (solo en la versión compilada, no en desarrollo).
    provideServiceWorker('ngsw-worker.js', {
      enabled: !isDevMode(),
      registrationStrategy: 'registerWhenStable:30000',
    }),
  ],
};
