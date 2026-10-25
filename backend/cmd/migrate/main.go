// Comando migrate: administra el esquema de la base e importa el Excel histórico.
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate status
//	go run ./cmd/migrate down
//	go run ./cmd/migrate reset
//	go run ./cmd/migrate -empresa Natura importar-excel "ruta/al/archivo.xlsx"
//	go run ./cmd/migrate -driver sqlite -sqlite magui.db up
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	"magui/internal/importacion"
	"magui/internal/platform/config"
	"magui/internal/platform/db"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	driver := flag.String("driver", "mysql", "motor: mysql o sqlite")
	sqlitePath := flag.String("sqlite", "magui.db", "archivo SQLite (solo con -driver sqlite)")
	empresa := flag.String("empresa", "", "empresa a asignar a los productos importados (ej. Natura)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "uso: migrate [flags] up|down|status|reset|importar-excel <archivo.xlsx>")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		return fmt.Errorf("falta el comando")
	}

	ctx := context.Background()
	conn, dialect, err := abrir(ctx, *driver, *sqlitePath)
	if err != nil {
		return err
	}
	defer conn.Close()

	m, err := db.NewMigrator(conn, dialect)
	if err != nil {
		return err
	}

	switch cmd := flag.Arg(0); cmd {
	case "up":
		if err := m.Up(ctx); err != nil {
			return err
		}
		v, _ := m.Version(ctx)
		fmt.Printf("esquema actualizado (versión %d)\n", v)
	case "down":
		return m.Down(ctx)
	case "reset":
		return m.Reset(ctx)
	case "status":
		return m.PrintStatus(ctx, os.Stdout)
	case "importar-excel":
		if dialect != db.MySQL {
			return fmt.Errorf("la importación se hace en MySQL; el escritorio recibe los datos por sincronización")
		}
		if flag.NArg() < 2 {
			return fmt.Errorf("falta la ruta del archivo .xlsx")
		}
		return importarExcel(ctx, conn, flag.Arg(1), *empresa)
	default:
		flag.Usage()
		return fmt.Errorf("comando desconocido %q", cmd)
	}
	return nil
}

func abrir(ctx context.Context, driver, sqlitePath string) (*sql.DB, db.Dialect, error) {
	switch db.Dialect(driver) {
	case db.MySQL:
		cfg, err := config.Load()
		if err != nil {
			return nil, "", err
		}
		conn, err := db.OpenMySQL(ctx, cfg.DB.DSN())
		return conn, db.MySQL, err
	case db.SQLite:
		conn, err := db.OpenSQLite(ctx, sqlitePath)
		return conn, db.SQLite, err
	default:
		return nil, "", fmt.Errorf("driver desconocido %q (use mysql o sqlite)", driver)
	}
}

func importarExcel(ctx context.Context, conn *sql.DB, ruta, empresa string) error {
	filas, err := importacion.LeerExcel(ruta)
	if err != nil {
		return err
	}
	fmt.Printf("filas leídas del Excel: %d\n", len(filas))
	res, err := importacion.Importar(ctx, conn, filas, importacion.Opciones{Empresa: empresa})
	if err != nil {
		return err
	}
	fmt.Printf("productos nuevos: %d | ya existían: %d | categorías nuevas: %d\n", res.Productos, res.Omitidos, res.Categorias)
	fmt.Printf("lotes: %d (con vencimiento: %d) | unidades en stock: %d\n", res.Lotes, res.ConVencimiento, res.Unidades)
	return nil
}
