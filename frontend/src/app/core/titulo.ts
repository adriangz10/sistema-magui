import { Injectable, inject } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { RouterStateSnapshot, TitleStrategy } from '@angular/router';

/** Título de la pestaña: "Vender · Magui". */
@Injectable({ providedIn: 'root' })
export class TituloMagui extends TitleStrategy {
  private readonly title = inject(Title);

  override updateTitle(estado: RouterStateSnapshot): void {
    const t = this.buildTitle(estado);
    this.title.setTitle(t ? `${t} · Magui` : 'Sistema Magui');
  }
}
