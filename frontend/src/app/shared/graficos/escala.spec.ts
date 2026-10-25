import { compacto, marcasEje } from './escala';

describe('escala de gráficos', () => {
  it('genera marcas redondas que cubren el máximo', () => {
    expect(marcasEje(98900)).toEqual([0, 25000, 50000, 75000, 100000]);
    expect(marcasEje(3)).toEqual([0, 1, 2, 3]);
    expect(marcasEje(0)).toEqual([0, 1]);
    const m = marcasEje(1234567);
    expect(m[0]).toBe(0);
    expect(m[m.length - 1]).toBeGreaterThanOrEqual(1234567);
  });

  it('formatea compacto en castellano', () => {
    expect(compacto(1500, '$')).toBe('$1,5 mil');
    expect(compacto(2_300_000, '$')).toBe('$2,3 M');
    expect(compacto(800)).toBe('800');
  });
});
