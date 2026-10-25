/** Página de resultados (formato común de todas las listas de la API). */
export interface Pagina<T> {
  items: T[];
  page: number;
  size: number;
  total: number;
  totalPages: number;
}

/** Montos: la API los envía y recibe como texto con 2 decimales ("59200.00"). */
export type Monto = string;

/** Fecha sin hora: "2026-10-25". */
export type FechaISO = string;

/** Arma los parámetros de una consulta omitiendo valores vacíos. */
export function parametros(valores: Record<string, string | number | boolean | null | undefined>): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [clave, valor] of Object.entries(valores)) {
    if (valor !== null && valor !== undefined && valor !== '' && valor !== false) {
      out[clave] = String(valor);
    }
  }
  return out;
}
