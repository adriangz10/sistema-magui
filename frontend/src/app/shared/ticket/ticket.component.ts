import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { ConfigTicket } from '../../core/api/negocio.api';
import { MetodoPago, VentaDetalle } from '../../core/api/ventas.api';
import { DineroPipe } from '../formato/dinero';
import { FechaPipe } from '../formato/fecha';

const NOMBRE_MEDIO: Record<MetodoPago, string> = { EFECTIVO: 'Efectivo', TRANSFERENCIA: 'Transferencia', QR: 'QR', TARJETA: 'Tarjeta' };

/** Número corto para el ticket: las últimas 8 letras del ID de la venta. */
export function numeroTicket(id: string): string {
  return id.replace(/-/g, '').slice(-8).toUpperCase();
}

/**
 * Ticket de una venta para ticketera térmica (58 u 80 mm). No es factura.
 * Se usa para imprimir y como vista previa en Configuración.
 */
@Component({
  selector: 'app-ticket',
  imports: [DineroPipe, FechaPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    :host { display: block; }
    .ticket {
      box-sizing: border-box; padding: 2mm 3mm; background: #fff; color: #000;
      font-family: ui-monospace, 'Cascadia Mono', Consolas, 'Courier New', monospace; line-height: 1.3;
    }
    .ancho-58 { width: 58mm; font-size: 10.5px; }
    .ancho-80 { width: 80mm; font-size: 12px; }
    .centro { text-align: center; }
    .nombre { font-size: 1.35em; font-weight: 700; margin: 1mm 0; }
    .logo { display: block; margin: 0 auto 1mm; max-width: 60%; max-height: 22mm; object-fit: contain; filter: grayscale(1); }
    .sep { border: 0; border-top: 1px dashed #000; margin: 1.5mm 0; }
    .fila { display: flex; justify-content: space-between; gap: 2mm; }
    .fila > :last-child { text-align: right; white-space: nowrap; }
    .item { margin-bottom: 0.8mm; }
    .detalle { padding-left: 2mm; }
    .total { font-size: 1.3em; font-weight: 700; }
    .multilinea { white-space: pre-line; }
    .anulada { font-weight: 700; border: 1px solid #000; padding: 0.5mm; margin: 1mm 0; }
    .chico { font-size: 0.85em; }
  `,
  template: `
    <div class="ticket" [class]="'ticket ancho-' + config().anchoPapel">
      <div class="centro">
        @if (config().mostrarLogo && logo()) {
          <img class="logo" [src]="logo()" alt="" />
        }
        <p class="nombre">{{ nombre() }}</p>
        @if (config().datos) { <p class="multilinea">{{ config().datos }}</p> }
      </div>
      <hr class="sep" />
      <div class="fila"><span>{{ venta().fecha | fecha }}</span><span>N° {{ numero() }}</span></div>
      @if (venta().estado === 'ANULADA') { <p class="centro anulada">VENTA ANULADA</p> }
      <hr class="sep" />
      @for (it of venta().items; track $index) {
        <div class="item">
          <div>{{ it.producto }}</div>
          <div class="fila detalle"><span>{{ it.cantidad }} x {{ it.precioUnitario | dinero }}</span><span>{{ it.subtotal | dinero }}</span></div>
        </div>
      }
      <hr class="sep" />
      <div class="fila total"><span>TOTAL</span><span>{{ venta().total | dinero }}</span></div>
      @for (p of venta().pagos; track $index) {
        <div class="fila"><span>{{ medio(p.metodo) }}</span><span>{{ (p.montoRecibido ?? p.monto) | dinero }}</span></div>
      }
      @if (venta().vuelto !== '0.00') {
        <div class="fila"><span>Vuelto</span><span>{{ venta().vuelto | dinero }}</span></div>
      }
      <hr class="sep" />
      @if (config().pie) { <p class="centro multilinea">{{ config().pie }}</p> }
      <p class="centro chico">Comprobante no válido como factura</p>
    </div>
  `,
})
export class TicketComponent {
  readonly venta = input.required<VentaDetalle>();
  readonly config = input.required<ConfigTicket>();
  readonly nombre = input.required<string>();
  readonly logo = input<string | null>(null);
  protected readonly numero = computed(() => numeroTicket(this.venta().id));

  protected medio(m: MetodoPago): string {
    return NOMBRE_MEDIO[m] ?? m;
  }
}
