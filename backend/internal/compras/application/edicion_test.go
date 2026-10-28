package application

import (
	"errors"
	"testing"
	"time"

	"magui/internal/compras/domain"
	"magui/internal/kernel/errores"
	"magui/internal/kernel/fecha"
)

func dia(t *testing.T, s string) FechaEditada {
	t.Helper()
	f, err := fecha.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return FechaEditada{Dia: &f}
}

func compraBase(t *testing.T) (*Servicio, *repoMem, *catalogoMem, *inventarioMem, string) {
	t.Helper()
	s, repo, cat, inv := entorno()
	det, err := s.RegistrarCompra(ctx, ComandoCompra{EmpresaID: "natura", Items: []ItemCmd{{ProductoID: "kaiak", Cantidad: 3, CostoUnitario: d("100")}}})
	if err != nil {
		t.Fatal(err)
	}
	// El costo del catálogo es distinto al del renglón (p. ej. editado a mano): cambiar solo la fecha no debe pisarlo (RN-EC-16/19).
	costo := d("999")
	cat.productos["kaiak"] = ProductoCompra{Nombre: "Kaiak", Costo: &costo}
	cat.costos, inv.ingresos = nil, nil
	return s, repo, cat, inv, det.ID
}

func TestEditarCompraFeliz(t *testing.T) {
	s, repo, cat, inv, id := compraBase(t)
	det, err := s.EditarCompra(ctx, ComandoEditarCompra{ID: id, Version: 1, Fecha: dia(t, "2026-10-19"), EmpresaID: "natura",
		Notas: " Pedido ciclo 15 ", Motivo: "Error de carga", Origen: "web"})
	if err != nil {
		t.Fatal(err)
	}
	c := repo.compras[id]
	if c.Version != 2 || *c.Notas != "Pedido ciclo 15" || c.Fecha.Format("2006-01-02T15") != "2026-10-19T15" {
		t.Errorf("compra = %+v", c)
	}
	if len(repo.ediciones) != 1 || repo.ediciones[0].Version != 2 || repo.ediciones[0].Origen != domain.OrigenWeb ||
		*repo.ediciones[0].Motivo != "Error de carga" || len(repo.ediciones[0].Cambios.Cabecera) != 2 {
		t.Errorf("ediciones = %+v", repo.ediciones)
	}
	if len(det.Ediciones) != 1 || det.Ediciones[0].Renglones == nil {
		t.Errorf("detalle = %+v", det.Ediciones)
	}
	if len(cat.costos) != 0 || len(inv.ingresos) != 0 {
		t.Error("la edición de cabecera no debe tocar catálogo ni inventario")
	}
}

func TestEditarCompraSinCambiosNoEscribe(t *testing.T) {
	s, repo, _, _, id := compraBase(t)
	det, err := s.EditarCompra(ctx, ComandoEditarCompra{ID: id, Version: 1, Fecha: dia(t, "2026-10-25"), EmpresaID: "natura", Motivo: "x", Origen: "MOVIL"})
	if err != nil {
		t.Fatal(err)
	}
	if det.Version != 1 || len(repo.ediciones) != 0 || repo.escrituras != 0 || det.Ediciones == nil {
		t.Errorf("version=%d ediciones=%d escrituras=%d", det.Version, len(repo.ediciones), repo.escrituras)
	}
}

func TestEditarCompraErrores(t *testing.T) {
	s, repo, _, _, id := compraBase(t)
	base := ComandoEditarCompra{ID: id, Version: 1, Fecha: dia(t, "2026-10-19"), EmpresaID: "natura"}

	v := base
	v.Version = 5
	if _, err := s.EditarCompra(ctx, v); !errors.Is(err, domain.ErrVersionDesactualizada) {
		t.Errorf("versión: %v", err)
	}
	n := base
	n.ID = "nope"
	if _, err := s.EditarCompra(ctx, n); !errors.Is(err, domain.ErrCompraNoEncontrada) {
		t.Errorf("compra: %v", err)
	}
	e := base
	e.EmpresaID = "avon"
	if _, err := s.EditarCompra(ctx, e); !errors.Is(err, errores.NoEncontrado("empresa_no_encontrada", "")) {
		t.Errorf("empresa: %v", err)
	}
	it := base
	it.Items = &[]ItemEdicionCmd{}
	if ee, _ := errores.Como(func() error { _, err := s.EditarCompra(ctx, it); return err }()); ee == nil || ee.Campos["items"] == "" {
		t.Errorf("items vacío debería ser datos_invalidos")
	}
	f := base
	f.Fecha = dia(t, "2026-11-30")
	f.Version = 0
	f.Origen = "FAX"
	_, err := s.EditarCompra(ctx, f)
	if ee, _ := errores.Como(err); ee == nil || ee.Campos["fecha"] == "" || ee.Campos["version"] == "" || ee.Campos["origen"] == "" {
		t.Errorf("validación: %v", err)
	}
	if repo.compras[id].Version != 1 || len(repo.ediciones) != 0 {
		t.Error("nada debería haberse aplicado")
	}
}

func TestEditarCompraFallaLaAuditoria(t *testing.T) {
	s, repo, _, _, id := compraBase(t)
	repo.falloEdicion = errors.New("disco lleno")
	_, err := s.EditarCompra(ctx, ComandoEditarCompra{ID: id, Version: 1, Fecha: dia(t, "2026-10-19"), Notas: "x"})
	if err == nil || err.Error() != "disco lleno" {
		t.Fatalf("err = %v", err)
	}
	// El revertir real lo hace la transacción (ver mysql_test); acá se verifica que el error se propaga.
}

func TestEditarCompraInstanteISO(t *testing.T) {
	s, repo, _, _, id := compraBase(t)
	ins := time.Date(2026, 10, 20, 9, 30, 0, 0, time.UTC)
	if _, err := s.EditarCompra(ctx, ComandoEditarCompra{ID: id, Version: 1, Fecha: FechaEditada{Instante: &ins}, EmpresaID: "natura"}); err != nil {
		t.Fatal(err)
	}
	if !repo.compras[id].Fecha.Equal(ins) {
		t.Errorf("fecha = %v", repo.compras[id].Fecha)
	}
}
