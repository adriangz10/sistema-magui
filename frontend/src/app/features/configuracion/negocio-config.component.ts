import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ApiError } from '../../core/api/api-error';
import { AnchoPapel, ConfigTicket, NegocioApi } from '../../core/api/negocio.api';
import { VentaDetalle } from '../../core/api/ventas.api';
import { NegocioService } from '../../core/negocio/negocio.service';
import { ImpresionTicketService } from '../../core/ticket/impresion-ticket.service';
import { ConfirmacionService } from '../../core/ui/confirmacion.service';
import { ToastService } from '../../core/ui/toast.service';
import { TicketComponent } from '../../shared/ticket/ticket.component';
import { CampoComponent } from '../../shared/ui/campo.component';
import { LogoNegocioComponent } from '../../shared/ui/logo-negocio.component';

const TAMANO_MAXIMO_LOGO = 512 * 1024;
const TIPOS_LOGO = ['image/png', 'image/jpeg', 'image/webp'];

/** Venta de ejemplo para la vista previa y la impresión de prueba. */
function ventaDeEjemplo(): VentaDetalle {
  return {
    id: '00000000-0000-0000-0000-000000000001', fecha: new Date().toISOString(), total: '17500.00', estado: 'CONFIRMADA',
    origen: 'WEB', productos: 2, unidades: 3, metodos: ['EFECTIVO'], costoTotal: '0.00', ganancia: '0.00', costoCompleto: true,
    vuelto: '2500.00', anuladaEn: null, motivoAnulacion: null,
    items: [
      { productoId: 'a', sku: 'EJ1', producto: 'Producto de ejemplo', tipo: 'SIMPLE', cantidad: 1, precioUnitario: '12000.00', subtotal: '12000.00' },
      { productoId: 'b', sku: 'EJ2', producto: 'Otro producto', tipo: 'SIMPLE', cantidad: 2, precioUnitario: '2750.00', subtotal: '5500.00' },
    ],
    pagos: [{ metodo: 'EFECTIVO', monto: '17500.00', montoRecibido: '20000.00', vuelto: '2500.00' }],
  };
}

