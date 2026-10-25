/**
 * Acceso seguro a localStorage: en modo privado o con el almacenamiento bloqueado
 * las llamadas pueden fallar, y la app tiene que seguir funcionando.
 */
export const almacenamiento = {
  leer(clave: string): string | null {
    try {
      return globalThis.localStorage?.getItem(clave) ?? null;
    } catch {
      return null;
    }
  },
  guardar(clave: string, valor: string): void {
    try {
      globalThis.localStorage?.setItem(clave, valor);
    } catch {
      // sin almacenamiento: se pierde al recargar, no es grave
    }
  },
  borrar(clave: string): void {
    try {
      globalThis.localStorage?.removeItem(clave);
    } catch {
      // nada que hacer
    }
  },
};
