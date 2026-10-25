import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { CircleAlert, CircleCheck, Info, LucideAngularModule, X } from 'lucide-angular';
import { ConfirmacionService } from '../../core/ui/confirmacion.service';
import { ToastService } from '../../core/ui/toast.service';
import { ModalComponent } from './modal.component';

/** Muestra los avisos (toasts) y el diálogo de confirmación. Se incluye una vez en la app. */
@Component({
  selector: 'app-avisos',
  imports: [LucideAngularModule, ModalComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="pointer-events-none fixed inset-x-0 bottom-20 z-50 flex flex-col items-center gap-2 px-4 lg:bottom-6 lg:items-end"
         aria-live="polite">
      @for (t of toasts.toasts(); track t.id) {
        <div class="pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-xl border px-4 py-3 text-sm shadow-lg"
             [class]="clases[t.tipo]" [attr.role]="t.tipo === 'error' ? 'alert' : 'status'">
          <lucide-icon [img]="iconos[t.tipo]" [size]="18" class="mt-0.5 shrink-0" aria-hidden="true" />
          <p class="flex-1">{{ t.mensaje }}</p>
          <button type="button" class="-m-1 rounded p-1 opacity-70 hover:opacity-100" (click)="toasts.cerrar(t.id)" aria-label="Cerrar aviso">
            <lucide-icon [img]="iconoCerrar" [size]="16" aria-hidden="true" />
          </button>
        </div>
      }
    </div>

    @if (confirmacion.activo(); as c) {
      <app-modal [abierto]="true" [titulo]="c.titulo" (cerrar)="c.responder(false)">
        <p class="text-ink-600 dark:text-ink-300">{{ c.mensaje }}</p>
        <div pie>
          <button type="button" class="btn-secundario" (click)="c.responder(false)">{{ c.cancelar ?? 'Cancelar' }}</button>
          <button type="button" [class]="c.peligro ? 'btn-peligro' : 'btn-primario'" (click)="c.responder(true)">
            {{ c.confirmar ?? 'Confirmar' }}
          </button>
        </div>
      </app-modal>
    }
  `,
})
export class AvisosComponent {
  protected readonly toasts = inject(ToastService);
  protected readonly confirmacion = inject(ConfirmacionService);
  protected readonly iconoCerrar = X;
  protected readonly iconos = { exito: CircleCheck, error: CircleAlert, info: Info };
  protected readonly clases = {
    exito: 'border-green-200 bg-green-50 text-green-900 dark:border-green-900 dark:bg-green-950 dark:text-green-100',
    error: 'border-red-200 bg-red-50 text-red-900 dark:border-red-900 dark:bg-red-950 dark:text-red-100',
    info: 'border-ink-200 bg-white text-ink-900 dark:border-ink-700 dark:bg-ink-900 dark:text-ink-100',
  };
}
