import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { LucideAngularModule, ScanBarcode } from 'lucide-angular';
import { LectorCodigosService } from '../../core/hardware/lector-codigos.service';
import { FechaPipe } from '../../shared/formato/fecha';

/** Estado y prueba del lector de códigos de barras (US-003). */
@Component({
  selector: 'app-lector-config',
  imports: [LucideAngularModule, FechaPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="card p-5" aria-labelledby="titulo-lector">
      <h2 id="titulo-lector" class="font-semibold">Lector de códigos de barras</h2>
      <p class="mt-1 mb-3 text-sm text-muted">
        Enchufalo (USB o Bluetooth): no hace falta instalar nada. En Vender, cada código que leas se agrega al carrito
        aunque el cursor no esté en el buscador.
      </p>
      <div tabindex="0" role="status" aria-live="polite" aria-describedby="lector-ayuda"
           class="flex min-h-24 flex-col items-center justify-center gap-1 rounded-xl border-2 border-dashed border-ink-200 p-4 text-center
                  focus-visible:border-brand-600 focus-visible:outline-none dark:border-ink-700">
        <lucide-icon [img]="icono" [size]="24" class="text-muted" aria-hidden="true" />
        @if (lector.ultima(); as l) {
          <p class="text-sm text-ok">Lector funcionando ✓</p>
          <p class="font-mono text-lg font-semibold">{{ l.codigo }}</p>
          <p class="text-xs text-muted">Leído a las {{ l.en | fecha: 'hora' }}</p>
        } @else {
          <p class="text-sm font-medium">Tocá acá y escaneá cualquier código para probarlo</p>
        }
      </div>
      <p id="lector-ayuda" class="mt-2 text-xs text-muted">
        @if (lector.detectado()) {
          En este dispositivo ya se detectó un lector.
        } @else {
          Todavía no se detectó un lector en este dispositivo. Si no responde, revisá que esté configurado para terminar cada lectura con Enter.
        }
      </p>
    </section>
  `,
})
export class LectorConfigComponent {
  protected readonly lector = inject(LectorCodigosService);
  protected readonly icono = ScanBarcode;
}
