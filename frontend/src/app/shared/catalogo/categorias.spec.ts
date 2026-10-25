import { CategoriaNodo } from '../../core/api/catalogo.api';
import { aplanar, conDescendientes } from './categorias';

const nodo = (id: string, nombre: string, hijos: CategoriaNodo[] = [], padreId: string | null = null): CategoriaNodo =>
  ({ id, nombre, padreId, version: 1, productos: 0, hijos });

const arbol = [
  nodo('perf', 'PERFUMERÍA', [nodo('fem', 'Femenina', [nodo('kits', 'Kits', [], 'fem')], 'perf')]),
  nodo('reg', 'REGALOS'),
];

describe('categorías', () => {
  it('aplana en orden con nivel y ruta', () => {
    const plana = aplanar(arbol);
    expect(plana.map((c) => [c.id, c.nivel, c.ruta])).toEqual([
      ['perf', 0, 'PERFUMERÍA'],
      ['fem', 1, 'PERFUMERÍA > Femenina'],
      ['kits', 2, 'PERFUMERÍA > Femenina > Kits'],
      ['reg', 0, 'REGALOS'],
    ]);
  });

  it('junta la categoría con sus descendientes', () => {
    expect([...conDescendientes(arbol, 'perf')].sort()).toEqual(['fem', 'kits', 'perf']);
    expect([...conDescendientes(arbol, 'reg')]).toEqual(['reg']);
    expect(conDescendientes(arbol, 'nope').size).toBe(0);
  });
});
