import { Injectable, effect, inject, signal } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { RouterStateSnapshot, TitleStrategy } from '@angular/router';
import { NegocioService } from './negocio/negocio.service';

/** Título de la pestaña: "Vender · <nombre del negocio>". Se actualiza también si cambia el nombre. */
@Injectable({ providedIn: 'root' })
export class TituloMagui extends TitleStrategy {
  private readonly title = inject(Title);
  private readonly negocio = inject(NegocioService);
  private readonly pantalla = signal<string | undefined>(undefined);

  constructor() {
    super();
    effect(() => {
      const t = this.pantalla();
      const nombre = this.negocio.nombre();
      this.title.setTitle(t ? `${t} · ${nombre}` : nombre);
    });
  }

  override updateTitle(estado: RouterStateSnapshot): void {
    this.pantalla.set(this.buildTitle(estado));
  }
}
