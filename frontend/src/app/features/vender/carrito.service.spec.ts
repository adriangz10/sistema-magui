import { TestBed } from '@angular/core/testing';
import { CarritoService } from './carrito.service';

const kaiak = { id: 'k', sku: 'P1', nombre: 'Kaiak', tipo: 'SIMPLE' as const, precioVenta: '59200.00', stock: 3 };
const jabon = { id: 'j', sku: 'P2', nombre: 'Jabón', tipo: 'SIMPLE' as const, precioVenta: '5500.00', stock: 10 };

describe('CarritoService', () => {
  let c: CarritoService;

  beforeEach(() => {
    localStorage.clear();
    c = TestBed.inject(CarritoService);
  });

  it('agrega, suma cantidades y calcula el total', () => {
    c.agregar(kaiak);
    c.agregar(jabon);
    expect(c.agregar(jabon)).toBe(2);
    expect(c.unidades()).toBe(3);
    expect(c.totalCentavos()).toBe(7_020_000);
    c.cambiarCantidad('j', 0);
    expect(c.lineas().map((l) => l.productoId)).toEqual(['k']);
  });

  it('propone lo que falta al elegir un método y no duplica el efectivo', () => {
    c.agregar(kaiak);
    c.agregarPago('QR');
    c.cambiarMonto(0, '50000.00');
    const i = c.agregarPago('EFECTIVO');
    expect(c.pagos()[i].monto).toBe('9200.00');
    expect(c.agregarPago('EFECTIVO')).toBe(i);
    c.sumarBillete(i, 1000);
    expect(c.cobro()).toMatchObject({ falta: 0, vuelto: 100_000, listo: true });
  });

  it('un pago único con tarjeta acompaña los cambios del total', () => {
    c.agregar(jabon);
    c.agregarPago('TARJETA');
    c.agregar(jabon);
    expect(c.pagos()[0].monto).toBe('11000.00');
    expect(c.cobro().listo).toBe(true);
  });

  it('se guarda en el dispositivo y vacía los pagos al vaciar', () => {
    c.agregar(kaiak);
    c.agregarPago('EFECTIVO');
    expect(JSON.parse(localStorage.getItem('magui.carrito') ?? '[]')).toHaveLength(0); // el effect corre al detectar cambios
    TestBed.tick();
    expect(JSON.parse(localStorage.getItem('magui.carrito')!)).toHaveLength(1);
    c.vaciar();
    expect(c.pagos()).toEqual([]);
  });
});
