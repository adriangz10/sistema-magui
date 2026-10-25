// Package application contiene los casos de uso del acceso de la dueña.
package application

import (
	"context"

	"magui/internal/auth/domain"
	"magui/internal/kernel/reloj"
)

// Servicio implementa iniciar sesión, validar sesión y cambiar la contraseña.
type Servicio struct {
	repo      RepositorioCredencial
	hasher    Hasher
	tokens    Tokens
	limitador Limitador
	reloj     reloj.Reloj
}

func NuevoServicio(repo RepositorioCredencial, h Hasher, t Tokens, l Limitador, r reloj.Reloj) *Servicio {
	return &Servicio{repo: repo, hasher: h, tokens: t, limitador: l, reloj: r}
}

// IniciarSesion verifica la contraseña y emite una sesión. origen identifica al cliente
// (IP) para limitar intentos fallidos.
func (s *Servicio) IniciarSesion(ctx context.Context, origen, clave string) (Sesion, error) {
	ahora := s.reloj.Ahora()
	if !s.limitador.Permitido(origen, ahora) {
		return Sesion{}, domain.ErrDemasiadosIntentos
	}
	cred, existe, err := s.repo.Obtener(ctx)
	if err != nil {
		return Sesion{}, err
	}
	if !existe {
		return Sesion{}, domain.ErrSinClave
	}
	if !s.hasher.Coincide(cred.Hash, clave) {
		s.limitador.RegistrarFallo(origen, ahora)
		return Sesion{}, domain.ErrClaveIncorrecta
	}
	s.limitador.Reiniciar(origen)
	return s.tokens.Emitir(cred.Version, ahora)
}

// ValidarSesion confirma que el token es válido y que no se cambió la contraseña después.
func (s *Servicio) ValidarSesion(ctx context.Context, token string) error {
	version, err := s.tokens.Leer(token, s.reloj.Ahora())
	if err != nil {
		return domain.ErrSesionInvalida
	}
	cred, existe, err := s.repo.Obtener(ctx)
	if err != nil {
		return err
	}
	if !existe || cred.Version != version {
		return domain.ErrSesionInvalida
	}
	return nil
}

// CambiarClave cambia la contraseña. Todas las sesiones anteriores dejan de valer;
// devuelve una sesión nueva para el dispositivo que hizo el cambio.
func (s *Servicio) CambiarClave(ctx context.Context, actual, nueva string) (Sesion, error) {
	if err := domain.ValidarClaveNueva("nueva", nueva); err != nil {
		return Sesion{}, err
	}
	cred, existe, err := s.repo.Obtener(ctx)
	if err != nil {
		return Sesion{}, err
	}
	if !existe {
		return Sesion{}, domain.ErrSinClave
	}
	if !s.hasher.Coincide(cred.Hash, actual) {
		return Sesion{}, domain.ErrClaveIncorrecta
	}
	nuevaCred, err := s.guardarNueva(ctx, cred, nueva)
	if err != nil {
		return Sesion{}, err
	}
	return s.tokens.Emitir(nuevaCred.Version, s.reloj.Ahora())
}

// EstablecerClave fija la contraseña sin pedir la anterior. Solo lo usa el comando admin
// (requiere acceso al servidor), para la primera configuración o si se olvidó.
func (s *Servicio) EstablecerClave(ctx context.Context, nueva string) error {
	if err := domain.ValidarClaveNueva("clave", nueva); err != nil {
		return err
	}
	cred, _, err := s.repo.Obtener(ctx) // si no existe, parte de la versión 0
	if err != nil {
		return err
	}
	_, err = s.guardarNueva(ctx, cred, nueva)
	return err
}

func (s *Servicio) guardarNueva(ctx context.Context, cred domain.Credencial, clave string) (domain.Credencial, error) {
	hash, err := s.hasher.Hashear(clave)
	if err != nil {
		return domain.Credencial{}, err
	}
	nueva := cred.SiguienteVersion(hash, s.reloj.Ahora())
	if err := s.repo.Guardar(ctx, nueva); err != nil {
		return domain.Credencial{}, err
	}
	return nueva, nil
}
