import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { InicioPage } from './inicio.page';

describe('InicioPage', () => {
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [InicioPage],
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
  });

  it('muestra "En línea" cuando el servidor responde', async () => {
    const fixture = TestBed.createComponent(InicioPage);
    http.expectOne('/api/health').flush({ status: 'ok', version: '1.2.3' });
    await fixture.whenStable();
    expect(fixture.nativeElement.textContent).toContain('En línea · versión 1.2.3');
  });

  it('muestra error cuando el servidor no responde', async () => {
    const fixture = TestBed.createComponent(InicioPage);
    http.expectOne('/api/health').error(new ProgressEvent('error'));
    await fixture.whenStable();
    expect(fixture.nativeElement.textContent).toContain('Sin conexión con el servidor');
  });
});
