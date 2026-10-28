import { CambioCampo, CompraDetalle, EditarCompra, ItemEditado } from '../../core/api/compras.api';
import { OrigenOperacion } from '../../core/config/origen';
import { aCentavos, deCentavos, formatearDinero } from '../../shared/formato/dinero';
import { formatearFecha } from '../../shared/formato/fecha';

export interface CabeceraEditable {
  /** AAAA-MM-DD */
  fecha: string;
  empresaId: string;
  notas: string;
  motivo: string;
}

/** "2026-10-20T15:00:00Z" -> "2026-10-20" en hora argentina (el input de fecha no lleva hora). */
export function diaArgentino(instante: string): string {
  if (/^\d{4}-\d{2}-\d{2}$/.test(instante)) {
    return instante;
  }
  const f = new Date(instante);
  if (Number.isNaN(f.getTime())) {
    return instante.slice(0, 10);
  }
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'America/Argentina/Buenos_Aires' }).format(f);
}

/** Valores iniciales del formulario a partir del detalle. */
export function cabeceraDe(c: CompraDetalle): CabeceraEditable {
  return { fecha: diaArgentino(c.fecha), empresaId: c.empresaId ?? '', notas: c.notas ?? '', motivo: '' };
}

/** Hay algo para guardar: cambió fecha (por día), empresa o notas. El motivo solo no cuenta (RN-A-02). */
export function hayCambios(original: CompraDetalle, actual: CabeceraEditable): boolean {
  const base = cabeceraDe(original);
  return base.fecha !== actual.fecha || base.empresaId !== actual.empresaId || base.notas.trim() !== actual.notas.trim();
}

/** Cuerpo del PUT: siempre fecha AAAA-MM-DD, `version` leída, sin `items` y sin `motivo` si está vacío. */
export function armarEdicion(
  original: CompraDetalle, actual: CabeceraEditable, origen: OrigenOperacion, items?: ItemEditado[],
): EditarCompra {
  const cuerpo: EditarCompra = {
    fecha: actual.fecha,
    empresaId: actual.empresaId,
    notas: actual.notas.trim(),
    version: original.version,
    origen,
  };
  const motivo = actual.motivo.trim();
  if (motivo) {
    cuerpo.motivo = motivo;
  }
  if (items) {
    cuerpo.items = items;
  }
  return cuerpo;
}

/** El servidor responde 200 con la misma versión cuando no hubo cambios. */
export function seAplicoEdicion(versionLeida: number, respuesta: { version: number }): boolean {
  return respuesta.version > versionLeida;
}

export const ETIQUETA_CAMPO: Record<string, string> = {
  fecha: 'Fecha', empresa: 'Empresa', notas: 'Notas', cantidad: 'Cantidad', costoUnitario: 'Costo',
  precioVentaNuevo: 'Nuevo precio', vencimiento: 'Vencimiento',
};

/** Texto de un valor de cambio según el campo; vacío se muestra como "sin …". */
export function formatearValorCambio(campo: string, valor: CambioCampo['antes'], otro?: CambioCampo['antes']): string {
  if (valor === null || valor === undefined || valor === '') {
    return campo === 'empresa' ? 'sin empresa' : campo === 'notas' ? 'sin notas' : 'vacío';
  }
  switch (campo) {
    case 'fecha': {
      // Mismo día antes y después (cambió solo la hora): mostrar también la hora.
      const mismoDia = otro != null && diaArgentino(String(valor)) === diaArgentino(String(otro));
      return formatearFecha(String(valor), mismoDia ? 'fecha-hora' : 'fecha');
    }
    case 'vencimiento':
      return formatearFecha(String(valor));
    case 'costoUnitario':
    case 'precioVentaNuevo':
      return formatearDinero(String(valor));
    default:
      return String(valor);
  }
}

export const ORIGEN_ETIQUETA: Record<OrigenOperacion, string> = { WEB: 'Web', MOVIL: 'Celular', DESKTOP: 'Escritorio' };

/** Renglón existente tal como se edita en pantalla. */
export interface RenglonEditable {
  id: string;
  productoId: string;
  producto: string;
  sku: string;
  cantidad: number | string | null;
  costo: string | null;
  precioNuevo: string | null;
  /** AAAA-MM-DD o vacío */
  vencimiento: string;
  consumido: number;
  /** Cantidad con la que se leyó la compra. */
  cantidadOriginal: number;
  /** Nuevo precio con el que se leyó la compra. */
  precioNuevoOriginal: string | null;
  /** Precio de venta vigente del producto (null/undefined si no se conoce). */
  precioActual: string | null;
}

