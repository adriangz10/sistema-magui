import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { filter } from 'rxjs';
import { CloudUpload, Ellipsis, LogOut, LucideAngularModule, Moon, Sun, TriangleAlert, Wifi, WifiOff, X } from 'lucide-angular';
import { AuthService } from '../core/auth/auth.service';
import { EscritorioService } from '../core/escritorio/escritorio.service';
import { ConexionService } from '../core/red/conexion.service';
import { ConfirmacionService } from '../core/ui/confirmacion.service';
import { TemaService } from '../core/ui/tema.service';
import { MENU } from './navegacion';
import { NegocioService } from '../core/negocio/negocio.service';
import { LogoNegocioComponent } from '../shared/ui/logo-negocio.component';

/** Estructura de la app con sesión: menú lateral (computadora) y barra inferior (celular). */
@Component({
  selector: 'app-shell',
  host: { '(document:keydown)': 'teclas($event)' },
  imports: [RouterOutlet, RouterLink, RouterLinkActive, LucideAngularModule, LogoNegocioComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a href="#contenido" class="sr-only z-50 rounded-lg bg-ink-900 px-4 py-2 text-white focus:not-sr-only focus:fixed focus:top-2 focus:left-2">
      Saltar al contenido
    </a>

    <div class="flex min-h-full">
      <!-- Menú lateral (pantallas grandes) -->
      <aside class="sticky top-0 hidden h-screen w-64 shrink-0 flex-col border-r border-ink-200 bg-white lg:flex dark:border-ink-800 dark:bg-ink-900">
        <div class="flex items-center gap-3 px-5 py-5">
          <app-logo-negocio />
          <p class="min-w-0 truncate font-semibold leading-tight">{{ negocio.nombre() }}</p>
        </div>
        <nav class="flex-1 space-y-1 overflow-y-auto px-3" aria-label="Principal">
          @for (item of menu; track item.ruta) {
            <a [routerLink]="item.ruta" routerLinkActive="bg-ink-100 text-ink-900 dark:bg-ink-800 dark:text-ink-50"
               ariaCurrentWhenActive="page"
               class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-ink-600 transition-colors duration-150 hover:bg-ink-50 dark:text-ink-300 dark:hover:bg-ink-800/60">
              <lucide-icon [img]="item.icono" [size]="18" aria-hidden="true" />
              {{ item.etiqueta }}
            </a>
          }
        </nav>
        <div class="border-t border-ink-100 p-3 dark:border-ink-800">
          <button type="button" class="btn-fantasma w-full justify-start" (click)="salir()">
            <lucide-icon [img]="iconos.salir" [size]="18" aria-hidden="true" /> Salir
          </button>
        </div>
      </aside>

      <div class="flex min-w-0 flex-1 flex-col">
        <!-- Barra superior -->
        <header class="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-ink-200 bg-white px-4 lg:px-8 dark:border-ink-800 dark:bg-ink-900">
          <div class="flex items-center gap-2 lg:hidden">
            <app-logo-negocio tamano="chico" />
            <span class="max-w-[50vw] truncate font-semibold">{{ negocio.nombre() }}</span>
          </div>
          <div class="flex-1"></div>
          @if (escritorio.pendientes() > 0) {
            <span class="flex items-center gap-1.5 rounded-full bg-ink-100 px-2.5 py-1 text-xs font-medium text-ink-700 dark:bg-ink-800 dark:text-ink-200"
                  role="status" [attr.title]="'Ventas cobradas en esta PC que todavía no llegaron al servidor'">
              <lucide-icon [img]="iconos.subir" [size]="14" aria-hidden="true" />
              {{ escritorio.pendientes() }} {{ escritorio.pendientes() === 1 ? 'venta pendiente' : 'ventas pendientes' }} de subir
            </span>
          }
          <span class="flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium"
                [class]="conexion.enLinea() ? 'bg-ink-100 text-ink-700 dark:bg-ink-800 dark:text-ink-300' : 'bg-warn-soft text-warn'"
                role="status">
            <lucide-icon [img]="conexion.enLinea() ? iconos.enLinea : iconos.sinConexion" [size]="14" aria-hidden="true" />
            {{ conexion.enLinea() ? 'En línea' : 'Sin conexión' }}
          </span>
          <button type="button" class="btn-fantasma btn-icono" (click)="tema.alternar()"
                  [attr.aria-label]="tema.tema() === 'oscuro' ? 'Cambiar a modo claro' : 'Cambiar a modo oscuro'">
            <lucide-icon [img]="tema.tema() === 'oscuro' ? iconos.sol : iconos.luna" [size]="18" aria-hidden="true" />
          </button>
        </header>

        @if (!conexion.enLinea()) {
          <div class="aviso-warn rounded-none border-x-0 border-t-0 px-4 text-center lg:px-8" role="status">
            @if (escritorio.activo) {
              Sin conexión: podés seguir vendiendo; las ventas se suben solas cuando vuelva internet. El resto de las pantallas necesita conexión.
            } @else {
              Sin conexión: podés consultar productos y precios, pero para vender o registrar cambios hace falta internet.
            }
          </div>
        }
        @if (escritorio.rechazadas() > 0) {
          <div class="flex items-center justify-center gap-2 aviso-danger rounded-none border-x-0 border-t-0 px-4 lg:px-8" role="alert">
            <lucide-icon [img]="iconos.alerta" [size]="16" aria-hidden="true" />
            El servidor no aceptó {{ escritorio.rechazadas() === 1 ? '1 venta cobrada' : escritorio.rechazadas() + ' ventas cobradas' }} en esta PC. Revisalas con soporte.
          </div>
        } @else if (escritorio.necesitaIngresar() && escritorio.pendientes() > 0) {
          <div class="aviso-warn rounded-none border-x-0 border-t-0 px-4 text-center lg:px-8" role="status">
            Para subir las ventas pendientes, salí e ingresá de nuevo con conexión.
          </div>
        }
        <main id="contenido" class="mx-auto w-full max-w-7xl flex-1 px-4 pt-6 pb-24 lg:px-8 lg:pb-10" tabindex="-1">
          <router-outlet />
        </main>
      </div>
    </div>

    <!-- Barra inferior (celular) -->
    <nav class="fixed inset-x-0 bottom-0 z-40 grid grid-cols-5 border-t border-ink-200 bg-white pb-[env(safe-area-inset-bottom)] lg:hidden dark:border-ink-800 dark:bg-ink-900"
         aria-label="Principal">
      @for (item of menuMovil; track item.ruta) {
        <a [routerLink]="item.ruta" routerLinkActive="font-semibold text-ink-900 dark:text-ink-50" ariaCurrentWhenActive="page"
           class="flex flex-col items-center gap-0.5 py-2 text-[11px] font-medium text-muted">
          <lucide-icon [img]="item.icono" [size]="22" aria-hidden="true" />
          {{ item.etiqueta }}
        </a>
      }
      <button type="button" class="flex flex-col items-center gap-0.5 py-2 text-[11px] font-medium text-muted"
              (click)="masAbierto.set(true)" [attr.aria-expanded]="masAbierto()" aria-controls="menu-mas">
        <lucide-icon [img]="iconos.mas" [size]="22" aria-hidden="true" /> Más
      </button>
    </nav>

    <!-- Menú "Más" (celular) -->
    @if (masAbierto()) {
      <div class="fixed inset-0 z-50 bg-ink-950/40 lg:hidden" (click)="masAbierto.set(false)" aria-hidden="true"></div>
      <div id="menu-mas" class="fixed inset-x-0 bottom-0 z-50 rounded-t-2xl bg-white p-4 pb-[calc(1rem+env(safe-area-inset-bottom))] shadow-md lg:hidden dark:bg-ink-900"
           role="dialog" aria-modal="true" aria-label="Más secciones">
        <div class="mb-2 flex items-center justify-between">
          <p class="font-semibold">Más secciones</p>
          <button type="button" class="btn-fantasma btn-icono" (click)="masAbierto.set(false)" aria-label="Cerrar">
            <lucide-icon [img]="iconos.cerrar" [size]="20" aria-hidden="true" />
          </button>
        </div>
        <div class="grid grid-cols-3 gap-2">
          @for (item of menu; track item.ruta) {
            <a [routerLink]="item.ruta" (click)="masAbierto.set(false)"
               class="flex flex-col items-center gap-1.5 rounded-lg p-3 text-xs font-medium text-ink-700 hover:bg-ink-100 dark:text-ink-200 dark:hover:bg-ink-800">
              <lucide-icon [img]="item.icono" [size]="22" aria-hidden="true" />
              {{ item.etiqueta }}
            </a>
          }
          <button type="button" (click)="salir()"
                  class="flex flex-col items-center gap-1.5 rounded-lg p-3 text-xs font-medium text-danger hover:bg-danger-soft">
            <lucide-icon [img]="iconos.salir" [size]="22" aria-hidden="true" /> Salir
          </button>
        </div>
      </div>
    }
  `,
})
export class ShellComponent {
  protected readonly negocio = inject(NegocioService);
  protected readonly conexion = inject(ConexionService);
  protected readonly escritorio = inject(EscritorioService);
  protected readonly tema = inject(TemaService);
  private readonly auth = inject(AuthService);
  private readonly confirmacion = inject(ConfirmacionService);

  protected readonly menu = MENU;
  protected readonly menuMovil = MENU.filter((m) => m.movil);
  protected readonly masAbierto = signal(false);
  protected readonly iconos = { salir: LogOut, luna: Moon, sol: Sun, enLinea: Wifi, sinConexion: WifiOff, mas: Ellipsis, cerrar: X, subir: CloudUpload, alerta: TriangleAlert };

  private readonly router = inject(Router);

  constructor() {
    this.router.events.pipe(filter((e) => e instanceof NavigationEnd), takeUntilDestroyed())
      .subscribe(() => this.masAbierto.set(false));
    void this.negocio.cargar(); // incluye la configuración del ticket
  }

  /**
   * Esc desde cualquier pantalla lleva a Vender. Antes cierra lo que esté abierto: un diálogo (lo cierra el
   * propio diálogo), el menú "Más" o las sugerencias del buscador (que marcan la tecla como usada).
   * En Vender, Esc lo maneja la pantalla (vuelve el cursor al buscador).
   */
  protected teclas(e: KeyboardEvent): void {
    if (e.key !== 'Escape' || e.defaultPrevented || e.ctrlKey || e.altKey || e.metaKey || document.querySelector('dialog[open]')) {
      return;
    }
    if (this.masAbierto()) {
      this.masAbierto.set(false);
      return;
    }
    if (!this.router.url.startsWith('/vender')) {
      e.preventDefault();
      void this.router.navigateByUrl('/vender');
    }
  }

  protected async salir(): Promise<void> {
    this.masAbierto.set(false);
    const ok = await this.confirmacion.preguntar({
      titulo: 'Cerrar sesión', mensaje: '¿Querés salir del sistema en este dispositivo?', confirmar: 'Salir',
    });
    if (ok) {
      this.auth.cerrarSesion();
    }
  }
}
