package application

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"magui/internal/kernel/errores"
	"magui/internal/kernel/reloj"
)

// Catalogo es el puerto hacia el catálogo.
type Catalogo interface {
	Copia(ctx context.Context) ([]ProductoCopia, []ComponenteCopia, error)
}

// Ventas es el puerto hacia ventas.
type Ventas interface {
	// RegistrarOffline guarda la venta; yaExistia es true si ya se había recibido.
	// Los errores tipados (errores.Error) son rechazos; cualquier otro es una falla del servidor.
	RegistrarOffline(ctx context.Context, v VentaOffline) (yaExistia bool, err error)
}

// Servicio atiende la sincronización del lado del servidor.
type Servicio struct {
	catalogo Catalogo
	ventas   Ventas
	reloj    reloj.Reloj
}

func NuevoServicio(c Catalogo, v Ventas, r reloj.Reloj) *Servicio {
	return &Servicio{catalogo: c, ventas: v, reloj: r}
}

// CopiaCatalogo devuelve el catálogo completo para el escritorio.
func (s *Servicio) CopiaCatalogo(ctx context.Context) (CopiaCatalogo, error) {
	ahora := s.reloj.Ahora()
	ps, cs, err := s.catalogo.Copia(ctx)
	if err != nil {
		return CopiaCatalogo{}, err
	}
	if ps == nil {
		ps = []ProductoCopia{}
	}
	if cs == nil {
		cs = []ComponenteCopia{}
	}
	return CopiaCatalogo{GeneradaEn: ahora, Productos: ps, Componentes: cs}, nil
}

// codigoConcurrente es el código de platform/db.ErrOperacionConcurrente (deadlock o espera de
// bloqueo agotada). Se repite acá para no acoplar sync a la capa de plataforma.
const codigoConcurrente = "operacion_concurrente"

// RecibirVentas registra cada venta por separado: una venta rechazada no frena a las demás.
// Si el servidor falla (ej. se cae la base) devuelve error y el escritorio reintenta todo
// más tarde; como el registro es idempotente, las que ya entraron no se duplican.
func (s *Servicio) RecibirVentas(ctx context.Context, ventas []VentaOffline) (RespuestaEnvio, error) {
	if len(ventas) > MaxVentasPorEnvio {
		return RespuestaEnvio{}, errores.Invalido("demasiadas_ventas",
			fmt.Sprintf("Se pueden enviar hasta %d ventas por vez", MaxVentasPorEnvio))
	}
	out := RespuestaEnvio{Resultados: make([]ResultadoEnvio, 0, len(ventas))}
	for _, v := range ventas {
		r := ResultadoEnvio{ID: v.ID, Estado: Registrada}
		yaExistia, err := s.ventas.RegistrarOffline(ctx, v)
		switch e, tipado := errores.Como(err); {
		case err == nil && yaExistia:
			r.Estado = YaExistia
		case err == nil:
		case tipado && e.Codigo == codigoConcurrente:
			// Transitorio (choque de bloqueos con otra operación, ya reintentado): no es un rechazo.
			// Se corta el envío y el escritorio reintenta toda la tanda; el registro es idempotente.
			return RespuestaEnvio{}, err
		case tipado:
			r.Estado, r.Error = Rechazada, &ErrorEnvio{Code: e.Codigo, Message: detalle(e)}
		default:
			return RespuestaEnvio{}, err
		}
		out.Resultados = append(out.Resultados, r)
	}
	return out, nil
}

// detalle arma un mensaje legible con los campos inválidos ("Revisá los datos: id: Debe ser un UUID").
func detalle(e *errores.Error) string {
	if len(e.Campos) == 0 {
		return e.Mensaje
	}
	partes := make([]string, 0, len(e.Campos))
	for campo, msg := range e.Campos {
		partes = append(partes, campo+": "+msg)
	}
	sort.Strings(partes)
	return e.Mensaje + ": " + strings.Join(partes, "; ")
}
