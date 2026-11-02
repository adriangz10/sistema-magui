package application

import (
	"context"
	"strings"
	"time"

	"magui/internal/caja/domain"
	"magui/internal/kernel/dinero"
	"magui/internal/kernel/id"
	"magui/internal/kernel/paginacion"
	"magui/internal/kernel/reloj"
	"magui/internal/kernel/tx"
)

// RepositorioSesiones persiste las cajas (apertura/cierre, US-002).
type RepositorioSesiones interface {
	// Crear guarda una caja nueva. Si ya hay una abierta devuelve domain.ErrCajaYaAbierta.
	Crear(ctx context.Context, s domain.Sesion) error
	Obtener(ctx context.Context, id string) (domain.Sesion, error)
	// Abierta devuelve la caja abierta o nil si no hay.
	Abierta(ctx context.Context) (*domain.Sesion, error)
	// Actualizar guarda cabecera y arqueo solo si la versión guardada sigue siendo versionAnterior;
	// si no, devuelve domain.ErrVersionDesactualizada.
	Actualizar(ctx context.Context, s domain.Sesion, versionAnterior int) error
	Listar(ctx context.Context, p paginacion.Solicitud) ([]domain.Sesion, int, error)
	GuardarEdicion(ctx context.Context, e EdicionCaja) error
	Ediciones(ctx context.Context, sesionID string) ([]EdicionCaja, error)
	// Totales suma ventas confirmadas y gastos no anulados por medio en [desde, hasta)
	// y cuenta las ventas confirmadas.
	Totales(ctx context.Context, desde, hasta time.Time) (domain.Totales, int, error)
}

// EdicionCaja es una entrada del historial de ediciones de una caja.
type EdicionCaja struct {
	ID       string          `json:"id"`
	SesionID string          `json:"-"`
	Fecha    time.Time       `json:"fecha"`
	Motivo   string          `json:"motivo"`
	Version  int             `json:"version"`
	Cambios  []domain.Cambio `json:"cambios"`
}

type ArqueoDTO struct {
	Metodo     domain.Metodo  `json:"metodo"`
	Ventas     dinero.Dinero  `json:"ventas"`
	Gastos     dinero.Dinero  `json:"gastos"`
	Esperado   dinero.Dinero  `json:"esperado"`
	Contado    *dinero.Dinero `json:"contado"`
	Diferencia *dinero.Dinero `json:"diferencia"`
}

// SesionDTO es una caja con su arqueo. Si está abierta, el arqueo es en vivo y sin contado.
type SesionDTO struct {
	ID             string         `json:"id"`
	AbiertaEn      time.Time      `json:"abiertaEn"`
	CerradaEn      *time.Time     `json:"cerradaEn"`
	Abierta        bool           `json:"abierta"`
	MontoInicial   dinero.Dinero  `json:"montoInicial"`
	Notas          *string        `json:"notas"`
	Version        int            `json:"version"`
	CantidadVentas int            `json:"cantidadVentas"`
	Arqueo         []ArqueoDTO    `json:"arqueo"`
	TotalEsperado  dinero.Dinero  `json:"totalEsperado"`
	TotalContado   *dinero.Dinero `json:"totalContado"`
	Diferencia     *dinero.Dinero `json:"diferencia"` // suma de las diferencias de todos los medios
	Editada        bool           `json:"editada"`
}

// SesionDetalle agrega el historial de ediciones.
type SesionDetalle struct {
	SesionDTO
	Ediciones []EdicionCaja `json:"ediciones"`
}