/** Nombre, logo y ticket de cada venta (US-003). */
@Component({
  selector: 'app-negocio-config',
  imports: [FormsModule, CampoComponent, LogoNegocioComponent, TicketComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="card p-5" aria-labelledby="titulo-negocio">
      <h2 id="titulo-negocio" class="font-semibold">Negocio y ticket</h2>
      <p class="mb-4 text-sm text-muted">El nombre y el logo se ven en el menú, en la pantalla de ingreso y en el ticket.</p>

      <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_auto]">
        <form id="form-negocio" class="space-y-4" (ngSubmit)="guardar()">
          <app-campo etiqueta="Nombre del negocio" para="ng-nombre" [error]="errores()['nombre']">
            <input id="ng-nombre" name="nombre" class="input" maxlength="60" [(ngModel)]="nombre"
                   [attr.aria-invalid]="errores()['nombre'] ? true : null" [attr.aria-describedby]="errores()['nombre'] ? 'ng-nombre-error' : null" />
          </app-campo>

          <div>
            <p class="etiqueta" id="ng-logo-etiqueta">Logo</p>
            <div class="flex flex-wrap items-center gap-3">
              <app-logo-negocio tamano="grande" />
              <label class="btn-secundario cursor-pointer" for="ng-logo">{{ negocio.logo() ? 'Cambiar logo' : 'Subir logo' }}</label>
              <input id="ng-logo" type="file" class="sr-only" accept="image/png,image/jpeg,image/webp"
                     aria-describedby="ng-logo-ayuda" (change)="elegirLogo($event)" [disabled]="subiendo()" />
              @if (negocio.logo()) {
                <button type="button" class="btn-fantasma" (click)="quitarLogo()" [disabled]="subiendo()">Quitar</button>
              }
            </div>
            <p id="ng-logo-ayuda" class="mt-1 text-xs" [class]="errorLogo() ? 'text-danger' : 'text-muted'" [attr.role]="errorLogo() ? 'alert' : null">
              {{ errorLogo() || 'PNG, JPG o WEBP de hasta 512 KB. Mejor cuadrado y con fondo blanco o transparente.' }}
            </p>
          </div>

          <fieldset class="space-y-3 border-t border-ink-100 pt-4 dark:border-ink-800">
            <legend class="sr-only">Ticket de venta</legend>
            <label class="flex items-start gap-3">
              <input type="checkbox" name="habilitado" class="mt-0.5 size-4 accent-brand-600" [ngModel]="ticket().habilitado"
                     (ngModelChange)="cambiarTicket({ habilitado: $event })" />
              <span><span class="font-medium">Emitir ticket de cada venta</span>
                <span class="block text-sm text-muted">Para una ticketera instalada en esta computadora (o cualquier impresora).</span></span>
            </label>
            @if (ticket().habilitado) {
              <label class="flex items-start gap-3 pl-7">
                <input type="checkbox" name="alCobrar" class="mt-0.5 size-4 accent-brand-600" [ngModel]="ticket().imprimirAlCobrar"
                       (ngModelChange)="cambiarTicket({ imprimirAlCobrar: $event })" />
                <span><span class="font-medium">Imprimir apenas se cobra</span>
                  <span class="block text-sm text-muted">Si no, queda el botón "Imprimir ticket" al terminar la venta.</span></span>
              </label>
              <div class="grid gap-4 pl-7 sm:grid-cols-2">
                <div>
                  <p class="etiqueta" id="ng-ancho">Ancho del papel</p>
                  <div class="flex gap-2" role="radiogroup" aria-labelledby="ng-ancho">
                    @for (a of anchos; track a) {
                      <button type="button" role="radio" class="chip flex-1" [attr.aria-checked]="ticket().anchoPapel === a"
                              [class.chip-activo]="ticket().anchoPapel === a" (click)="cambiarTicket({ anchoPapel: a })">{{ a }} mm</button>
                    }
                  </div>
                </div>
                <label class="flex items-center gap-3 self-end pb-2">
                  <input type="checkbox" name="mostrarLogo" class="size-4 accent-brand-600" [ngModel]="ticket().mostrarLogo"
                         (ngModelChange)="cambiarTicket({ mostrarLogo: $event })" />
                  <span class="font-medium">Logo en el ticket</span>
                </label>
                <app-campo class="sm:col-span-2" etiqueta="Datos debajo del nombre" para="ng-datos" [opcional]="true"
                           [error]="errores()['ticket.datos']" ayuda="Dirección, teléfono, CUIT, redes... uno por línea (máximo 6).">
                  <textarea id="ng-datos" name="datos" class="input min-h-20 py-2" maxlength="300" [ngModel]="ticket().datos ?? ''"
                            (ngModelChange)="cambiarTicket({ datos: $event })"
                            [attr.aria-describedby]="errores()['ticket.datos'] ? 'ng-datos-error' : 'ng-datos-ayuda'"></textarea>
                </app-campo>
                <app-campo class="sm:col-span-2" etiqueta="Mensaje al pie" para="ng-pie" [opcional]="true" [error]="errores()['ticket.pie']">
                  <textarea id="ng-pie" name="pie" class="input min-h-14 py-2" maxlength="200" [ngModel]="ticket().pie ?? ''"
                            (ngModelChange)="cambiarTicket({ pie: $event })"></textarea>
                </app-campo>
              </div>
            }
          </fieldset>

          <div class="flex flex-wrap justify-end gap-2">
            @if (ticket().habilitado) {
              <button type="button" class="btn-secundario" (click)="imprimirPrueba()">Imprimir prueba</button>
            }
            <button type="submit" class="btn-primario" [disabled]="guardando()">Guardar</button>
          </div>
        </form>

        @if (ticket().habilitado) {
          <div aria-label="Vista previa del ticket" role="img" class="justify-self-center">
            <p class="mb-2 text-center text-xs text-muted" aria-hidden="true">Vista previa</p>
            <div class="overflow-hidden rounded-md shadow-md ring-1 ring-ink-200 dark:ring-ink-700" aria-hidden="true">
              <app-ticket [venta]="ejemplo" [config]="ticket()" [nombre]="nombre || negocio.nombre()" [logo]="negocio.logo()" />
            </div>
          </div>
        }
      </div>
    </section>
  `,
})
export class NegocioConfigComponent {
  private readonly api = inject(NegocioApi);
  private readonly toast = inject(ToastService);
  private readonly confirmacion = inject(ConfirmacionService);
  private readonly impresion = inject(ImpresionTicketService);
  protected readonly negocio = inject(NegocioService);

  protected readonly anchos: readonly AnchoPapel[] = [58, 80];
  protected readonly ejemplo = ventaDeEjemplo();
  protected nombre = this.negocio.nombre();
  protected readonly borrador = signal<ConfigTicket>(this.negocio.ticket());
  protected readonly ticket = computed(() => this.borrador());
  protected readonly errores = signal<Record<string, string>>({});
  protected readonly errorLogo = signal('');
  protected readonly guardando = signal(false);
  protected readonly subiendo = signal(false);

  constructor() {
    // Datos frescos del servidor (la copia local puede ser vieja).
    void this.negocio.cargar().then(() => {
      this.nombre = this.negocio.nombre();
      this.borrador.set(this.negocio.ticket());
    });
  }

  protected cambiarTicket(c: Partial<ConfigTicket>): void {
    this.borrador.update((t) => ({ ...t, ...c }));
  }

  protected async guardar(): Promise<void> {
    if (!this.nombre.trim()) {
      this.errores.set({ nombre: 'Ingresá el nombre del negocio' });
      return;
    }
    this.errores.set({});
    this.guardando.set(true);
    try {
      const n = await firstValueFrom(this.api.guardar({ nombre: this.nombre, ticket: this.borrador() }));
      this.negocio.actualizar(n);
      this.nombre = n.nombre;
      this.borrador.set(n.ticket);
      this.toast.exito('Datos del negocio guardados');
    } catch (e) {
      const err = ApiError.desde(e);
      if (Object.keys(err.campos).length) {
        this.errores.set({ ...err.campos });
      } else {
        this.toast.error(err);
      }
    } finally {
      this.guardando.set(false);
    }
  }

  protected async elegirLogo(e: Event): Promise<void> {
    const campo = e.target as HTMLInputElement;
    const archivo = campo.files?.[0];
    campo.value = ''; // permite volver a elegir el mismo archivo
    if (!archivo) {
      return;
    }
    if (!TIPOS_LOGO.includes(archivo.type)) {
      this.errorLogo.set('Usá una imagen PNG, JPG o WEBP');
      return;
    }
    if (archivo.size > TAMANO_MAXIMO_LOGO) {
      this.errorLogo.set('La imagen es muy pesada: máximo 512 KB');
      return;
    }
    this.errorLogo.set('');
    this.subiendo.set(true);
    try {
      this.negocio.actualizar(await firstValueFrom(this.api.subirLogo(archivo)));
      this.toast.exito('Logo actualizado');
    } catch (err) {
      this.errorLogo.set(ApiError.desde(err).message);
    } finally {
      this.subiendo.set(false);
    }
  }

  protected async quitarLogo(): Promise<void> {
    if (!(await this.confirmacion.preguntar({ titulo: 'Quitar logo', mensaje: '¿Querés quitar el logo del negocio?', confirmar: 'Quitar', peligro: true }))) {
      return;
    }
    this.subiendo.set(true);
    try {
      this.negocio.actualizar(await firstValueFrom(this.api.quitarLogo()));
      this.toast.exito('Logo quitado');
    } catch (err) {
      this.toast.error(err);
    } finally {
      this.subiendo.set(false);
    }
  }

  /** Imprime el ticket de ejemplo con lo que está en pantalla (aunque todavía no se haya guardado). */
  protected async imprimirPrueba(): Promise<void> {
    const guardado = this.negocio.negocio();
    this.negocio.actualizar({ ...guardado, nombre: this.nombre || guardado.nombre, ticket: this.borrador() });
    try {
      await this.impresion.imprimir(this.ejemplo);
    } finally {
      this.negocio.actualizar(guardado);
    }
  }
}
