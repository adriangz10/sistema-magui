import { CategoriaNodo } from '../../core/api/catalogo.api';

/** Categoría aplanada para listas y selects, con su nivel y ruta legible. */
export interface CategoriaPlana {
  id: string;
  nombre: string;
  ruta: string; // "PERFUMERÍA > Perfumería femenina"
  nivel: number; // 0 = principal
  padreId: string | null;
  productos: number;
  version: number;
}

/** Recorre el árbol en orden (padre antes que hijos). */
export function aplanar(nodos: readonly CategoriaNodo[], nivel = 0, rutaPadre = ''): CategoriaPlana[] {
  const out: CategoriaPlana[] = [];
  for (const n of nodos) {
    const ruta = rutaPadre ? `${rutaPadre} > ${n.nombre}` : n.nombre;
    out.push({ id: n.id, nombre: n.nombre, ruta, nivel, padreId: n.padreId, productos: n.productos, version: n.version });
    out.push(...aplanar(n.hijos, nivel + 1, ruta));
  }
  return out;
}

/** IDs de la categoría y todas sus subcategorías (para no permitir moverla dentro de sí misma). */
export function conDescendientes(nodos: readonly CategoriaNodo[], id: string): Set<string> {
  const buscar = (lista: readonly CategoriaNodo[]): CategoriaNodo | null => {
    for (const n of lista) {
      if (n.id === id) {
        return n;
      }
      const h = buscar(n.hijos);
      if (h) {
        return h;
      }
    }
    return null;
  };
  const out = new Set<string>();
  const juntar = (n: CategoriaNodo) => {
    out.add(n.id);
    n.hijos.forEach(juntar);
  };
  const raiz = buscar(nodos);
  if (raiz) {
    juntar(raiz);
  }
  return out;
}

/** Sangría visual para opciones de <select>. */
export function sangria(nivel: number): string {
  return nivel === 0 ? '' : '  '.repeat(nivel) + '└ ';
}
