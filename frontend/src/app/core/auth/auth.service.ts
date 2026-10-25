import { HttpClient } from '@angular/common/http';
import { Injectable, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { ApiError } from '../api/api-error';
import { almacenamiento } from '../almacenamiento';
import { API_BASE_URL } from '../config/api.config';

interface Sesion {
  token: string;
  expiraEn: string;
}

const CLAVE = 'magui.sesion';

/** Sesión de la dueña: guarda el token y avisa si está vencido. */
@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly baseUrl = inject(API_BASE_URL);

  private readonly sesion = signal<Sesion | null>(leerSesion());

  readonly token = computed(() => this.sesion()?.token ?? null);
  readonly autenticada = computed(() => {
    const s = this.sesion();
    return s !== null && new Date(s.expiraEn).getTime() > Date.now();
  });

  async iniciarSesion(clave: string): Promise<void> {
    try {
      const s = await firstValueFrom(this.http.post<Sesion>(`${this.baseUrl}/auth/login`, { clave }));
      this.guardar(s);
    } catch (e) {
      throw ApiError.desde(e);
    }
  }

  async cambiarClave(actual: string, nueva: string): Promise<void> {
    try {
      const s = await firstValueFrom(this.http.post<Sesion>(`${this.baseUrl}/auth/cambiar-clave`, { actual, nueva }));
      this.guardar(s);
    } catch (e) {
      throw ApiError.desde(e);
    }
  }

  /** Cierra la sesión y vuelve al ingreso (recordando adónde quería ir). */
  cerrarSesion(volverA?: string): void {
    this.sesion.set(null);
    almacenamiento.borrar(CLAVE);
    void this.router.navigate(['/ingresar'], { queryParams: volverA ? { volver: volverA } : {} });
  }

  private guardar(s: Sesion): void {
    this.sesion.set(s);
    almacenamiento.guardar(CLAVE, JSON.stringify(s));
  }
}

function leerSesion(): Sesion | null {
  const crudo = almacenamiento.leer(CLAVE);
  if (!crudo) {
    return null;
  }
  try {
    const s = JSON.parse(crudo) as Partial<Sesion>;
    return typeof s.token === 'string' && typeof s.expiraEn === 'string' ? { token: s.token, expiraEn: s.expiraEn } : null;
  } catch {
    return null;
  }
}
