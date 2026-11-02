import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { SesionCaja } from '../../core/api/caja.api';
import { DineroPipe } from '../../shared/formato/dinero';
import { claseDiferencia, NOMBRE_MEDIO, textoDiferencia } from './arqueo';

/** Arqueo de una caja por medio de pago: sistema contra lo contado (US-002). */
@Component({
  selector: 'app-arqueo-tabla',
  imports: [DineroPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="overflow-x-auto">
      <table class="tabla">
        <caption class="sr-only">Arqueo por medio de pago</caption>
        <thead>
          <tr>
            <th scope="col">Medio</th>
            <th scope="col" class="text-right">Ventas</th>
            <th scope="col" class="text-right">Gastos</th>
            <th scope="col" class="text-right">Según el sistema</th>
            @if (cerrada()) {
              <th scope="col" class="text-right">Contado</th>
              <th scope="col" class="text-right">Diferencia</th>
            }
          </tr>
        </thead>
        <tbody>
          @for (a of sesion().arqueo; track a.metodo) {
            <tr>
              <th scope="row" class="font-medium">
                {{ nombre[a.metodo] }}
                @if (a.metodo === 'EFECTIVO') {
                  <span class="block text-xs font-normal text-muted">Incluye {{ sesion().montoInicial | dinero }} iniciales</span>
                }
              </th>
              <td class="text-right tabular-nums">{{ a.ventas | dinero }}</td>
              <td class="text-right tabular-nums">{{ a.gastos | dinero }}</td>
              <td class="text-right font-semibold tabular-nums">{{ a.esperado | dinero }}</td>
              @if (cerrada()) {
                <td class="text-right tabular-nums">{{ a.contado | dinero }}</td>
                <td class="text-right"><span [class]="clase(a.diferencia)">{{ texto(a.diferencia) }}</span></td>
              }
            </tr>
          }
        </tbody>
        <tfoot>
          <tr class="border-t border-ink-200 font-semibold dark:border-ink-700">
            <th scope="row">Total</th>
            <td></td>
            <td></td>
            <td class="text-right tabular-nums">{{ sesion().totalEsperado | dinero }}</td>
            @if (cerrada()) {
              <td class="text-right tabular-nums">{{ sesion().totalContado | dinero }}</td>
              <td class="text-right"><span [class]="clase(sesion().diferencia)">{{ texto(sesion().diferencia) }}</span></td>
            }
          </tr>
        </tfoot>
      </table>
    </div>
  `,
})
export class ArqueoTablaComponent {
  readonly sesion = input.required<SesionCaja>();
  protected readonly cerrada = computed(() => !this.sesion().abierta);
  protected readonly nombre = NOMBRE_MEDIO;
  protected readonly texto = textoDiferencia;
  protected readonly clase = claseDiferencia;
}
