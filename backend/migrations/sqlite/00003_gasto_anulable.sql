-- Los gastos de caja no se borran: se anulan.

-- +goose Up
ALTER TABLE gasto_caja ADD COLUMN anulado_en TEXT NULL;
ALTER TABLE gasto_caja ADD COLUMN motivo_anulacion TEXT NULL;
CREATE INDEX idx_gasto_categoria ON gasto_caja (categoria);

-- +goose Down
DROP INDEX IF EXISTS idx_gasto_categoria;
ALTER TABLE gasto_caja DROP COLUMN motivo_anulacion;
ALTER TABLE gasto_caja DROP COLUMN anulado_en;
