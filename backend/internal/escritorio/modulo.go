package escritorio

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"magui/internal/auth/adapters/seguridad"
	localhttp "magui/internal/escritorio/adapters/http"
	"magui/internal/escritorio/adapters/remoto"
	"magui/internal/escritorio/adapters/sqlite"
	"magui/internal/escritorio/application"
	"magui/internal/kernel/id"
	"magui/internal/kernel/reloj"
	"magui/internal/platform/db"
	ventashttp "magui/internal/ventas/adapters/http"
	ventas "magui/internal/ventas/application"
)

// Config de la app de escritorio.
type Config struct {
	Servidor string       // dirección del servidor central, ej. https://magui.ejemplo.com
	Datos    string       // archivo SQLite local, ej. %APPDATA%\Magui\magui.db
	Frontend http.Handler // opcional: sirve la app compilada
	Log      *slog.Logger
	// Transporte hacia el servidor (en tests, uno falso). Nil = el normal.
	Transporte http.RoundTripper
}

// App es el escritorio armado: el handler que usa la pantalla y la sincronización en segundo plano.
type App struct {
	Handler http.Handler
	Sincro  *application.Sincronizador
	db      *sql.DB
}

// claves adapta bcrypt al puerto del escritorio (costo menor: es solo para ingresar sin conexión).
type claves struct{ seguridad.Bcrypt }

func (c claves) Cifrar(clave string) (string, error) { return c.Hashear(clave) }

// Abrir crea o abre la base local, aplica sus migraciones y arma la app.
func Abrir(ctx context.Context, cfg Config) (*App, error) {
	srv, err := url.Parse(strings.TrimRight(strings.TrimSpace(cfg.Servidor), "/"))
	if err != nil || (srv.Scheme != "http" && srv.Scheme != "https") || srv.Host == "" {
		return nil, fmt.Errorf("dirección del servidor inválida %q: use por ejemplo https://magui.ejemplo.com", cfg.Servidor)
	}
	conn, err := db.OpenSQLite(ctx, cfg.Datos)
	if err != nil {
		return nil, err
	}
	m, err := db.NewMigrator(conn, db.Escritorio)
	if err == nil {
		err = m.Up(ctx)
	}
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("preparar la base local: %w", err)
	}

	r := reloj.Sistema{}
	base := sqlite.Base{DB: conn}
	sesiones := sqlite.Sesiones{DB: conn}
	cliente := remoto.NuevoCliente(srv.String())
	if cfg.Transporte != nil {
		cliente.HTTP.Transport = cfg.Transporte
	}

	var sincro *application.Sincronizador
	conexion := application.NuevaConexion(func() { sincro.Avisar() })
	sincro = application.NuevoSincronizador(base, cliente, sesiones, conexion, r, cfg.Log)

	// Mismo caso de uso de ventas que el servidor, sobre la base local.
	svcVentas := ventas.NuevoServicio(ventas.Dependencias{
		Repo:       sqlite.RepositorioVentas{DB: conn},
		Catalogo:   sqlite.CatalogoVentas{DB: conn},
		Inventario: sqlite.InventarioVentas{DB: conn},
		Tx:         db.Transactor{DB: conn},
		IDs:        id.UUIDv7{},
		Reloj:      r,
	})

	h := localhttp.NuevoHandler(localhttp.Opciones{
		Servidor:   srv,
		Ventas:     ventashttp.NuevoHandler(svcVentas, cfg.Log),
		Catalogo:   application.NuevoCatalogoLocal(base),
		Sincro:     sincro,
		Ingreso:    application.NuevoIngreso(sesiones, claves{seguridad.Bcrypt{Costo: 10}}, r),
		Conexion:   conexion,
		Reloj:      r,
		Log:        cfg.Log,
		Frontend:   cfg.Frontend,
		Transporte: cfg.Transporte,
	})
	return &App{Handler: h, Sincro: sincro, db: conn}, nil
}

// Ejecutar corre la sincronización en segundo plano hasta que se cancele ctx.
func (a *App) Ejecutar(ctx context.Context) { a.Sincro.Ejecutar(ctx) }

// Cerrar cierra la base local.
func (a *App) Cerrar() error { return a.db.Close() }
