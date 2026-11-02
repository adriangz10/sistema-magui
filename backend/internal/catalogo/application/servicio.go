package application

import (
	"magui/internal/kernel/id"
	"magui/internal/kernel/reloj"
	"magui/internal/kernel/tx"
)

// Servicio reúne los casos de uso del catálogo.
type Servicio struct {
	empresas   RepositorioEmpresas
	categorias RepositorioCategorias
	productos  RepositorioProductos
	imagenes   RepositorioImagenes
	tx         tx.Transactor
	ids        id.Generador
	reloj      reloj.Reloj
}

// Dependencias del servicio.
type Dependencias struct {
	Empresas   RepositorioEmpresas
	Categorias RepositorioCategorias
	Productos  RepositorioProductos
	Imagenes   RepositorioImagenes // fotos de productos (opcional en tests)
	Tx         tx.Transactor
	IDs        id.Generador
	Reloj      reloj.Reloj
}

func NuevoServicio(d Dependencias) *Servicio {
	return &Servicio{
		empresas: d.Empresas, categorias: d.Categorias, productos: d.Productos, imagenes: d.Imagenes,
		tx: d.Tx, ids: d.IDs, reloj: d.Reloj,
	}
}
