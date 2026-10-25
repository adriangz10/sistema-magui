import { HttpErrorResponse } from '@angular/common/http';

/** Error de la API ya interpretado (formato de docs/API.md). */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly codigo: string,
    mensaje: string,
    readonly campos: Readonly<Record<string, string>> = {},
  ) {
    super(mensaje);
    this.name = 'ApiError';
  }

  /** Mensaje del primer campo con error, o el general. */
  get detalle(): string {
    const primero = Object.values(this.campos)[0];
    return primero ? `${this.message}: ${primero}` : this.message;
  }

  static desde(err: unknown): ApiError {
    if (err instanceof ApiError) {
      return err;
    }
    if (err instanceof HttpErrorResponse) {
      if (err.status === 0 || err.status === 502 || err.status === 503 || err.status === 504) {
        return new ApiError(0, 'sin_conexion', 'No se pudo conectar con el servidor. Revisá la conexión.');
      }
      const cuerpo = err.error as { error?: { code?: string; message?: string; fields?: Record<string, string> } } | null;
      const e = cuerpo?.error;
      if (e?.code) {
        return new ApiError(err.status, e.code, e.message ?? 'Ocurrió un error', e.fields ?? {});
      }
      return new ApiError(err.status, 'error_http', 'Ocurrió un error inesperado. Intentá de nuevo.');
    }
    return new ApiError(0, 'desconocido', 'Ocurrió un error inesperado. Intentá de nuevo.');
  }
}
