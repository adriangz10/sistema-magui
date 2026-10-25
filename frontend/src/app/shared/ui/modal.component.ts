import { ChangeDetectionStrategy, Component, ElementRef, effect, input, output, viewChild } from '@angular/core';
import { LucideAngularModule, X } from 'lucide-angular';

/**
 * Ventana modal basada en <dialog> nativo (atrapa el foco, cierra con Escape).
 *
 *   <app-modal [abierto]="editando()" titulo="Editar producto" (cerrar)="editando.set(false)">
 *     ...contenido...
 *     <div pie> botones </div>
 *   </app-modal>
 */
@Component({
  selector: 'app-modal',
  imports: [LucideAngularModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    dialog::backdrop { background: rgb(2 6 23 / 0.55); backdrop-filter: blur(2px); }
  `,
  template: `
    <dialog #dialogo (cancel)="$event.preventDefault(); cerrar.emit()" (click)="clicFuera($event)"
            [attr.aria-labelledby]="idTitulo"
            class="m-auto w-[calc(100%-2rem)] rounded-2xl bg-white p-0 text-ink-900 shadow-2xl dark:bg-ink-900 dark:text-ink-100"
            [class]="ancho() === 'grande' ? 'max-w-3xl' : 'max-w-lg'">
      <div class="flex items-center justify-between border-b border-ink-100 px-5 py-4 dark:border-ink-800">
        <h2 [id]="idTitulo" class="text-lg font-semibold">{{ titulo() }}</h2>
        <button type="button" class="btn-fantasma btn-icono -mr-2" (click)="cerrar.emit()" aria-label="Cerrar">
          <lucide-icon [img]="iconoCerrar" [size]="20" aria-hidden="true" />
        </button>
      </div>
      <div class="max-h-[70vh] overflow-y-auto px-5 py-4">
        <ng-content />
      </div>
      <div class="flex flex-wrap justify-end gap-2 border-t border-ink-100 px-5 py-4 empty:hidden dark:border-ink-800">
        <ng-content select="[pie]" />
      </div>
    </dialog>
  `,
})
export class ModalComponent {
  readonly abierto = input(false);
  readonly titulo = input.required<string>();
  readonly ancho = input<'normal' | 'grande'>('normal');
  readonly cerrar = output<void>();

  private static contador = 0;
  protected readonly idTitulo = `modal-titulo-${++ModalComponent.contador}`;
  protected readonly iconoCerrar = X;
  private readonly dialogo = viewChild.required<ElementRef<HTMLDialogElement>>('dialogo');

  constructor() {
    effect(() => {
      const d = this.dialogo().nativeElement;
      if (this.abierto() && !d.open) {
        d.showModal();
      } else if (!this.abierto() && d.open) {
        d.close();
      }
    });
  }

  protected clicFuera(e: MouseEvent): void {
    if (e.target === this.dialogo().nativeElement) {
      this.cerrar.emit();
    }
  }
}
