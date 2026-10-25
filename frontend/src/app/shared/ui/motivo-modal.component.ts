import { ChangeDetectionStrategy, Component, effect, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CampoComponent } from './campo.component';
import { ModalComponent } from './modal.component';

/**
 * Pide un motivo antes de una acción que se registra (anular, descartar...).
 *
 *   <app-motivo-modal [abierto]="anulando()" titulo="Anular venta" [error]="error()"
 *                     (confirmar)="anular($event)" (cerrar)="anulando.set(false)" />
 */
@Component({
  selector: 'app-motivo-modal',
  imports: [FormsModule, CampoComponent, ModalComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-modal [abierto]="abierto()" [titulo]="titulo()" (cerrar)="cerrar.emit()">
      @if (descripcion()) { <p class="mb-3 text-sm text-ink-600 dark:text-ink-300">{{ descripcion() }}</p> }
      <app-campo etiqueta="Motivo" [para]="idCampo" [error]="error() || errorLocal()">
        <textarea [id]="idCampo" class="input min-h-20 py-2" maxlength="255" [(ngModel)]="motivo" [placeholder]="placeholder()"
                  [attr.aria-invalid]="error() || errorLocal() ? true : null"
                  [attr.aria-describedby]="error() || errorLocal() ? idCampo + '-error' : null"></textarea>
      </app-campo>
      <div pie>
        <button type="button" class="btn-secundario" (click)="cerrar.emit()">Cancelar</button>
        <button type="button" [class]="peligro() ? 'btn-peligro' : 'btn-primario'" [disabled]="procesando()" (click)="enviar()">{{ textoConfirmar() }}</button>
      </div>
    </app-modal>
  `,
})
export class MotivoModalComponent {
  readonly abierto = input(false);
  readonly titulo = input.required<string>();
  readonly descripcion = input('');
  readonly placeholder = input('');
  readonly textoConfirmar = input('Confirmar');
  readonly peligro = input(true);
  readonly procesando = input(false);
  /** Error del servidor para mostrar debajo del campo. */
  readonly error = input('');
  readonly confirmar = output<string>();
  readonly cerrar = output<void>();

  private static contador = 0;
  protected readonly idCampo = `motivo-${++MotivoModalComponent.contador}`;
  protected motivo = '';
  protected readonly errorLocal = signal('');

  constructor() {
    // Cada vez que se abre, empieza vacío.
    effect(() => {
      if (this.abierto()) {
        this.motivo = '';
        this.errorLocal.set('');
      }
    });
  }

  protected enviar(): void {
    if (!this.motivo.trim()) {
      this.errorLocal.set('Contá el motivo');
      return;
    }
    this.errorLocal.set('');
    this.confirmar.emit(this.motivo.trim());
  }
}
