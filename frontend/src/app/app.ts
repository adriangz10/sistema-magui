import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { TemaService } from './core/ui/tema.service';
import { AvisosComponent } from './shared/ui/avisos.component';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, AvisosComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<router-outlet /><app-avisos />`,
})
export class App {
  constructor() {
    inject(TemaService); // aplica el modo claro/oscuro desde el inicio
  }
}
