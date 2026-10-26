package application

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"magui/internal/kernel/reloj"
	sincro "magui/internal/sync/application"
)

// Tiempos de la sincronización.
const (
	IntervaloNormal = time.Minute     // con conexión: baja la copia del catálogo cada minuto
	EsperaMinima    = 5 * time.Second // sin conexión: primer reintento
	EsperaMaxima    = time.Minute     // sin conexión: los reintentos se espacian hasta este tope
)

// Estado es lo que muestra la pantalla ("3 ventas pendientes de subir").
type Estado struct {
	EnLinea    bool `json:"enLinea"`
	Pendientes int  `json:"pendientes"`
	Rechazadas int  `json:"rechazadas"`
	// NecesitaIngresar: no hay sesión válida con el servidor; las ventas esperan hasta que se ingrese.
	NecesitaIngresar bool       `json:"necesitaIngresar"`
	UltimaSincro     *time.Time `json:"ultimaSincronizacion"`
	CopiaDel         *time.Time `json:"copiaDelCatalogo"`
	UltimoError      string     `json:"ultimoError,omitempty"`
}

// Sincronizador sube las ventas locales y baja la copia del catálogo.
type Sincronizador struct {
	cola     Cola
	servidor Servidor
	sesiones Sesiones
	conexion *Conexion
	reloj    reloj.Reloj
	log      *slog.Logger

	mu        sync.Mutex // una sincronización a la vez
	avisos    chan struct{}
	estMu     sync.Mutex
	ultima    *time.Time
	error     string
	sinSesion bool
}

func NuevoSincronizador(cola Cola, srv Servidor, ses Sesiones, con *Conexion, r reloj.Reloj, log *slog.Logger) *Sincronizador {
	return &Sincronizador{cola: cola, servidor: srv, sesiones: ses, conexion: con, reloj: r, log: log, avisos: make(chan struct{}, 1)}
}

// Avisar pide sincronizar cuanto antes (después de una venta, al ingresar o al volver la conexión).
func (s *Sincronizador) Avisar() {
	select {
	case s.avisos <- struct{}{}:
	default: // ya hay un aviso pendiente
	}
}

// Sincronizar sube todas las ventas pendientes (de a tandas) y después baja la copia del catálogo.
// En ese orden: así la copia ya incluye las ventas recién subidas.
func (s *Sincronizador) Sincronizar(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.sincronizar(ctx)
	s.registrar(err)
	return err
}

func (s *Sincronizador) sincronizar(ctx context.Context) error {
	ses, err := s.sesiones.Leer(ctx)
	if err != nil {
		return err
	}
	if ses.Token == "" || !ses.ExpiraEn.After(s.reloj.Ahora()) {
		return ErrSesionVencida
	}
	for {
		pend, err := s.cola.Pendientes(ctx, sincro.MaxVentasPorEnvio)
		if err != nil || len(pend) == 0 {
			if err != nil {
				return err
			}
			break
		}
		res, err := s.servidor.EnviarVentas(ctx, ses.Token, pend)
		if err != nil {
			return err
		}
		var enviadas []string
		for _, r := range res.Resultados {
			switch r.Estado {
			case sincro.Registrada, sincro.YaExistia:
				enviadas = append(enviadas, r.ID)
			case sincro.Rechazada:
				motivo := "El servidor la rechazó"
				if r.Error != nil {
					motivo = r.Error.Message
				}
				s.log.Warn("venta rechazada por el servidor", "venta", r.ID, "motivo", motivo)
				if err := s.cola.MarcarRechazada(ctx, r.ID, motivo); err != nil {
					return err
				}
			}
		}
		if err := s.cola.MarcarEnviadas(ctx, enviadas, s.reloj.Ahora()); err != nil {
			return err
		}
		if len(pend) < sincro.MaxVentasPorEnvio {
			break
		}
	}
	copia, err := s.servidor.CopiaCatalogo(ctx, ses.Token)
	if err != nil {
		return err
	}
	return s.cola.ReemplazarCopia(ctx, copia)
}

func (s *Sincronizador) registrar(err error) {
	ahora := s.reloj.Ahora()
	s.estMu.Lock()
	defer s.estMu.Unlock()
	s.sinSesion = errors.Is(err, ErrSesionVencida)
	switch {
	case err == nil:
		s.ultima, s.error = &ahora, ""
		s.conexion.Informar(true, ahora)
	case errors.Is(err, ErrSinConexion):
		s.error = "Sin conexión con el servidor"
		s.conexion.Informar(false, ahora)
	case s.sinSesion:
		s.error = "Ingresá con conexión para subir las ventas"
		s.conexion.Informar(true, ahora) // el servidor respondió (o no hace falta llegar)
	default:
		s.error = err.Error()
		s.log.Error("sincronización", "error", err)
	}
}

// Estado arma el resumen para la pantalla.
func (s *Sincronizador) Estado(ctx context.Context) (Estado, error) {
	pend, rech, err := s.cola.Contar(ctx)
	if err != nil {
		return Estado{}, err
	}
	hay, del, err := s.cola.HayCopia(ctx)
	if err != nil {
		return Estado{}, err
	}
	s.estMu.Lock()
	defer s.estMu.Unlock()
	e := Estado{EnLinea: s.conexion.EnLinea(), Pendientes: pend, Rechazadas: rech, NecesitaIngresar: s.sinSesion,
		UltimaSincro: s.ultima, UltimoError: s.error}
	if hay {
		e.CopiaDel = &del
	}
	return e, nil
}

// Ejecutar sincroniza en segundo plano hasta que se cancele ctx: cada minuto con conexión,
// con reintentos cada vez más espaciados sin conexión, y enseguida ante cada aviso.
func (s *Sincronizador) Ejecutar(ctx context.Context) {
	espera := EsperaMinima
	for {
		err := s.Sincronizar(ctx)
		siguiente := IntervaloNormal
		if errors.Is(err, ErrSinConexion) {
			siguiente = espera
			espera = min(espera*2, EsperaMaxima)
		} else {
			espera = EsperaMinima
		}
		t := time.NewTimer(siguiente)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		case <-s.avisos:
			t.Stop()
		}
	}
}
