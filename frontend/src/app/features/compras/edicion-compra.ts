import { CambioCampo, CambioRenglon, CompraDetalle, EditarCompra, ItemEditado } from '../../core/api/compras.api';
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

/** Renglón tal como se edita en pantalla: existente (con `id` del servidor) o nuevo (`nuevo`, id local). */
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
  /** Cantidad con la que se leyó la compra (0 en renglones nuevos). */
  cantidadOriginal: number;
  /** Nuevo precio con el que se leyó la compra. */
  precioNuevoOriginal: string | null;
  /** Precio de venta vigente del producto (null/undefined si no se conoce). */
  precioActual: string | null;
  /** Renglón agregado en esta edición: se envía sin `id`. */
  nuevo?: boolean;
  /** Renglón existente marcado "se quitará": no se envía. */
  quitado?: boolean;
}

export const MAX_RENGLONES = 300;

export function renglonesDe(c: CompraDetalle): RenglonEditable[] {
  return c.items.map((it) => ({
    id: it.id, productoId: it.productoId, producto: it.producto, sku: it.sku,
    cantidad: it.cantidad, costo: it.costoUnitario, precioNuevo: it.precioVentaNuevo,
    vencimiento: it.vencimiento ?? '', consumido: it.consumido ?? 0, cantidadOriginal: it.cantidad,
    precioNuevoOriginal: it.precioVentaNuevo, precioActual: it.precioVentaActual ?? null,
  }));
}

/** Renglón nuevo para un producto elegido en el buscador (id local, nunca se envía). */
export function renglonNuevo(
  clave: number, p: { id: string; nombre: string; sku: string; precioVenta: string | null; costo: string | null },
): RenglonEditable {
  return {
    id: `nuevo-${clave}`, productoId: p.id, producto: p.nombre, sku: p.sku, cantidad: 1, costo: p.costo, precioNuevo: null,
    vencimiento: '', consumido: 0, cantidadOriginal: 0, precioNuevoOriginal: null, precioActual: p.precioVenta ?? null, nuevo: true,
  };
}

export const activos = (rs: RenglonEditable[]): RenglonEditable[] => rs.filter((r) => !r.quitado);

/** Cantidad mínima editable: lo ya consumido, y nunca menos de 1. */
export function minimoCantidad(consumido: number): number {
  return Math.max(consumido ?? 0, 1);
}

export function mensajeConsumido(consumido: number): string {
  return `Ya salieron ${consumido} unidades de este lote, no podés dejar menos de ${consumido}`;
}

/** Por qué no se puede quitar el renglón; null si se puede. */
export function motivoNoQuitar(r: RenglonEditable, renglones: RenglonEditable[]): string | null {
  if (r.consumido > 0) {
    // El "ya salieron N de M" ya está en la línea del SKU: acá solo la consecuencia.
    return r.consumido < r.cantidadOriginal
      ? `No se puede quitar. Podés bajar la cantidad hasta ${r.consumido}.`
      : 'No se puede quitar porque ya salió todo el lote.';
  }
  if (!r.nuevo && activos(renglones).length <= 1) {
    return 'La compra necesita al menos un producto.';
  }
  return null;
}

