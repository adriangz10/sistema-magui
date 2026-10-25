import { DOCUMENT } from '@angular/common';
import { Injectable, effect, inject, signal } from '@angular/core';
import { almacenamiento } from '../almacenamiento';

export type Tema = 'claro' | 'oscuro';

const CLAVE = 'magui.tema';

/** Modo claro/oscuro: respeta la preferencia del sistema hasta que la usuaria elige uno. */
@Injectable({ providedIn: 'root' })
export class TemaService {
  private readonly documento = inject(DOCUMENT);
  readonly tema = signal<Tema>(temaInicial());

  constructor() {
    effect(() => {
      const oscuro = this.tema() === 'oscuro';
      this.documento.documentElement.classList.toggle('dark', oscuro);
      this.documento.documentElement.style.colorScheme = oscuro ? 'dark' : 'light';
    });
  }

  alternar(): void {
    const nuevo: Tema = this.tema() === 'oscuro' ? 'claro' : 'oscuro';
    this.tema.set(nuevo);
    almacenamiento.guardar(CLAVE, nuevo);
  }
}

function temaInicial(): Tema {
  const guardado = almacenamiento.leer(CLAVE);
  if (guardado === 'claro' || guardado === 'oscuro') {
    return guardado;
  }
  return globalThis.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'oscuro' : 'claro';
}
