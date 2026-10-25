import { ProductoVendido, VentaDia, VentaHora } from '../../core/api/estadisticas.api';
import { aCentavos, formatearDinero } from '../formato/dinero';
import { FilaBarra } from './barras-horizontales.component';
import { PuntoColumna } from './grafico-columnas.component';

const DIAS = ['dom', 'lun', 'mar', 'mié', 'jue', 'vie', 'sáb'];

/** "2026-10-20" -> "lun 20" (o "20/10" si son muchos días). */
export function etiquetaDia(fechaISO: string, corta = false): string {
  const [a, m, d] = fechaISO.split('-').map(Number);
  if (corta) {
    return `${d}/${m}`;
  }
  return `${DIAS[new Date(Date.UTC(a, m - 1, d)).getUTCDay()]} ${d}`;
}

const cantidadVentas = (n: number) => `${n} ${n === 1 ? 'venta' : 'ventas'}`;

export function columnasPorDia(dias: VentaDia[]): PuntoColumna[] {
  const corta = dias.length > 10;
  return dias.map((d) => ({
    etiqueta: etiquetaDia(d.fecha, corta),
    valor: aCentavos(d.total) / 100,
    texto: formatearDinero(d.total),
    detalle: `${cantidadVentas(d.cantidad)} · ganancia ${formatearDinero(d.ganancia)}`,
  }));
}

/** Solo el horario con movimiento (de la primera a la última hora con ventas, mínimo 9 a 20 h). */
export function columnasPorHora(horas: VentaHora[]): PuntoColumna[] {
  const conVentas = horas.filter((h) => h.cantidad > 0).map((h) => h.hora);
  const desde = Math.min(9, ...conVentas);
  const hasta = Math.max(20, ...conVentas);
  return horas.filter((h) => h.hora >= desde && h.hora <= hasta).map((h) => ({
    etiqueta: `${h.hora} h`,
    valor: aCentavos(h.total) / 100,
    texto: formatearDinero(h.total),
    detalle: cantidadVentas(h.cantidad),
  }));
}

export function filasMasVendidos(lista: ProductoVendido[], orden: 'unidades' | 'importe'): FilaBarra[] {
  return lista.map((p) => ({
    etiqueta: p.nombre,
    valor: orden === 'unidades' ? p.unidades : aCentavos(p.total),
    texto: orden === 'unidades' ? `${p.unidades} u.` : formatearDinero(p.total),
    detalle: orden === 'unidades'
      ? `${formatearDinero(p.total)}${p.ganancia ? ' · ganancia ' + formatearDinero(p.ganancia) : ''}`
      : `${p.unidades} u.${p.ganancia ? ' · ganancia ' + formatearDinero(p.ganancia) : ''}`,
  }));
}
