-- Renglones de compra quitados al editar (US-001c, ADR 0004).

-- +goose Up
ALTER TABLE compra_item ADD COLUMN quitado_en TEXT NULL;

-- +goose Down
ALTER TABLE compra_item DROP COLUMN quitado_en;
