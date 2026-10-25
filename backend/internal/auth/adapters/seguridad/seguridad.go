// Package seguridad implementa los puertos criptográficos del acceso:
// hash de contraseña (bcrypt), tokens firmados (JWT HS256) y limitador de intentos.
package seguridad

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"magui/internal/auth/application"
)

// --- Hash de contraseñas ---

// Bcrypt implementa application.Hasher.
type Bcrypt struct{ Costo int }

func (b Bcrypt) Hashear(clave string) (string, error) {
	costo := b.Costo
	if costo == 0 {
		costo = 12
	}
	h, err := bcrypt.GenerateFromPassword([]byte(clave), costo)
	return string(h), err
}

func (Bcrypt) Coincide(hash, clave string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(clave)) == nil
}

// --- Tokens ---

// LongitudMinimaSecreto en bytes para firmar HS256 con seguridad.
const LongitudMinimaSecreto = 32

const sujeto = "duena"

// JWT implementa application.Tokens.
type JWT struct {
	secreto  []byte
	duracion time.Duration
}

func NuevoJWT(secreto string, duracion time.Duration) (*JWT, error) {
	if len(secreto) < LongitudMinimaSecreto {
		return nil, fmt.Errorf("el secreto de tokens debe tener al menos %d caracteres", LongitudMinimaSecreto)
	}
	if duracion <= 0 {
		return nil, errors.New("la duración de la sesión debe ser positiva")
	}
	return &JWT{secreto: []byte(secreto), duracion: duracion}, nil
}

type claims struct {
	Version int `json:"ver"`
	jwt.RegisteredClaims
}

func (j *JWT) Emitir(version int, ahora time.Time) (application.Sesion, error) {
	exp := ahora.Add(j.duracion)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Version: version,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sujeto,
			IssuedAt:  jwt.NewNumericDate(ahora),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	})
	firmado, err := t.SignedString(j.secreto)
	if err != nil {
		return application.Sesion{}, err
	}
	return application.Sesion{Token: firmado, ExpiraEn: exp}, nil
}

func (j *JWT) Leer(token string, ahora time.Time) (int, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return j.secreto, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithSubject(sujeto),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return ahora }),
	)
	if err != nil {
		return 0, err
	}
	return c.Version, nil
}

// --- Limitador de intentos ---

// Limitador en memoria: tras MaxFallos dentro de Ventana, bloquea el origen hasta que
// pase la ventana. Implementa application.Limitador.
type Limitador struct {
	MaxFallos int
	Ventana   time.Duration

	mu      sync.Mutex
	origens map[string]*intentos
}

type intentos struct {
	fallos int
	desde  time.Time
}

func NuevoLimitador(maxFallos int, ventana time.Duration) *Limitador {
	return &Limitador{MaxFallos: maxFallos, Ventana: ventana, origens: map[string]*intentos{}}
}

func (l *Limitador) Permitido(origen string, ahora time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	i, ok := l.origens[origen]
	if !ok {
		return true
	}
	if ahora.Sub(i.desde) > l.Ventana {
		delete(l.origens, origen)
		return true
	}
	return i.fallos < l.MaxFallos
}

func (l *Limitador) RegistrarFallo(origen string, ahora time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	i, ok := l.origens[origen]
	if !ok || ahora.Sub(i.desde) > l.Ventana {
		l.origens[origen] = &intentos{fallos: 1, desde: ahora}
		return
	}
	i.fallos++
}

func (l *Limitador) Reiniciar(origen string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.origens, origen)
}
