// Comando admin: tareas que requieren acceso al servidor (no se exponen por internet).
//
//	go run ./cmd/admin establecer-clave
//
// Pide la contraseña sin mostrarla. También acepta la contraseña por entrada estándar:
//
//	echo "nueva-clave" | go run ./cmd/admin establecer-clave
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"magui/internal/auth"
	"magui/internal/platform/config"
	"magui/internal/platform/db"
	"magui/internal/platform/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 || os.Args[1] != "establecer-clave" {
		return errors.New("uso: admin establecer-clave")
	}
	clave, err := leerClave()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn, err := db.OpenMySQL(ctx, cfg.DB.DSN())
	if err != nil {
		return err
	}
	defer conn.Close()

	m, err := db.NewMigrator(conn, db.MySQL)
	if err != nil {
		return err
	}
	if err := m.Up(ctx); err != nil {
		return err
	}

	acceso, err := auth.NuevoModuloMySQL(conn, cfg.Auth.Secreto, cfg.Auth.DuracionSesion, logger.New(cfg.Env))
	if err != nil {
		return err
	}
	if err := acceso.Servicio.EstablecerClave(ctx, clave); err != nil {
		return err
	}
	fmt.Println("contraseña guardada. Las sesiones abiertas anteriormente quedaron cerradas.")
	return nil
}

func leerClave() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		linea, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && linea == "" {
			return "", errors.New("no se recibió la contraseña por entrada estándar")
		}
		return strings.TrimRight(linea, "\r\n"), nil
	}
	fmt.Fprint(os.Stderr, "Nueva contraseña: ")
	a, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	fmt.Fprint(os.Stderr, "Repetila: ")
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	if string(a) != string(b) {
		return "", errors.New("las contraseñas no coinciden")
	}
	return string(a), nil
}
