-- Movimiento de stock por edición de una compra (US-001b, ADR 0003).

-- +goose Up
-- Agregar un valor al final del ENUM es solo metadatos; INSTANT hace fallar la migración si no lo fuera.
ALTER TABLE movimiento_stock
  MODIFY COLUMN tipo ENUM('INICIAL','COMPRA','VENTA','ANULACION','AJUSTE','EDICION_COMPRA') NOT NULL,
  ALGORITHM=INSTANT;

-- +goose Down
-- No se puede quitar el valor si hay filas: se pasan a AJUSTE marcando la nota (el stock no cambia).
UPDATE movimiento_stock
   SET tipo = 'AJUSTE', nota = LEFT(CONCAT('[EDICION_COMPRA] ', COALESCE(nota, '')), 255)
 WHERE tipo = 'EDICION_COMPRA';
ALTER TABLE movimiento_stock
  MODIFY COLUMN tipo ENUM('INICIAL','COMPRA','VENTA','ANULACION','AJUSTE') NOT NULL;
