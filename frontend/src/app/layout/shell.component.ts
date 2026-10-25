import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { filter } from 'rxjs';
import { Ellipsis, LogOut, LucideAngularModule, Moon, Sun, Wifi, WifiOff, X } from 'lucide-angular';
import { AuthService } from '../core/auth/auth.service';
import { ConexionService } from '../core/red/conexion.service';
import { ConfirmacionService } from '../core/ui/confirmacion.service';
import { TemaService } from '../core/ui/tema.service';
import { MENU } from './navegacion';

/** Estructura de la app con sesión: menú lateral (computadora) y barra inferior (celular). */
@Component({
  selector: 'app-shell',
  imports: [RouterOutlet, RouterLink, RouterLinkActive, LucideAngularModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a href="#contenido" class="sr-only z-50 rounded-lg bg-brand-600 px-4 py-2 text-white focus:not-sr-only focus:fixed focus:top-2 focus:left-2">
      Saltar al contenido
    </a>

    <div class="flex min-h-full">
      <!-- Menú lateral (pantallas grandes) -->
      <aside class="sticky top-0 hidden h-screen w-64 shrink-0 flex-col border-r border-ink-200 bg-white lg:flex dark:border-ink-800 dark:bg-ink-900">
        <div class="flex items-center gap-3 px-5 py-5">
          <div class="flex size-9 items-center justify-center rounded-xl bg-brand-600 font-bold text-white shadow">M</div>
          <div>
            <p class="font-semibold leading-tight">Magui</p>
            <p class="text-xs text-ink-500">Cosmética</p>
          </div>
        </div>
        <nav class="flex-1 space-y-1 overflow-y-auto px-3" aria-label="Principal">
          @for (item of menu; track item.ruta) {
            <a [routerLink]="item.ruta" routerLinkActive="bg-brand-50 text-brand-700 dark:bg-brand-950 dark:text-brand-200"
               ariaCurrentWhenActive="page"
               class="flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium text-ink-600 transition hover:bg-ink-100 dark:text-ink-300 dark:hover:bg-ink-800">
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
        <header class="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-ink-200 bg-white/85 px-4 backdrop-blur lg:px-8 dark:border-ink-800 dark:bg-ink-900/85">
          <div class="flex items-center gap-2 lg:hidden">
            <div class="flex size-8 items-center justify-center rounded-lg bg-brand-600 text-sm font-bold text-white">M</div>
            <span class="font-semibold">Magui</span>
          </div>
          <div class="flex-1"></div>
          <span class="flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium"
                [class]="conexion.enLinea() ? 'bg-green-50 text-ok dark:bg-green-950 dark:text-green-300' : 'bg-amber-50 text-warn dark:bg-amber-950 dark:text-amber-300'"
                role="status">
            <lucide-icon [img]="conexion.enLinea() ? iconos.enLinea : iconos.sinConexion" [size]="14" aria-hidden="true" />
            {{ conexion.enLinea() ? 'En línea' : 'Sin conexión' }}
          </span>
          <button type="button" class="btn-fantasma btn-icono" (click)="tema.alternar()"
                  [attr.aria-label]="tema.tema() === 'oscuro' ? 'Cambiar a modo claro' : 'Cambiar a modo oscuro'">
            <lucide-icon [img]="tema.tema() === 'oscuro' ? iconos.sol : iconos.luna" [size]="18" aria-hidden="true" />
          </button>
        </header>

        <main id="contenido" class="mx-auto w-full max-w-7xl flex-1 px-4 pt-6 pb-24 lg:px-8 lg:pb-10" tabindex="-1">
          <router-outlet />
        </main>
      </div>
    </div>

    <!-- Barra inferior (celular) -->
    <nav class="fixed inset-x-0 bottom-0 z-40 grid grid-cols-5 border-t border-ink-200 bg-white pb-[env(safe-area-inset-bottom)] lg:hidden dark:border-ink-800 dark:bg-ink-900"
         aria-label="Principal">
      @for (item of menuMovil; track item.ruta) {
        <a [routerLink]="item.ruta" routerLinkActive="text-brand-600 dark:text-brand-300" ariaCurrentWhenActive="page"
           class="flex flex-col items-center gap-0.5 py-2 text-[11px] font-medium text-ink-500">
          <lucide-icon [img]="item.icono" [size]="22" aria-hidden="true" />
          {{ item.etiqueta }}
        </a>
      }
      <button type="button" class="flex flex-col items-center gap-0.5 py-2 text-[11px] font-medium text-ink-500"
              (click)="masAbierto.set(true)" [attr.aria-expanded]="masAbierto()" aria-controls="menu-mas">
        <lucide-icon [img]="iconos.mas" [size]="22" aria-hidden="true" /> Más
      </button>
    </nav>

    <!-- Menú "Más" (celular) -->
    @if (masAbierto()) {
      <div class="fixed inset-0 z-50 bg-ink-950/50 lg:hidden" (click)="masAbierto.set(false)" aria-hidden="true"></div>
      <div id="menu-mas" class="fixed inset-x-0 bottom-0 z-50 rounded-t-3xl bg-white p-4 pb-[calc(1rem+env(safe-area-inset-bottom))] shadow-2xl lg:hidden dark:bg-ink-900"
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
               class="flex flex-col items-center gap-1.5 rounded-2xl p-3 text-xs font-medium text-ink-700 hover:bg-ink-100 dark:text-ink-200 dark:hover:bg-ink-800">
              <lucide-icon [img]="item.icono" [size]="22" aria-hidden="true" />
              {{ item.etiqueta }}
            </a>
          }
          <button type="button" (click)="salir()"
                  class="flex flex-col items-center gap-1.5 rounded-2xl p-3 text-xs font-medium text-danger hover:bg-red-50 dark:hover:bg-red-950">
            <lucide-icon [img]="iconos.salir" [size]="22" aria-hidden="true" /> Salir
          </button>
        </div>
      </div>
    }
  `,
})
export class ShellComponent {
  protected readonly conexion = inject(ConexionService);
  protected readonly tema = inject(TemaService);
  private readonly auth = inject(AuthService);
  private readonly confirmacion = inject(ConfirmacionService);

  protected readonly menu = MENU;
  protected readonly menuMovil = MENU.filter((m) => m.movil);
  protected readonly masAbierto = signal(false);
  protected readonly iconos = { salir: LogOut, luna: Moon, sol: Sun, enLinea: Wifi, sinConexion: WifiOff, mas: Ellipsis, cerrar: X };

  constructor() {
    inject(Router).events.pipe(filter((e) => e instanceof NavigationEnd), takeUntilDestroyed())
      .subscribe(() => this.masAbierto.set(false));
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
