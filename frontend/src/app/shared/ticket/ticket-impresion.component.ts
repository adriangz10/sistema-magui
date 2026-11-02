import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { NegocioService } from '../../core/negocio/negocio.service';
import { ImpresionTicketService } from '../../core/ticket/impresion-ticket.service';
import { TicketComponent } from './ticket.component';

/** Lugar del ticket a imprimir: va en la raíz de la app y solo se ve al imprimir (ver styles.css). */
@Component({
  selector: 'app-ticket-impresion',
  imports: [TicketComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'ticket-impresion', 'aria-hidden': 'true' },
  template: `
    @if (impresion.venta(); as v) {
      <app-ticket [venta]="v" [config]="negocio.ticket()" [nombre]="negocio.nombre()" [logo]="negocio.logo()" />
    }
  `,
})
export class TicketImpresionComponent {
  protected readonly impresion = inject(ImpresionTicketService);
  protected readonly negocio = inject(NegocioService);
}
