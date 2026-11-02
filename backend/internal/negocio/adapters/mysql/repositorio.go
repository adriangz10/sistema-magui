// Package mysql guarda los datos del negocio en MySQL.
package mysql

import (
	"context"
	"database/sql"

	"magui/internal/negocio/domain"
	"magui/internal/platform/db"
)

type Repositorio struct{ DB *sql.DB }

func (r Repositorio) Obtener(ctx context.Context) (domain.Negocio, error) {
	var n domain.Negocio
	t := &n.Ticket
	err := db.Conn(ctx, r.DB).QueryRowContext(ctx,
		`SELECT nombre, datos_ticket, pie_ticket, ticket_habilitado, ticket_al_cobrar, ticket_ancho, ticket_logo,
		        logo IS NOT NULL, logo_version FROM negocio WHERE id = 1`).
		Scan(&n.Nombre, &t.Datos, &t.Pie, &t.Habilitado, &t.ImprimirAlCobrar, &t.AnchoPapel, &t.MostrarLogo, &n.TieneLogo, &n.LogoVersion)
	return n, err
}

func (r Repositorio) Guardar(ctx context.Context, n domain.Negocio) error {
	t := n.Ticket
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`UPDATE negocio SET nombre = ?, datos_ticket = ?, pie_ticket = ?, ticket_habilitado = ?, ticket_al_cobrar = ?,
		        ticket_ancho = ?, ticket_logo = ?, updated_at = CURRENT_TIMESTAMP(3) WHERE id = 1`,
		n.Nombre, t.Datos, t.Pie, t.Habilitado, t.ImprimirAlCobrar, t.AnchoPapel, t.MostrarLogo)
	return err
}

func (r Repositorio) GuardarLogo(ctx context.Context, datos []byte, tipo string) error {
	var tipoCol any
	if datos != nil {
		tipoCol = tipo
	}
	_, err := db.Conn(ctx, r.DB).ExecContext(ctx,
		`UPDATE negocio SET logo = ?, logo_tipo = ?, logo_version = logo_version + 1, updated_at = CURRENT_TIMESTAMP(3) WHERE id = 1`,
		datos, tipoCol)
	return err
}

func (r Repositorio) Logo(ctx context.Context) ([]byte, string, error) {
	var datos []byte
	var tipo sql.NullString
	if err := db.Conn(ctx, r.DB).QueryRowContext(ctx, `SELECT logo, logo_tipo FROM negocio WHERE id = 1`).Scan(&datos, &tipo); err != nil {
		return nil, "", err
	}
	if datos == nil {
		return nil, "", domain.ErrSinLogo
	}
	return datos, tipo.String, nil
}
