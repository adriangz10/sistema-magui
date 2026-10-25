package seguridad

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ahora = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

const secreto = "un-secreto-de-prueba-bien-largo-1234567890"

func TestBcrypt(t *testing.T) {
	b := Bcrypt{Costo: 4} // costo bajo: solo para que el test sea rápido
	h, err := b.Hashear("secreta123")
	if err != nil || strings.Contains(h, "secreta123") {
		t.Fatalf("hash = %q, err = %v", h, err)
	}
	if !b.Coincide(h, "secreta123") || b.Coincide(h, "otra") {
		t.Fatal("Coincide no funciona")
	}
}

func TestJWT(t *testing.T) {
	j, err := NuevoJWT(secreto, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ses, err := j.Emitir(7, ahora)
	if err != nil || !ses.ExpiraEn.Equal(ahora.Add(time.Hour)) {
		t.Fatalf("sesión = %+v, err = %v", ses, err)
	}
	if v, err := j.Leer(ses.Token, ahora.Add(30*time.Minute)); err != nil || v != 7 {
		t.Fatalf("leer = %d, %v", v, err)
	}
	if _, err := j.Leer(ses.Token, ahora.Add(2*time.Hour)); err == nil {
		t.Error("un token vencido debería rechazarse")
	}
	otro, _ := NuevoJWT(strings.Repeat("x", 40), time.Hour)
	if _, err := otro.Leer(ses.Token, ahora); err == nil {
		t.Error("un token firmado con otro secreto debería rechazarse")
	}
}

func TestJWTRechazaAlgoritmoNone(t *testing.T) {
	j, _ := NuevoJWT(secreto, time.Hour)
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims{Version: 1, RegisteredClaims: jwt.RegisteredClaims{
		Subject: sujeto, ExpiresAt: jwt.NewNumericDate(ahora.Add(time.Hour)),
	}})
	s, _ := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := j.Leer(s, ahora); err == nil {
		t.Fatal("un token sin firma debería rechazarse")
	}
}

func TestNuevoJWTValidaSecreto(t *testing.T) {
	if _, err := NuevoJWT("corto", time.Hour); err == nil {
		t.Error("secreto corto debería rechazarse")
	}
}

func TestLimitador(t *testing.T) {
	l := NuevoLimitador(3, 15*time.Minute)
	for range 3 {
		l.RegistrarFallo("ip", ahora)
	}
	if l.Permitido("ip", ahora.Add(time.Minute)) {
		t.Error("debería bloquear tras 3 fallos")
	}
	if !l.Permitido("otra-ip", ahora) {
		t.Error("otro origen no debería estar bloqueado")
	}
	if !l.Permitido("ip", ahora.Add(16*time.Minute)) {
		t.Error("debería liberar tras la ventana")
	}
	l.RegistrarFallo("x", ahora)
	l.Reiniciar("x")
	if !l.Permitido("x", ahora) {
		t.Error("Reiniciar debería limpiar")
	}
}
