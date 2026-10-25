package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"magui/internal/auth/domain"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/reloj"
)

// --- dobles de prueba ---

type repoMem struct {
	cred   domain.Credencial
	existe bool
}

func (r *repoMem) Obtener(context.Context) (domain.Credencial, bool, error) {
	return r.cred, r.existe, nil
}
func (r *repoMem) Guardar(_ context.Context, c domain.Credencial) error {
	r.cred, r.existe = c, true
	return nil
}

type hasherFalso struct{}

func (hasherFalso) Hashear(c string) (string, error) { return "hash:" + c, nil }
func (hasherFalso) Coincide(h, c string) bool        { return h == "hash:"+c }

type tokensFalsos struct{}

func (tokensFalsos) Emitir(v int, ahora time.Time) (Sesion, error) {
	return Sesion{Token: fmt.Sprintf("tok-v%d", v), ExpiraEn: ahora.Add(time.Hour)}, nil
}
func (tokensFalsos) Leer(tok string, _ time.Time) (int, error) {
	var v int
	if _, err := fmt.Sscanf(tok, "tok-v%d", &v); err != nil {
		return 0, errors.New("inválido")
	}
	return v, nil
}

type limitadorFalso struct{ fallos map[string]int }

func (l *limitadorFalso) Permitido(o string, _ time.Time) bool { return l.fallos[o] < 3 }
func (l *limitadorFalso) RegistrarFallo(o string, _ time.Time) { l.fallos[o]++ }
func (l *limitadorFalso) Reiniciar(o string)                   { delete(l.fallos, o) }

func nuevoServicio(conClave bool) (*Servicio, *repoMem) {
	repo := &repoMem{}
	if conClave {
		repo.cred, repo.existe = domain.Credencial{Hash: "hash:secreta123", Version: 1}, true
	}
	s := NuevoServicio(repo, hasherFalso{}, tokensFalsos{}, &limitadorFalso{fallos: map[string]int{}},
		reloj.Fijo{T: time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)})
	return s, repo
}

// --- tests ---

func TestIniciarSesion(t *testing.T) {
	s, _ := nuevoServicio(true)
	ses, err := s.IniciarSesion(context.Background(), "1.1.1.1", "secreta123")
	if err != nil || ses.Token != "tok-v1" {
		t.Fatalf("sesión = %+v, err = %v", ses, err)
	}
	if err := s.ValidarSesion(context.Background(), ses.Token); err != nil {
		t.Fatalf("la sesión recién emitida debería ser válida: %v", err)
	}
}

func TestIniciarSesionClaveIncorrecta(t *testing.T) {
	s, _ := nuevoServicio(true)
	if _, err := s.IniciarSesion(context.Background(), "ip", "otra"); !errors.Is(err, domain.ErrClaveIncorrecta) {
		t.Fatalf("err = %v", err)
	}
}

func TestIniciarSesionSinClaveConfigurada(t *testing.T) {
	s, _ := nuevoServicio(false)
	if _, err := s.IniciarSesion(context.Background(), "ip", "x"); !errors.Is(err, domain.ErrSinClave) {
		t.Fatalf("err = %v", err)
	}
}

func TestLimitaIntentosFallidos(t *testing.T) {
	s, _ := nuevoServicio(true)
	ctx := context.Background()
	for range 3 {
		_, _ = s.IniciarSesion(ctx, "atacante", "mal")
	}
	// Bloqueado aunque ahora acierte
	if _, err := s.IniciarSesion(ctx, "atacante", "secreta123"); !errors.Is(err, domain.ErrDemasiadosIntentos) {
		t.Fatalf("err = %v", err)
	}
	// Otro origen no se ve afectado
	if _, err := s.IniciarSesion(ctx, "duena", "secreta123"); err != nil {
		t.Fatalf("otro origen: %v", err)
	}
}

func TestCambiarClaveInvalidaSesionesAnteriores(t *testing.T) {
	s, repo := nuevoServicio(true)
	ctx := context.Background()
	vieja, _ := s.IniciarSesion(ctx, "ip", "secreta123")

	nueva, err := s.CambiarClave(ctx, "secreta123", "otraClave456")
	if err != nil {
		t.Fatal(err)
	}
	if repo.cred.Version != 2 || repo.cred.Hash != "hash:otraClave456" {
		t.Fatalf("credencial = %+v", repo.cred)
	}
	if err := s.ValidarSesion(ctx, vieja.Token); !errors.Is(err, domain.ErrSesionInvalida) {
		t.Fatalf("la sesión vieja debería quedar inválida: %v", err)
	}
	if err := s.ValidarSesion(ctx, nueva.Token); err != nil {
		t.Fatalf("la sesión nueva debería valer: %v", err)
	}
}

func TestCambiarClaveValidaciones(t *testing.T) {
	s, _ := nuevoServicio(true)
	ctx := context.Background()
	if _, err := s.CambiarClave(ctx, "mal", "otraClave456"); !errors.Is(err, domain.ErrClaveIncorrecta) {
		t.Errorf("actual incorrecta: %v", err)
	}
	_, err := s.CambiarClave(ctx, "secreta123", "corta")
	if e, ok := errores.Como(err); !ok || e.Campos["nueva"] == "" {
		t.Errorf("clave corta: %v", err)
	}
	if _, err := s.CambiarClave(ctx, "secreta123", strings.Repeat("a", 73)); err == nil {
		t.Error("clave de más de 72 bytes debería rechazarse")
	}
}

func TestEstablecerClavePrimeraVez(t *testing.T) {
	s, repo := nuevoServicio(false)
	if err := s.EstablecerClave(context.Background(), "primera123"); err != nil {
		t.Fatal(err)
	}
	if !repo.existe || repo.cred.Version != 1 {
		t.Fatalf("credencial = %+v", repo.cred)
	}
}

func TestValidarSesionTokenBasura(t *testing.T) {
	s, _ := nuevoServicio(true)
	if err := s.ValidarSesion(context.Background(), "cualquier-cosa"); !errors.Is(err, domain.ErrSesionInvalida) {
		t.Fatalf("err = %v", err)
	}
}
