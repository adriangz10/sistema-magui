import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { CambioCampo, EdicionCompra } from '../../core/api/compras.api';
import { FechaPipe } from '../../shared/formato/fecha';
import { ETIQUETA_CAMPO, formatearValorCambio, ORIGEN_ETIQUETA } from './edicion-compra';

/** Historial de ediciones de una compra, de la más reciente a la más vieja. */
@Component({
  selector: 'app-historial-ediciones',
  imports: [FechaPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section aria-labelledby="hist-ed-titulo" class="mt-6">
      <h3 id="hist-ed-titulo" class="mb-2 text-sm font-semibold">Historial de ediciones</h3>
      @if (ediciones().length) {
        <ul class="divide-y divide-ink-100 dark:divide-ink-800">
          @for (e of ediciones(); track e.id) {
            <li class="py-3 text-sm">
              <p class="flex flex-wrap items-center gap-2">
                <strong class="tabular-nums">{{ e.fecha | fecha }}</strong>
                <span class="badge-neutro">{{ origen(e) }}</span>
              </p>
              @if (e.motivo) { <p class="text-muted">Motivo: {{ e.motivo }}</p> }
              <ul class="mt-1 space-y-0.5">
                @for (c of e.cabecera; track c.campo) {
                  <li>{{ etiqueta(c) }}: <span class="text-muted">{{ valor(c.campo, c.antes, c.despues) }}</span> → <strong>{{ valor(c.campo, c.despues, c.antes) }}</strong></li>
                }
                @for (r of e.renglones; track $index) {
                  <li>
                    {{ r.producto }}
                    @if (r.accion === 'AGREGADO') { <span class="badge-ok">Agregado</span> }
                    @else if (r.accion === 'QUITADO') { <span class="badge-danger">Quitado</span> }
                    @for (c of r.cambios; track c.campo) {
                      <span class="block pl-3">{{ etiqueta(c) }}: <span class="text-muted">{{ valor(c.campo, c.antes, c.despues) }}</span> → <strong>{{ valor(c.campo, c.despues, c.antes) }}</strong></span>
                    }
                  </li>
                }
              </ul>
            </li>
          }
        </ul>
      } @else {
        <p class="text-sm text-muted">Sin ediciones</p>
      }
    </section>
  `,
})
export class HistorialEdicionesComponent {
  readonly ediciones = input<EdicionCompra[]>([]);

  protected origen(e: EdicionCompra): string {
    return ORIGEN_ETIQUETA[e.origen] ?? e.origen;
  }
  protected etiqueta(c: CambioCampo): string {
    return ETIQUETA_CAMPO[c.campo] ?? c.campo;
  }
  protected valor(campo: string, v: CambioCampo['antes'], otro: CambioCampo['antes']): string {
    return formatearValorCambio(campo, v, otro);
  }
}
