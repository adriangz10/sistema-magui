/**
 * Utilidades de escala para gráficos: marcas de eje "redondas" y formato compacto.
 */

/** Devuelve marcas de eje limpias (0, 5K, 10K…) que cubren el máximo. */
export function marcasEje(maximo: number, cantidad = 4): number[] {
  if (maximo <= 0) {
    return [0, 1];
  }
  const crudo = maximo / cantidad;
  const potencia = 10 ** Math.floor(Math.log10(crudo));
  const paso = [1, 2, 2.5, 5, 10].map((m) => m * potencia).find((p) => p >= crudo) ?? 10 * potencia;
  const marcas: number[] = [];
  for (let v = 0; v <= maximo + paso * 0.001; v += paso) {
    marcas.push(Math.round(v * 100) / 100);
  }
  if (marcas[marcas.length - 1] < maximo) {
    marcas.push(marcas[marcas.length - 1] + paso);
  }
  return marcas;
}

/** 1500 -> "1,5 mil", 2300000 -> "2,3 M" (formato argentino compacto). */
export function compacto(valor: number, prefijo = ''): string {
  const abs = Math.abs(valor);
  const signo = valor < 0 ? '-' : '';
  const fmt = (n: number) => new Intl.NumberFormat('es-AR', { maximumFractionDigits: 1 }).format(n);
  if (abs >= 1_000_000) {
    return `${signo}${prefijo}${fmt(abs / 1_000_000)} M`;
  }
  if (abs >= 1_000) {
    return `${signo}${prefijo}${fmt(abs / 1_000)} mil`;
  }
  return `${signo}${prefijo}${fmt(abs)}`;
}
