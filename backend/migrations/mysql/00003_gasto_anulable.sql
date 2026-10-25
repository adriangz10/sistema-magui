-- Los gastos de caja no se borran: se anulan (queda registro y la sincronización offline sigue
-- siendo de solo agregar).

-- +goose Up
ALTER TABLE gasto_caja
  ADD COLUMN anulado_en DATETIME(3) NULL,
  ADD COLUMN motivo_anulacion VARCHAR(255) NULL,
  ADD INDEX idx_gasto_categoria (categoria);

-- +goose Down
ALTER TABLE gasto_caja
  DROP INDEX idx_gasto_categoria,
  DROP COLUMN motivo_anulacion,
  DROP COLUMN anulado_en;
