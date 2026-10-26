/**
 * true solo en la compilación de escritorio (`npm run build:escritorio`), que reemplaza este archivo
 * por plataforma.escritorio.ts. En el escritorio no hay service worker (la app viaja dentro del .exe),
 * las ventas se marcan DESKTOP y se muestran las ventas pendientes de subir.
 */
export const ES_ESCRITORIO: boolean = false;
