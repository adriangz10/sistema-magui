/** Lado mayor de la foto de un producto: alcanza para verla nítida en pantalla. */
export const LADO_MAXIMO = 800;
/** El servidor acepta hasta 1 MB; se apunta bastante por debajo. */
const PESO_OBJETIVO = 900 * 1024;

/** Medidas para que el lado mayor no pase de `lado`, sin agrandar fotos chicas. */
export function medidasReducidas(ancho: number, alto: number, lado = LADO_MAXIMO): { ancho: number; alto: number } {
  const escala = Math.min(1, lado / Math.max(ancho, alto));
  return { ancho: Math.max(1, Math.round(ancho * escala)), alto: Math.max(1, Math.round(alto * escala)) };
}

export interface ImagenReducida {
  blob: Blob;
  /** data: URL para la vista previa (la política de seguridad no permite blob:). */
  vistaPrevia: string;
}

function aBlob(canvas: HTMLCanvasElement, tipo: string, calidad: number): Promise<Blob | null> {
  return new Promise((ok) => canvas.toBlob(ok, tipo, calidad));
}

/**
 * Achica una foto (por ejemplo, de 4000 px sacada con el celular) antes de subirla: WEBP o JPG de hasta 800 px,
 * con fondo blanco (las transparencias quedan blancas, como en un catálogo).
 */
export async function reducirImagen(archivo: Blob): Promise<ImagenReducida> {
  let bitmap: ImageBitmap;
  try {
    bitmap = await createImageBitmap(archivo, { imageOrientation: 'from-image' });
  } catch {
    throw new Error('No se pudo leer la imagen. Probá con una foto JPG o PNG.');
  }
  const { ancho, alto } = medidasReducidas(bitmap.width, bitmap.height);
  const canvas = document.createElement('canvas');
  canvas.width = ancho;
  canvas.height = alto;
  const ctx = canvas.getContext('2d');
  if (!ctx) {
    throw new Error('No se pudo procesar la imagen en este dispositivo.');
  }
  ctx.fillStyle = '#ffffff';
  ctx.fillRect(0, 0, ancho, alto);
  ctx.drawImage(bitmap, 0, 0, ancho, alto);
  bitmap.close();

  // WEBP pesa menos; si el navegador no lo genera, JPG.
  let tipo = 'image/webp';
  let calidad = 0.85;
  let blob = await aBlob(canvas, tipo, calidad);
  if (!blob || blob.type !== tipo) {
    tipo = 'image/jpeg';
    blob = await aBlob(canvas, tipo, calidad);
  }
  while (blob && blob.size > PESO_OBJETIVO && calidad > 0.4) {
    calidad -= 0.15;
    blob = await aBlob(canvas, tipo, calidad);
  }
  if (!blob) {
    throw new Error('No se pudo procesar la imagen en este dispositivo.');
  }
  return { blob, vistaPrevia: canvas.toDataURL(tipo, calidad) };
}
