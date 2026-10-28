-- Renglones de compra quitados al editar (US-001c, ADR 0004). Baja logica: el lote sigue apuntando al renglon.

-- +goose Up
ALTER TABLE compra_item ADD COLUMN quitado_en DATETIME(3) NULL;

-- +goose Down
-- ATENCION: si hay renglones quitados, al bajar la columna "reviven" (ver docs/data/editar-compra.md, Rollback).
ALTER TABLE compra_item DROP COLUMN quitado_en;
