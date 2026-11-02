import { ChangeDetectionStrategy, Component, signal } from '@angular/core';
import { EncabezadoComponent } from '../../shared/ui/encabezado.component';
import { CajaActualComponent } from './caja-actual.component';
import { GastosComponent } from './gastos.component';
import { HistorialCajasComponent } from './historial-cajas.component';

type Pestana = 'actual' | 'historial' | 'gastos';

/** Caja: apertura y cierre con conciliación (US-002), historial de cajas y gastos. */
@Component({
  selector: 'app-caja',
  imports: [EncabezadoComponent, CajaActualComponent, GastosComponent, HistorialCajasComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-encabezado titulo="Caja" descripcion="Apertura y cierre del día, cajas anteriores y gastos" />

    <div role="tablist" aria-label="Secciones de caja" class="mb-4 flex flex-wrap gap-2" (keydown)="teclas($event)">
      @for (p of pestanas; track p.id) {
        <button type="button" role="tab" [id]="'tab-' + p.id" [attr.aria-selected]="activa() === p.id"
                [attr.aria-controls]="'panel-' + p.id" [tabindex]="activa() === p.id ? 0 : -1"
                [class]="activa() === p.id ? 'chip chip-activo' : 'chip'" (click)="activa.set(p.id)">{{ p.etiqueta }}</button>
      }
    </div>

    <div role="tabpanel" [id]="'panel-' + activa()" [attr.aria-labelledby]="'tab-' + activa()">
      @switch (activa()) {
        @case ('actual') { <app-caja-actual /> }
        @case ('historial') { <app-historial-cajas /> }
        @case ('gastos') { <app-gastos /> }
      }
    </div>
  `,
})
export class CajaPage {
  protected readonly pestanas: readonly { id: Pestana; etiqueta: string }[] = [
    { id: 'actual', etiqueta: 'Caja del día' },
    { id: 'historial', etiqueta: 'Cajas anteriores' },
    { id: 'gastos', etiqueta: 'Gastos' },
  ];
  protected readonly activa = signal<Pestana>('actual');

  /** Flechas izquierda/derecha entre pestañas (patrón de pestañas accesible). */
  protected teclas(e: KeyboardEvent): void {
    if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') {
      return;
    }
    e.preventDefault();
    const i = this.pestanas.findIndex((p) => p.id === this.activa());
    const n = this.pestanas.length;
    const sig = this.pestanas[(i + (e.key === 'ArrowRight' ? 1 : n - 1)) % n];
    this.activa.set(sig.id);
    document.getElementById('tab-' + sig.id)?.focus();
  }
}
