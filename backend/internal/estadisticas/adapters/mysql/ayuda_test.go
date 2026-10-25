package mysql_test

import (
	"time"

	invapp "magui/internal/inventario/application"
	"magui/internal/kernel/fecha"
	"magui/internal/kernel/reloj"
)

// invAjuste carga 5 unidades sin costo (para probar ventas con ganancia desconocida).
func invAjuste(productoID string) invapp.ComandoAjuste {
	return invapp.ComandoAjuste{ProductoID: productoID, Cantidad: 5, Motivo: "Muestras gratis"}
}

func fechaDe(t time.Time) fecha.Fecha { return fecha.DelInstante(t, reloj.ZonaArgentina) }
