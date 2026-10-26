import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { EscritorioService } from './escritorio.service';

describe('EscritorioService', () => {
  it('en la web no consulta nada y no muestra pendientes', () => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    const svc = TestBed.inject(EscritorioService);
    const http = TestBed.inject(HttpTestingController);
    expect(svc.activo).toBe(false);
    http.expectNone('/api/escritorio/estado');
    expect(svc.pendientes()).toBe(0);
  });

  it('actualizar() trae el estado de la sincronización', async () => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    const svc = TestBed.inject(EscritorioService);
    const http = TestBed.inject(HttpTestingController);
    const p = svc.actualizar();
    http.expectOne('/api/escritorio/estado').flush({ enLinea: false, pendientes: 3, rechazadas: 1, necesitaIngresar: false,
      ultimaSincronizacion: null, copiaDelCatalogo: null });
    await p;
    expect(svc.pendientes()).toBe(3);
    expect(svc.rechazadas()).toBe(1);
  });
});

describe('noIndicaConexion', () => {
  it('ignora lo que responde la copia o el núcleo local', async () => {
    const { noIndicaConexion } = await import('../red/conexion.interceptor');
    expect(noIndicaConexion('GET', '/api/productos?q=x')).toBe(true);
    expect(noIndicaConexion('POST', '/api/ventas')).toBe(false); // en la web, cobrar prueba que hay servidor
    expect(noIndicaConexion('POST', '/api/ventas', true)).toBe(true); // en el escritorio se cobra en local
    expect(noIndicaConexion('GET', '/api/escritorio/estado', true)).toBe(true);
    expect(noIndicaConexion('GET', '/api/ventas?page=1', true)).toBe(false); // el listado sale del servidor
    expect(noIndicaConexion('POST', '/api/ventas/abc/anular', true)).toBe(false);
  });
});
