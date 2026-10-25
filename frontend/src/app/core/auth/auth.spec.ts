import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { ApiError } from '../api/api-error';
import { authInterceptor } from './auth.interceptor';
import { AuthService } from './auth.service';

describe('AuthService e interceptor', () => {
  let http: HttpTestingController;
  let auth: AuthService;
  let cliente: HttpClient;
  const futuro = new Date(Date.now() + 3600_000).toISOString();

  beforeEach(() => {
    localStorage.clear();
    TestBed.configureTestingModule({
      providers: [provideRouter([]), provideHttpClient(withInterceptors([authInterceptor])), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    auth = TestBed.inject(AuthService);
    cliente = TestBed.inject(HttpClient);
  });

  afterEach(() => http.verify());

  async function ingresar(): Promise<void> {
    const p = auth.iniciarSesion('secreta123');
    const req = http.expectOne('/api/auth/login');
    expect(req.request.body).toEqual({ clave: 'secreta123' });
    expect(req.request.headers.has('Authorization')).toBe(false);
    req.flush({ token: 'tok-1', expiraEn: futuro });
    await p;
  }

  it('inicia sesión, guarda el token y lo envía en cada llamada', async () => {
    await ingresar();
    expect(auth.autenticada()).toBe(true);
    expect(JSON.parse(localStorage.getItem('magui.sesion')!).token).toBe('tok-1');

    cliente.get('/api/productos').subscribe();
    expect(http.expectOne('/api/productos').request.headers.get('Authorization')).toBe('Bearer tok-1');
  });

  it('informa el error de contraseña incorrecta', async () => {
    const p = auth.iniciarSesion('mal');
    http.expectOne('/api/auth/login').flush(
      { error: { code: 'clave_incorrecta', message: 'La contraseña es incorrecta' } },
      { status: 401, statusText: 'Unauthorized' },
    );
    await expect(p).rejects.toMatchObject({ codigo: 'clave_incorrecta', message: 'La contraseña es incorrecta' });
    expect(auth.autenticada()).toBe(false);
  });

  it('cierra la sesión si la API responde 401', async () => {
    await ingresar();
    const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    cliente.get('/api/ventas').subscribe({ error: () => undefined });
    http.expectOne('/api/ventas').flush({ error: { code: 'sesion_invalida', message: 'x' } }, { status: 401, statusText: 'x' });
    expect(auth.autenticada()).toBe(false);
    expect(localStorage.getItem('magui.sesion')).toBeNull();
    expect(navegar).toHaveBeenCalledWith(['/ingresar'], expect.anything());
  });

  it('no considera válida una sesión vencida', () => {
    localStorage.setItem('magui.sesion', JSON.stringify({ token: 't', expiraEn: '2020-01-01T00:00:00Z' }));
    TestBed.resetTestingModule();
    TestBed.configureTestingModule({ providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()] });
    expect(TestBed.inject(AuthService).autenticada()).toBe(false);
  });
});

describe('ApiError', () => {
  it('interpreta errores sin conexión y con campos', async () => {
    const { HttpErrorResponse } = await import('@angular/common/http');
    expect(ApiError.desde(new HttpErrorResponse({ status: 0 })).codigo).toBe('sin_conexion');
    const e = ApiError.desde(new HttpErrorResponse({
      status: 422, error: { error: { code: 'datos_invalidos', message: 'Revisá los datos', fields: { nombre: 'Obligatorio' } } },
    }));
    expect(e.campos['nombre']).toBe('Obligatorio');
    expect(e.detalle).toBe('Revisá los datos: Obligatorio');
  });
});
