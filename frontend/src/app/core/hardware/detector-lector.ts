/**
 * Distingue un lector de códigos de barras de una persona escribiendo.
 *
 * Los lectores USB/Bluetooth se comportan como un teclado: "tipean" el código muy rápido
 * (un carácter cada pocos milisegundos) y terminan con Enter. Una persona tarda 100 ms o más
 * entre teclas, así que una ráfaga de al menos LARGO_MINIMO caracteres con pausas cortas
 * seguida de Enter (o Tab) se considera una lectura.
 */
export const INTERVALO_MAXIMO_MS = 50;
export const LARGO_MINIMO = 4;
export const LARGO_MAXIMO = 64;

export class DetectorLector {
  private buffer = '';
  private ultimo = -Infinity;

  /**
   * Procesa una tecla. Devuelve el código leído cuando termina una ráfaga del lector, o null.
   * @param tecla KeyboardEvent.key
   * @param ms marca de tiempo del evento (performance.now() o event.timeStamp)
   */
  procesar(tecla: string, ms: number): string | null {
    const rapida = ms - this.ultimo <= INTERVALO_MAXIMO_MS;
    this.ultimo = ms;
    if (tecla === 'Enter' || tecla === 'Tab') {
      const codigo = this.buffer;
      this.buffer = '';
      // El Enter también tiene que llegar enseguida: si no, la persona terminó de escribir y apretó Enter.
      return rapida && codigo.length >= LARGO_MINIMO ? codigo : null;
    }
    if (tecla.length !== 1) {
      // Shift y otras teclas de control no cortan la ráfaga (el lector usa Shift para las mayúsculas).
      if (tecla !== 'Shift') {
        this.buffer = '';
      }
      return null;
    }
    this.buffer = rapida && this.buffer.length < LARGO_MAXIMO ? this.buffer + tecla : tecla;
    return null;
  }

  /** Hay una ráfaga en curso (sirve para frenar el Enter antes de que active un botón). */
  enRafaga(ms: number): boolean {
    return this.buffer.length >= LARGO_MINIMO - 1 && ms - this.ultimo <= INTERVALO_MAXIMO_MS;
  }

  reiniciar(): void {
    this.buffer = '';
    this.ultimo = -Infinity;
  }
}
