import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ApiError } from '../../core/api/api-error';
import { CatalogoApi, Empresa } from '../../core/api/catalogo.api';
import { Pagina } from '../../core/api/modelos';
import { ToastService } from '../../core/ui/toast.service';
import { PaginadorComponent } from '../../shared/ui/paginador.component';

/** Empresas a las que se les compra (Natura, Avon, ...). */
@Component({
  selector: 'app-empresas',
  imports: [FormsModule, PaginadorComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="card" aria-labelledby="titulo-empresas">
      <div class="border-b border-ink-100 p-4 dark:border-ink-800">
        <h2 id="titulo-empresas" class="font-semibold">Empresas</h2>
        <p class="text-sm text-ink-500">A quién le comprás la mercadería. Es opcional en cada producto.</p>
        <form class="mt-3 flex gap-2" (ngSubmit)="crear()">
          <label for="nueva-empresa" class="sr-only">Nombre de la nueva empresa</label>
          <input id="nueva-empresa" name="nueva" class="input" placeholder="Ej.: Avon" maxlength="100" [(ngModel)]="nueva"
                 [attr.aria-invalid]="error() ? true : null" [attr.aria-describedby]="error() ? 'nueva-empresa-error' : null" />
          <button type="submit" class="btn-primario shrink-0" [disabled]="!nueva.trim()">Agregar</button>
        </form>
        @if (error()) { <p id="nueva-empresa-error" class="mt-1 text-sm text-danger dark:text-red-400" role="alert">{{ error() }}</p> }
      </div>
      <ul class="divide-y divide-ink-100 dark:divide-ink-800">
        @for (e of pagina()?.items; track e.id) {
          <li class="flex items-center gap-2 px-4 py-2.5">
            @if (editando() === e.id) {
              <label [for]="'empresa-' + e.id" class="sr-only">Nombre de {{ e.nombre }}</label>
              <input [id]="'empresa-' + e.id" class="input min-h-9" [(ngModel)]="nombreEditado" maxlength="100" (keydown.enter)="guardar(e)" (keydown.escape)="editando.set(null)" />
              <button type="button" class="btn-primario min-h-9 px-3" (click)="guardar(e)">Guardar</button>
              <button type="button" class="btn-fantasma min-h-9 px-3" (click)="editando.set(null)">Cancelar</button>
            } @else {
              <span class="flex-1" [class.text-ink-400]="!e.activo">{{ e.nombre }}</span>
              @if (!e.activo) { <span class="badge-neutro">Inactiva</span> }
              <button type="button" class="btn-fantasma min-h-8 px-3" (click)="empezarEdicion(e)" [attr.aria-label]="'Renombrar ' + e.nombre">Renombrar</button>
              <button type="button" class="btn-fantasma min-h-8 px-3" (click)="alternar(e)" [attr.aria-label]="(e.activo ? 'Desactivar ' : 'Activar ') + e.nombre">
                {{ e.activo ? 'Desactivar' : 'Activar' }}
              </button>
            }
          </li>
        } @empty {
          <li class="px-4 py-6 text-center text-sm text-ink-500">Todavía no hay empresas.</li>
        }
      </ul>
      @if (pagina(); as p) {
        <app-paginador [pagina]="p.page" [totalPaginas]="p.totalPages" [total]="p.total" (cambio)="cargar($event)" />
      }
    </section>
  `,
})
export class EmpresasComponent {
  private readonly api = inject(CatalogoApi);
  private readonly toast = inject(ToastService);

  protected readonly pagina = signal<Pagina<Empresa> | null>(null);
  protected readonly editando = signal<string | null>(null);
  protected readonly error = signal('');
  protected nueva = '';
  protected nombreEditado = '';

  constructor() {
    void this.cargar(1);
  }

  protected async cargar(page: number): Promise<void> {
    try {
      this.pagina.set(await firstValueFrom(this.api.empresas({ page })));
    } catch (e) {
      this.toast.error(e);
    }
  }

  protected async crear(): Promise<void> {
    this.error.set('');
    try {
      const e = await firstValueFrom(this.api.crearEmpresa(this.nueva));
      this.nueva = '';
      this.toast.exito(`${e.nombre} agregada`);
      void this.cargar(this.pagina()?.page ?? 1);
    } catch (e) {
      this.error.set(ApiError.desde(e).detalle);
    }
  }

  protected empezarEdicion(e: Empresa): void {
    this.nombreEditado = e.nombre;
    this.editando.set(e.id);
  }

  protected async guardar(e: Empresa): Promise<void> {
    await this.actualizar({ ...e, nombre: this.nombreEditado });
    this.editando.set(null);
  }

  protected async alternar(e: Empresa): Promise<void> {
    await this.actualizar({ ...e, activo: !e.activo });
  }

  private async actualizar(e: Empresa): Promise<void> {
    try {
      await firstValueFrom(this.api.actualizarEmpresa(e));
      void this.cargar(this.pagina()?.page ?? 1);
    } catch (err) {
      this.toast.error(err);
      void this.cargar(this.pagina()?.page ?? 1);
    }
  }
}
