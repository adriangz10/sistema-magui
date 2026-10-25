import { Injectable, computed, effect, signal } from '@angular/core';
import { ProductoResumen } from '../../core/api/catalogo.api';
import { Monto } from '../../core/api/modelos';
import { MetodoPago } from '../../core/api/ventas.api';
import { almacenamiento } from '../../core/almacenamiento';
import { aCentavos, deCentavos } from '../../shared/formato/dinero';
import { PagoCargado, calcularCobro } from './cobro';

export interface LineaCarrito {
  productoId: string;
  sku: string;
  nombre: string;
  tipo: ProductoResumen['tipo'];
  precio: Monto;
  stock: number;
  cantidad: number;
}

const CLAVE = 'magui.carrito';
export const CANTIDAD_MAXIMA = 999;

/**
 * Carrito del punto de venta. Se guarda en el dispositivo para no perderlo al cambiar de
 * pantalla o recargar. Los precios definitivos los pone el servidor al cobrar.
 */
@Injectable({ providedIn: 'root' })
export class CarritoService {
  readonly lineas = signal<LineaCarrito[]>(leerGuardado());
  readonly pagos = signal<PagoCargado[]>([]);

  readonly totalCentavos = computed(() =>
    this.lineas().reduce((t, l) => t + aCentavos(l.precio) * l.cantidad, 0));
  readonly unidades = computed(() => this.lineas().reduce((t, l) => t + l.cantidad, 0));
  readonly cobro = computed(() => calcularCobro(this.totalCentavos(), this.pagos()));
  readonly vacio = computed(() => this.lineas().length === 0);

  constructor() {
    effect(() => almacenamiento.guardar(CLAVE, JSON.stringify(this.lineas())));
  }

  /** Agrega una unidad (o suma una si ya estaba). Devuelve la cantidad resultante. */
  agregar(p: Pick<ProductoResumen, 'id' | 'sku' | 'nombre' | 'tipo' | 'precioVenta' | 'stock'>): number {
    let cantidad = 1;
    this.lineas.update((ls) => {
      const i = ls.findIndex((l) => l.productoId === p.id);
      if (i >= 0) {
        cantidad = Math.min(ls[i].cantidad + 1, CANTIDAD_MAXIMA);
        // refresca precio y stock por si cambiaron
        return ls.map((l, j) => (j === i ? { ...l, cantidad, precio: p.precioVenta, stock: p.stock } : l));
      }
      return [...ls, { productoId: p.id, sku: p.sku, nombre: p.nombre, tipo: p.tipo, precio: p.precioVenta, stock: p.stock, cantidad }];
    });
    this.reajustarPagoUnico();
    return cantidad;
  }

  cambiarCantidad(productoId: string, cantidad: number): void {
    if (!Number.isFinite(cantidad)) {
      return;
    }
    const n = Math.trunc(cantidad);
    if (n <= 0) {
      this.quitar(productoId);
      return;
    }
    this.lineas.update((ls) => ls.map((l) => (l.productoId === productoId ? { ...l, cantidad: Math.min(n, CANTIDAD_MAXIMA) } : l)));
    this.reajustarPagoUnico();
  }

  quitar(productoId: string): void {
    this.lineas.update((ls) => ls.filter((l) => l.productoId !== productoId));
    if (this.vacio()) {
      this.pagos.set([]);
    } else {
      this.reajustarPagoUnico();
    }
  }

  vaciar(): void {
    this.lineas.set([]);
    this.pagos.set([]);
  }

  /**
   * Agrega un método de pago con lo que falta cobrar. Si ya hay efectivo y se elige
   * efectivo otra vez, no duplica (devuelve su posición para enfocarlo).
   */
  agregarPago(metodo: MetodoPago): number {
    const existente = metodo === 'EFECTIVO' ? this.pagos().findIndex((p) => p.metodo === 'EFECTIVO') : -1;
    if (existente >= 0) {
      return existente;
    }
    const falta = this.cobro().falta;
    this.pagos.update((ps) => [...ps, { metodo, monto: falta > 0 ? deCentavos(falta) : null }]);
    return this.pagos().length - 1;
  }

  cambiarMonto(indice: number, monto: string | null): void {
    this.pagos.update((ps) => ps.map((p, i) => (i === indice ? { ...p, monto } : p)));
  }

  /** Suma un billete al efectivo recibido. */
  sumarBillete(indice: number, pesos: number): void {
    const actual = aCentavos(this.pagos()[indice]?.monto);
    this.cambiarMonto(indice, deCentavos(actual + pesos * 100));
  }

  /** Completa un pago con lo que falta (o, en efectivo, con el monto exacto). */
  completar(indice: number): void {
    const pago = this.pagos()[indice];
    if (!pago) {
      return;
    }
    const otros = this.pagos().filter((_, i) => i !== indice).reduce((t, p) => t + aCentavos(p.monto), 0);
    this.cambiarMonto(indice, deCentavos(Math.max(0, this.totalCentavos() - otros)));
  }

  quitarPago(indice: number): void {
    this.pagos.update((ps) => ps.filter((_, i) => i !== indice));
  }

  /** Con un único pago que no es efectivo, acompaña los cambios del total. */
  private reajustarPagoUnico(): void {
    const ps = this.pagos();
    if (ps.length === 1 && ps[0].metodo !== 'EFECTIVO') {
      this.pagos.set([{ ...ps[0], monto: deCentavos(this.totalCentavos()) }]);
    }
  }
}

function leerGuardado(): LineaCarrito[] {
  try {
    const datos = JSON.parse(almacenamiento.leer(CLAVE) ?? '[]') as unknown;
    if (!Array.isArray(datos)) {
      return [];
    }
    return datos.filter((l): l is LineaCarrito =>
      typeof l?.productoId === 'string' && typeof l?.precio === 'string' && Number.isInteger(l?.cantidad) && l.cantidad > 0);
  } catch {
    return [];
  }
}