export function renglonesDe(c: CompraDetalle): RenglonEditable[] {
  return c.items.map((it) => ({
    id: it.id, productoId: it.productoId, producto: it.producto, sku: it.sku,
    cantidad: it.cantidad, costo: it.costoUnitario, precioNuevo: it.precioVentaNuevo,
    vencimiento: it.vencimiento ?? '', consumido: it.consumido ?? 0, cantidadOriginal: it.cantidad,
    precioNuevoOriginal: it.precioVentaNuevo, precioActual: it.precioVentaActual ?? null,
  }));
}

/** Cantidad mínima editable: lo ya consumido, y nunca menos de 1. */
export function minimoCantidad(consumido: number): number {
  return Math.max(consumido ?? 0, 1);
}

export function mensajeConsumido(consumido: number): string {
  return `Ya salieron ${consumido} unidades de este lote, no podés dejar menos de ${consumido}`;
}

/** Errores locales por clave `items[i].campo`. */
export function validarRenglones(renglones: RenglonEditable[]): Record<string, string> {
  const e: Record<string, string> = {};
  renglones.forEach((r, i) => {
    const cant = Number(r.cantidad);
    if (r.cantidad === null || r.cantidad === '' || !Number.isInteger(cant) || cant < 1) {
      e[`items[${i}].cantidad`] = 'Mayor a cero';
    } else if (r.consumido > 0 && cant < r.consumido) {
      e[`items[${i}].cantidad`] = mensajeConsumido(r.consumido);
    }
    if (r.costo === null) {
      e[`items[${i}].costoUnitario`] = 'Ingresá el costo';
    }
    if (r.precioNuevo !== null && aCentavos(r.precioNuevo) <= 0) {
      e[`items[${i}].precioVentaNuevo`] = 'Mayor a cero o vacío';
    }
  });
  return e;
}

/** Algún campo editable de algún renglón difiere del detalle leído. */
export function hayCambiosRenglones(original: CompraDetalle, renglones: RenglonEditable[]): boolean {
  return original.items.some((it, i) => {
    const r = renglones[i];
    return !r
      || Number(r.cantidad) !== it.cantidad
      || aCentavos(r.costo) !== aCentavos(it.costoUnitario)
      || (r.precioNuevo === null ? null : aCentavos(r.precioNuevo)) !== (it.precioVentaNuevo === null ? null : aCentavos(it.precioVentaNuevo))
      || (r.vencimiento || null) !== (it.vencimiento ?? null);
  });
}

/** Lista completa de renglones para el PUT (todos con `id`, en el orden del detalle). */
export function armarItems(renglones: RenglonEditable[]): ItemEditado[] {
  return renglones.map((r) => ({
    id: r.id, productoId: r.productoId, cantidad: Number(r.cantidad), costoUnitario: r.costo!,
    precioVentaNuevo: r.precioNuevo, vencimiento: r.vencimiento || null,
  }));
}

export function totalRenglones(renglones: RenglonEditable[]): string {
  return deCentavos(renglones.reduce((t, r) => t + aCentavos(r.costo) * (Number(r.cantidad) || 0), 0));
}

/** Efectos de stock antes de guardar: "El stock de Kaiak sube 2". */
export function resumenEfectos(renglones: RenglonEditable[]): string[] {
  const out: string[] = [];
  for (const r of renglones) {
    const d = (Number(r.cantidad) || 0) - r.cantidadOriginal;
    if (d > 0) {
      out.push(`El stock de ${r.producto} sube ${d}`);
    } else if (d < 0) {
      out.push(`El stock de ${r.producto} baja ${-d}`);
    }
    const nuevo = r.precioNuevo === null ? null : aCentavos(r.precioNuevo);
    const leido = r.precioNuevoOriginal === null ? null : aCentavos(r.precioNuevoOriginal);
    if (nuevo !== null && nuevo !== leido && (r.precioActual === null || nuevo !== aCentavos(r.precioActual))) {
      const hasta = formatearDinero(r.precioNuevo);
      const condicion = `si es la compra más reciente de ${r.producto}`;
      out.push(r.precioActual === null
        ? `El precio de venta de ${r.producto} queda en ${hasta} ${condicion}`
        : `El precio de venta de ${r.producto} pasará de ${formatearDinero(r.precioActual)} a ${hasta} ${condicion}`);
    }
  }
  return out;
}