func dtoSesion(s domain.Sesion, arqueo []domain.Arqueo, cantidad int) SesionDTO {
	d := SesionDTO{ID: s.ID, AbiertaEn: s.AbiertaEn, CerradaEn: s.CerradaEn, Abierta: s.Abierta(), MontoInicial: s.MontoInicial,
		Notas: s.Notas, Version: s.Version, CantidadVentas: cantidad, Arqueo: make([]ArqueoDTO, len(arqueo)),
		Editada: editada(s)}
	var contado, dif dinero.Dinero
	completo := len(arqueo) > 0
	for i, a := range arqueo {
		d.Arqueo[i] = ArqueoDTO{Metodo: a.Metodo, Ventas: a.Ventas, Gastos: a.Gastos, Esperado: a.Esperado,
			Contado: a.Contado, Diferencia: a.Diferencia()}
		d.TotalEsperado = d.TotalEsperado.Sumar(a.Esperado)
		if a.Contado == nil {
			completo = false
			continue
		}
		contado = contado.Sumar(*a.Contado)
		dif = dif.Sumar(*a.Diferencia())
	}
	if completo {
		d.TotalContado, d.Diferencia = &contado, &dif
	}
	return d
}

// ServicioSesiones implementa apertura, cierre, conciliación y edición de la caja.
type ServicioSesiones struct {
	repo  RepositorioSesiones
	tx    tx.Transactor
	ids   id.Generador
	reloj reloj.Reloj
}

func NuevoServicioSesiones(repo RepositorioSesiones, t tx.Transactor, ids id.Generador, r reloj.Reloj) *ServicioSesiones {
	return &ServicioSesiones{repo: repo, tx: t, ids: ids, reloj: r}
}

// aDTO arma el DTO: si la caja está abierta calcula el arqueo en vivo hasta ahora.
func (s *ServicioSesiones) aDTO(ctx context.Context, ses domain.Sesion) (SesionDTO, error) {
	if !ses.Abierta() {
		return dtoSesion(ses, ses.Arqueo, ses.CantidadVentas), nil
	}
	t, n, err := s.repo.Totales(ctx, ses.AbiertaEn, s.reloj.Ahora().Add(time.Millisecond))
	if err != nil {
		return SesionDTO{}, err
	}
	return dtoSesion(ses, domain.CalcularArqueo(ses.MontoInicial, t, nil), n), nil
}

// Abrir abre la caja con el monto inicial en efectivo.
func (s *ServicioSesiones) Abrir(ctx context.Context, inicial dinero.Dinero, notas string) (SesionDTO, error) {
	ses, err := domain.AbrirSesion(s.ids.Nuevo(), inicial, notas, s.reloj.Ahora())
	if err != nil {
		return SesionDTO{}, err
	}
	if err := s.repo.Crear(ctx, ses); err != nil {
		return SesionDTO{}, err
	}
	return s.aDTO(ctx, ses)
}

// Actual devuelve la caja abierta con el arqueo en vivo, o nil.
func (s *ServicioSesiones) Actual(ctx context.Context) (*SesionDTO, error) {
	ses, err := s.repo.Abierta(ctx)
	if err != nil || ses == nil {
		return nil, err
	}
	d, err := s.aDTO(ctx, *ses)
	return &d, err
}

// ExigirAbierta devuelve domain.ErrCajaCerrada si no hay una caja abierta (para bloquear ventas, D-1).
func (s *ServicioSesiones) ExigirAbierta(ctx context.Context) error {
	ses, err := s.repo.Abierta(ctx)
	if err != nil {
		return err
	}
	if ses == nil {
		return domain.ErrCajaCerrada
	}
	return nil
}

// ComandoCierre cierra la caja con lo contado por medio.
type ComandoCierre struct {
	Version int
	Contado map[domain.Metodo]dinero.Dinero
	Notas   string
}

func (s *ServicioSesiones) Cerrar(ctx context.Context, id string, cmd ComandoCierre) (SesionDTO, error) {
	var ses domain.Sesion
	err := s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		var err error
		if ses, err = s.repo.Obtener(ctx, id); err != nil {
			return err
		}
		if !ses.Abierta() {
			return domain.ErrCajaYaCerrada
		}
		if ses.Version != cmd.Version {
			return domain.ErrVersionDesactualizada
		}
		ahora := s.reloj.Ahora()
		anterior := ses.Version
		// Primero se valida y se fija la hora de cierre; después se suman los totales de ese período exacto.
		if err := ses.Cerrar(nil, cmd.Contado, cmd.Notas, ahora); err != nil {
			return err
		}
		t, n, err := s.repo.Totales(ctx, ses.AbiertaEn, *ses.CerradaEn)
		if err != nil {
			return err
		}
		ses.Arqueo = domain.CalcularArqueo(ses.MontoInicial, t, cmd.Contado)
		ses.CantidadVentas = n
		return s.repo.Actualizar(ctx, ses, anterior)
	})
	if err != nil {
		return SesionDTO{}, err
	}
	return s.aDTO(ctx, ses)
}

