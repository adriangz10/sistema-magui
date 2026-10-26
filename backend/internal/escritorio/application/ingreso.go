package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"magui/internal/kernel/errores"
	"magui/internal/kernel/reloj"
)

// DuracionSesionLocal: una sesión abierta sin conexión dura poco; al volver internet se ingresa de nuevo.
const DuracionSesionLocal = 12 * time.Hour

// PrefijoTokenLocal identifica las sesiones abiertas sin conexión (el servidor no las conoce).
const PrefijoTokenLocal = "local."

// Ingreso recuerda la sesión del servidor y permite ingresar sin conexión.
type Ingreso struct {
	sesiones Sesiones
	claves   Claves
	reloj    reloj.Reloj
}

func NuevoIngreso(s Sesiones, c Claves, r reloj.Reloj) *Ingreso {
	return &Ingreso{sesiones: s, claves: c, reloj: r}
}

// Recordar guarda la sesión que devolvió el servidor y la contraseña (cifrada) con la que se ingresó.
func (i *Ingreso) Recordar(ctx context.Context, token string, expira time.Time, clave string) error {
	s := Sesion{Token: token, ExpiraEn: expira}
	if clave != "" {
		h, err := i.claves.Cifrar(clave)
		if err != nil {
			return err
		}
		s.HashClave = h
	} else if previa, err := i.sesiones.Leer(ctx); err == nil {
		s.HashClave = previa.HashClave
	}
	return i.sesiones.Guardar(ctx, s)
}

// ActualizarToken guarda un token que el servidor acaba de aceptar (si es distinto del guardado).
func (i *Ingreso) ActualizarToken(ctx context.Context, token string) error {
	s, err := i.sesiones.Leer(ctx)
	if err != nil || s.Token == token {
		return err
	}
	// No se conoce su vencimiento: se toma el mismo plazo que la sesión anterior.
	s.Token = token
	return i.sesiones.Guardar(ctx, s)
}

// SesionAbierta es lo que recibe la pantalla al ingresar (igual que la respuesta del servidor).
type SesionAbierta struct {
	Token    string    `json:"token"`
	ExpiraEn time.Time `json:"expiraEn"`
}

// IngresarSinConexion acepta la última contraseña usada en esta PC. Si la sesión del servidor sigue
// vigente la reutiliza (así, al volver internet, todo sigue funcionando); si no, abre una local.
func (i *Ingreso) IngresarSinConexion(ctx context.Context, clave string) (SesionAbierta, error) {
	s, err := i.sesiones.Leer(ctx)
	if err != nil {
		return SesionAbierta{}, err
	}
	if s.HashClave == "" {
		return SesionAbierta{}, errores.Conflicto("sin_conexion",
			"No hay conexión con el servidor y esta PC todavía no tiene una contraseña guardada. Conectate para ingresar por primera vez.")
	}
	if !i.claves.Coincide(s.HashClave, clave) {
		return SesionAbierta{}, errores.NoAutorizado("clave_incorrecta", "La contraseña es incorrecta")
	}
	ahora := i.reloj.Ahora()
	if s.Token != "" && s.ExpiraEn.After(ahora.Add(DuracionSesionLocal)) {
		return SesionAbierta{Token: s.Token, ExpiraEn: s.ExpiraEn}, nil
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return SesionAbierta{Token: PrefijoTokenLocal + hex.EncodeToString(b), ExpiraEn: ahora.Add(DuracionSesionLocal)}, nil
}
