import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { LectorCodigosService } from './core/hardware/lector-codigos.service';
import { NegocioService } from './core/negocio/negocio.service';
import { ActualizacionService } from './core/pwa/actualizacion.service';
import { InstalacionService } from './core/pwa/instalacion.service';
import { TemaService } from './core/ui/tema.service';
import { TicketImpresionComponent } from './shared/ticket/ticket-impresion.component';
import { AvisosComponent } from './shared/ui/avisos.component';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, AvisosComponent, TicketImpresionComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<router-outlet /><app-avisos /><app-ticket-impresion />`,
})
export class App {
  constructor() {
    inject(TemaService); // aplica el modo claro/oscuro desde el inicio
    inject(InstalacionService); // escucha desde el arranque si la app se puede instalar
    inject(ActualizacionService).iniciar();
    inject(LectorCodigosService).iniciar(); // lector de códigos de barras en toda la app
    void inject(NegocioService).cargarMarca(); // nombre y logo, también en la pantalla de ingreso
  }
}
