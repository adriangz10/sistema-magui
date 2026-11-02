import { DetectorLector } from './detector-lector';

/** Simula teclas con un intervalo fijo; devuelve lo último que detectó. */
function teclear(d: DetectorLector, teclas: string[], intervalo: number, desde = 1000): (string | null)[] {
  return teclas.map((t, i) => d.procesar(t, desde + i * intervalo));
}

describe('DetectorLector', () => {
  it('reconoce la ráfaga rápida de un lector terminada en Enter', () => {
    const d = new DetectorLector();
    const r = teclear(d, [...'7791234567890', 'Enter'], 8);
    expect(r.at(-1)).toBe('7791234567890');
    expect(r.slice(0, -1).every((x) => x === null)).toBe(true);
  });

  it('acepta mayúsculas con Shift en el medio y Tab como final', () => {
    const d = new DetectorLector();
    expect(teclear(d, ['Shift', 'A', 'B', 'Shift', 'C', '1', '2', 'Tab'], 5).at(-1)).toBe('ABC12');
  });

  it('no confunde a una persona escribiendo', () => {
    const d = new DetectorLector();
    expect(teclear(d, [...'hola1234', 'Enter'], 120).at(-1)).toBeNull();
  });

  it('ignora códigos muy cortos y un Enter que llega tarde', () => {
    const d = new DetectorLector();
    expect(teclear(d, ['1', '2', 'Enter'], 5).at(-1)).toBeNull();
    teclear(d, [...'123456'], 5, 5000);
    expect(d.procesar('Enter', 5000 + 6 * 5 + 500)).toBeNull();
  });

  it('una pausa larga empieza una lectura nueva', () => {
    const d = new DetectorLector();
    teclear(d, [...'999'], 5, 0);
    expect(teclear(d, [...'12345', 'Enter'], 5, 2000).at(-1)).toBe('12345');
  });

  it('avisa que hay una ráfaga en curso', () => {
    const d = new DetectorLector();
    teclear(d, [...'1234'], 5, 0);
    expect(d.enRafaga(20)).toBe(true);
    expect(d.enRafaga(500)).toBe(false);
  });
});