// ComandoEdicion edita monto inicial, contado y notas (D-3). Contado se ignora si la caja está abierta.
type ComandoEdicion struct {
	Version      int
	MontoInicial dinero.Dinero
	Contado      map[domain.Metodo]dinero.Dinero
	Notas        string
	Motivo       string
}

func (s *ServicioSesiones) Editar(ctx context.Context, id string, cmd ComandoEdicion) (SesionDetalle, error) {
	err := s.tx.EnTransaccion(ctx, func(ctx context.Context) error {
		ses, err := s.repo.Obtener(ctx, id)
		if err != nil {
			return err
		}
		if ses.Version != cmd.Version {
			return domain.ErrVersionDesactualizada
		}
		var t domain.Totales
		n := ses.CantidadVentas
		if !ses.Abierta() {
			if t, n, err = s.repo.Totales(ctx, ses.AbiertaEn, *ses.CerradaEn); err != nil {
				return err
			}
		}
		ahora := s.reloj.Ahora()
		anterior := ses.Version
		cambios, err := ses.Editar(domain.Edicion{MontoInicial: cmd.MontoInicial, Contado: cmd.Contado,
			Notas: cmd.Notas, Motivo: cmd.Motivo}, t, ahora)
		if err != nil {
			return err
		}
		ses.CantidadVentas = n
		if err := s.repo.Actualizar(ctx, ses, anterior); err != nil {
			return err
		}
		return s.repo.GuardarEdicion(ctx, EdicionCaja{ID: s.ids.Nuevo(), SesionID: ses.ID, Fecha: ahora,
			Motivo: motivoNormalizado(cmd.Motivo), Version: ses.Version, Cambios: cambios})
	})
	if err != nil {
		return SesionDetalle{}, err
	}
	return s.Detalle(ctx, id)
}

// Detalle devuelve la caja con su arqueo e historial de ediciones (más reciente primero).
func (s *ServicioSesiones) Detalle(ctx context.Context, id string) (SesionDetalle, error) {
	ses, err := s.repo.Obtener(ctx, id)
	if err != nil {
		return SesionDetalle{}, err
	}
	d, err := s.aDTO(ctx, ses)
	if err != nil {
		return SesionDetalle{}, err
	}
	ed, err := s.repo.Ediciones(ctx, id)
	if err != nil {
		return SesionDetalle{}, err
	}
	if ed == nil {
		ed = []EdicionCaja{}
	}
	return SesionDetalle{SesionDTO: d, Ediciones: ed}, nil
}

// Listar devuelve el historial de cajas, de la más reciente a la más vieja.
func (s *ServicioSesiones) Listar(ctx context.Context, p paginacion.Solicitud) (paginacion.Pagina[SesionDTO], error) {
	lista, total, err := s.repo.Listar(ctx, p)
	if err != nil {
		return paginacion.Pagina[SesionDTO]{}, err
	}
	out := make([]SesionDTO, len(lista))
	for i, ses := range lista {
		if out[i], err = s.aDTO(ctx, ses); err != nil {
			return paginacion.Pagina[SesionDTO]{}, err
		}
	}
	return paginacion.NuevaPagina(out, p, total), nil
}

func motivoNormalizado(m string) string { return strings.Join(strings.Fields(m), " ") }

// editada: cada edición suma una versión además de la apertura (1) y el cierre (+1).
func editada(s domain.Sesion) bool {
	base := 1
	if !s.Abierta() {
		base = 2
	}
	return s.Version > base
}
