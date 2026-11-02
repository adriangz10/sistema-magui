import { DOCUMENT } from '@angular/common';
import { Injectable, inject, signal } from '@angular/core';
import { Subject } from 'rxjs';
import { almacenamiento } from '../almacenamiento';
import { DetectorLector } from './detector-lector';

const CLAVE_DETECTADO = 'magui.lector-detectado';

export interface Lectura {
  codigo: string;
  /** Hora de la lectura (ISO). */
  en: string;
}

/** Hay un campo de texto con el foco: ahí el código lo recibe el propio campo. */
function esEditable(el: EventTarget | null): boolean {
  if (!(el instanceof HTMLElement)) {
    return false;
  }
  const tag = el.tagName;
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || el.isContentEditable;
}

/**
 * Detecta un lector de códigos de barras en toda la app (US-003): aunque el foco no esté en el buscador
 * (por ejemplo, después de tocar un botón), una lectura se reconoce y se avisa por `lecturas`.
 * También frena el Enter del lector para que no "apriete" el botón que tiene el foco.
 */
@Injectable({ providedIn: 'root' })
export class LectorCodigosService {
  private readonly documento = inject(DOCUMENT);
  private readonly detector = new DetectorLector();
  private readonly detectorCampo = new DetectorLector();
  private readonly avisos = new Subject<Lectura>();
  private iniciado = false;

  /** Lecturas hechas fuera de un campo de texto. */
  readonly lecturas = this.avisos.asObservable();
  /** Última lectura (para probar el lector en Configuración). */
  readonly ultima = signal<Lectura | null>(null);
  /** Alguna vez se detectó un lector en este dispositivo. */
  readonly detectado = signal(almacenamiento.leer(CLAVE_DETECTADO) === '1');

  iniciar(): void {
    if (this.iniciado) {
      return;
    }
    this.iniciado = true;
    // Fase de captura: se ve la tecla antes que cualquier botón o atajo de la pantalla.
    this.documento.addEventListener('keydown', (e) => this.alTeclear(e), true);
  }

  private alTeclear(e: KeyboardEvent): void {
    if (e.ctrlKey || e.altKey || e.metaKey || e.isComposing) {
      this.detector.reiniciar();
      return;
    }
    if (esEditable(e.target)) {
      this.detector.reiniciar();
      return;
    }
    const ms = e.timeStamp || performance.now();
    const final = e.key === 'Enter' || e.key === 'Tab';
    const rafaga = final && this.detector.enRafaga(ms);
    const codigo = this.detector.procesar(e.key, ms);
    if (rafaga) {
      e.preventDefault();
      e.stopPropagation();
    }
    if (codigo) {
      this.avisos.next(this.marcar(codigo));
    }
  }

  /**
   * Para campos de búsqueda: el campo ya recibe el código, solo se anota que hubo una lectura.
   * Llamar desde su keydown.
   */
  observarCampo(e: KeyboardEvent): void {
    const codigo = this.detectorCampo.procesar(e.key, e.timeStamp || performance.now());
    if (codigo) {
      this.marcar(codigo);
    }
  }

  private marcar(codigo: string): Lectura {
    const lectura = { codigo, en: new Date().toISOString() };
    this.ultima.set(lectura);
    if (!this.detectado()) {
      this.detectado.set(true);
      almacenamiento.guardar(CLAVE_DETECTADO, '1');
    }
    return lectura;
  }
}
