import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, firstValueFrom } from 'rxjs';
import { API_BASE_URL } from '../config/api.config';
import { Monto, Pagina, parametros } from './modelos';

export type TipoProducto = 'SIMPLE' | 'COMBO';

export interface ProductoResumen {
  id: string;
  sku: string;
  codigoBarras: string | null;
  /** 0 = sin foto; si no, va en la URL de la foto para la caché (US-004). */
  imagenVersion: number;
  nombre: string;
  tipo: TipoProducto;
  empresaId: string | null;
  empresa: string | null;
  precioVenta: Monto;
  costo: Monto | null;
  stock: number;
  stockMinimo: number;
  activo: boolean;
  version: number;
}

export interface ComboItem {
  productoId: string;
  sku: string;
  nombre: string;
  cantidad: number;
  precioVenta: Monto;
  stock: number;
}

export interface CategoriaRef {
  id: string;
  nombre: string;
  ruta: string;
}

export interface ProductoDetalle extends ProductoResumen {
  descripcion: string | null;
  mostrarEnTienda: boolean;
  categorias: CategoriaRef[];
  items: ComboItem[];
  createdAt: string;
  updatedAt: string;
}

export interface ConsultaProductos {
  q?: string;
  categoriaId?: string;
  empresaId?: string;
  tipo?: TipoProducto | '';
  estado?: 'activos' | 'inactivos' | 'todos';
  stockBajo?: boolean;
  orden?: 'nombre' | 'recientes' | 'precio' | 'stock';
  page?: number;
  size?: number;
}

export interface ProductoGuardar {
  tipo?: TipoProducto;
  sku: string;
  codigoBarras: string;
  nombre: string;
  descripcion: string;
  empresaId: string;
  precioVenta: Monto;
  costo: Monto | null;
  stockMinimo: number;
  mostrarEnTienda: boolean;
  categoriaIds: string[];
  items?: { productoId: string; cantidad: number }[];
  version?: number;
}

export interface CambioPrecio {
  id: string;
  precioAnterior: Monto;
  precioNuevo: Monto;
  origen: 'MANUAL' | 'CASCADA' | 'COMPRA';
  motivo: string;
  fecha: string;
}

export interface Empresa {
  id: string;
  nombre: string;
  activo: boolean;
  version: number;
}

export interface CategoriaNodo {
  id: string;
  nombre: string;
  padreId: string | null;
  version: number;
  productos: number;
  hijos: CategoriaNodo[];
}

export interface AjustePrecios {
  tipo: 'PORCENTAJE' | 'MONTO';
  valor: string;
  redondeo?: Monto | null;
  incluirSubcategorias: boolean;
  motivo?: string;
}

export interface ResultadoAjuste {
  cantidad: number;
  aplicado: boolean;
  productos: { productoId: string; sku: string; nombre: string; precioAnterior: Monto; precioNuevo: Monto }[];
}

/** Cliente del catálogo: productos, combos, categorías y empresas. */
@Injectable({ providedIn: 'root' })
export class CatalogoApi {
  private readonly http = inject(HttpClient);
  private readonly base = inject(API_BASE_URL);

  listarProductos(c: ConsultaProductos = {}): Observable<Pagina<ProductoResumen>> {
    return this.http.get<Pagina<ProductoResumen>>(`${this.base}/productos`, { params: parametros({ ...c }) });
  }

  obtenerProducto(id: string): Observable<ProductoDetalle> {
    return this.http.get<ProductoDetalle>(`${this.base}/productos/${id}`);
  }

  /** Sube la foto ya achicada (ver reducirImagen). */
  subirImagen(id: string, foto: Blob): Observable<{ imagenVersion: number }> {
    return this.http.put<{ imagenVersion: number }>(`${this.base}/productos/${id}/imagen`, foto,
      { headers: { 'Content-Type': foto.type || 'application/octet-stream' } });
  }

  quitarImagen(id: string): Observable<{ imagenVersion: number }> {
    return this.http.delete<{ imagenVersion: number }>(`${this.base}/productos/${id}/imagen`);
  }

  /** Pública: las etiquetas <img> no mandan la sesión. */
  urlImagen(id: string, version: number): string {
    return `${this.base}/imagenes/productos/${id}?v=${version}`;
  }

  /** Búsqueda exacta por SKU o código de barras (lector). */
  porCodigo(codigo: string): Observable<ProductoDetalle> {
    return this.http.get<ProductoDetalle>(`${this.base}/productos/codigo/${encodeURIComponent(codigo)}`);
  }

  crearProducto(p: ProductoGuardar): Observable<ProductoDetalle> {
    return this.http.post<ProductoDetalle>(`${this.base}/productos`, p);
  }

  actualizarProducto(id: string, p: ProductoGuardar): Observable<ProductoDetalle> {
    return this.http.put<ProductoDetalle>(`${this.base}/productos/${id}`, p);
  }

  cambiarEstado(id: string, activo: boolean): Observable<ProductoDetalle> {
    return this.http.patch<ProductoDetalle>(`${this.base}/productos/${id}/estado`, { activo });
  }

  historialPrecios(id: string, page = 1): Observable<Pagina<CambioPrecio>> {
    return this.http.get<Pagina<CambioPrecio>>(`${this.base}/productos/${id}/precios`, { params: { page } });
  }

  categorias(): Observable<CategoriaNodo[]> {
    return this.http.get<CategoriaNodo[]>(`${this.base}/categorias`);
  }

  crearCategoria(nombre: string, padreId: string | null): Observable<CategoriaNodo> {
    return this.http.post<CategoriaNodo>(`${this.base}/categorias`, { nombre, padreId });
  }

  actualizarCategoria(id: string, nombre: string, padreId: string | null, version: number): Observable<CategoriaNodo> {
    return this.http.put<CategoriaNodo>(`${this.base}/categorias/${id}`, { nombre, padreId, version });
  }

  eliminarCategoria(id: string): Observable<void> {
    return this.http.delete<void>(`${this.base}/categorias/${id}`);
  }

  ajustarPrecios(categoriaId: string, a: AjustePrecios, aplicar: boolean): Observable<ResultadoAjuste> {
    const ruta = aplicar ? 'precios' : 'precios/vista-previa';
    return this.http.post<ResultadoAjuste>(`${this.base}/categorias/${categoriaId}/${ruta}`, a);
  }

  empresas(c: { q?: string; activas?: boolean; page?: number } = {}): Observable<Pagina<Empresa>> {
    return this.http.get<Pagina<Empresa>>(`${this.base}/empresas`, { params: parametros({ ...c }) });
  }

  /** Todas las empresas (recorre las páginas; son pocas). Para selects. */
  async todasLasEmpresas(soloActivas = false): Promise<Empresa[]> {
    const out: Empresa[] = [];
    for (let page = 1; page <= 50; page++) {
      const p = await firstValueFrom(this.empresas({ activas: soloActivas, page }));
      out.push(...p.items);
      if (page >= p.totalPages) {
        break;
      }
    }
    return out;
  }

  crearEmpresa(nombre: string): Observable<Empresa> {
    return this.http.post<Empresa>(`${this.base}/empresas`, { nombre });
  }

  actualizarEmpresa(e: Empresa): Observable<Empresa> {
    return this.http.put<Empresa>(`${this.base}/empresas/${e.id}`, e);
  }
}

