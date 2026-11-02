import { medidasReducidas } from './reducir-imagen';

describe('medidasReducidas', () => {
  it('achica el lado mayor a 800 manteniendo la proporción', () => {
    expect(medidasReducidas(4000, 3000)).toEqual({ ancho: 800, alto: 600 });
    expect(medidasReducidas(1080, 1920)).toEqual({ ancho: 450, alto: 800 });
  });

  it('no agranda fotos chicas', () => {
    expect(medidasReducidas(300, 200)).toEqual({ ancho: 300, alto: 200 });
  });

  it('nunca devuelve 0 px', () => {
    expect(medidasReducidas(10000, 1)).toEqual({ ancho: 800, alto: 1 });
  });
});