/** Errores locales por clave `items[i].campo` (i = posición en pantalla; los quitados se saltean). */
export function validarRenglones(renglones: RenglonEditable[]): Record<string, string> {
  const e: Record<string, string> = {};
  if (renglones.length > 0 && activos(renglones).length === 0) {
    e['items'] = 'Agregá al menos un producto';
  } else if (activos(renglones).length > MAX_RENGLONES) {
    e['items'] = `Máximo ${MAX_RENGLONES} renglones por compra`;
  }
  renglones.forEach((r, i) => {
    if (r.quitado) {
      return;
    }
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

/**
 * Traduce los campos del servidor a posiciones en pantalla: `items[k]` es la posición k del array enviado
 * (sin quitados, nuevos al final) y `quitados[<id>]` ya viene por id del renglón.
 */
export function mapearErroresServidor(renglones: RenglonEditable[], campos: Record<string, string>): Record<string, string> {
  const enviados = ordenEnvio(renglones);
  const out: Record<string, string> = {};
  for (const [clave, msg] of Object.entries(campos)) {
    const m = /^items\[(\d+)\](.*)$/.exec(clave);
    const destino = m ? enviados[Number(m[1])] : undefined;
    out[m && destino ? `items[${renglones.indexOf(destino)}]${m[2]}` : clave] = msg;
  }
  return out;
}

/** Renglones en el orden en que se envían: existentes, luego nuevos; sin quitados. */
function ordenEnvio(renglones: RenglonEditable[]): RenglonEditable[] {
  return activos([...renglones.filter((r) => !r.nuevo), ...renglones.filter((r) => r.nuevo)]);
}

/** Hay algo para guardar: renglones agregados o quitados, o algún campo de un existente distinto del detalle leído. */
export function hayCambiosRenglones(original: CompraDetalle, renglones: RenglonEditable[]): boolean {
  if (renglones.some((r) => r.nuevo || r.quitado)) {
    return true;
  }
  return original.items.some((it) => {
    const r = renglones.find((x) => x.id === it.id);
    return !r
      || Number(r.cantidad) !== it.cantidad
      || aCentavos(r.costo) !== aCentavos(it.costoUnitario)
      || (r.precioNuevo === null ? null : aCentavos(r.precioNuevo)) !== (it.precioVentaNuevo === null ? null : aCentavos(it.precioVentaNuevo))
      || (r.vencimiento || null) !== (it.vencimiento ?? null);
  });
}

/** Lista completa para el PUT: existentes con `id`, nuevos sin `id` (al final), quitados omitidos. */
export function armarItems(renglones: RenglonEditable[]): ItemEditado[] {
  return ordenEnvio(renglones).map((r) => ({
    ...(r.nuevo ? {} : { id: r.id }), productoId: r.productoId, cantidad: Number(r.cantidad), costoUnitario: r.costo!,
    precioVentaNuevo: r.precioNuevo, vencimiento: r.vencimiento || null,
  }));
}

export function totalRenglones(renglones: RenglonEditable[]): string {
  return deCentavos(activos(renglones).reduce((t, r) => t + aCentavos(r.costo) * (Number(r.cantidad) || 0), 0));
}

/** Efectos antes de guardar: "El stock de Kaiak sube 2", "Se agrega X (+N al stock)", "Se quita X (−N del stock)". */
export function resumenEfectos(renglones: RenglonEditable[]): string[] {
  const out: string[] = [];
  for (const r of renglones) {
    if (r.quitado) {
      out.push(`Se quita ${r.producto} (−${r.cantidadOriginal} del stock)`);
      continue;
    }
    const d = (Number(r.cantidad) || 0) - r.cantidadOriginal;
    if (r.nuevo) {
      if (d > 0) {
        out.push(`Se agrega ${r.producto} (+${d} al stock)`);
      }
    } else if (d > 0) {
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

/** Datos legibles de un renglón AGREGADO/QUITADO del historial: cantidad "×2" y detalle (costo, precio, vencimiento). */
export function detalleRenglonHistorial(r: CambioRenglon): { cantidad: string | null; resto: string[] } {
  const lado = r.accion === 'QUITADO' ? 'antes' : 'despues';
  const valor = (campo: string) => r.cambios.find((c) => c.campo === campo)?.[lado] ?? null;
  const cant = valor('cantidad');
  const resto: string[] = [];
  const costo = valor('costoUnitario');
  if (costo !== null) {
    resto.push(`costo ${formatearValorCambio('costoUnitario', costo)}`);
  }
  const precio = valor('precioVentaNuevo');
  if (precio !== null) {
    resto.push(`nuevo precio ${formatearValorCambio('precioVentaNuevo', precio)}`);
  }
  const venc = valor('vencimiento');
  if (venc !== null) {
    resto.push(`vence ${formatearValorCambio('vencimiento', venc)}`);
  }
  return { cantidad: cant === null ? null : `×${cant}`, resto };
}
