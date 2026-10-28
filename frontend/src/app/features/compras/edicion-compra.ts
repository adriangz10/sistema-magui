import { CambioCampo, CompraDetalle, EditarCompra } from '../../core/api/compras.api';
import { OrigenOperacion } from '../../core/config/origen';
import { formatearDinero } from '../../shared/formato/dinero';
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
export function armarEdicion(original: CompraDetalle, actual: CabeceraEditable, origen: OrigenOperacion): EditarCompra {
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
