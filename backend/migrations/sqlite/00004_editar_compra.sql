-- Editar compras (US-001a): control de versión y auditoría de ediciones.

-- +goose Up
ALTER TABLE compra ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
-- SQLite no admite un DEFAULT no constante en ADD COLUMN: la aplicación siempre informa updated_at.
ALTER TABLE compra ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';
UPDATE compra SET updated_at = created_at;

CREATE TABLE compra_edicion (
  id        TEXT NOT NULL PRIMARY KEY,
  compra_id TEXT NOT NULL REFERENCES compra(id) ON DELETE CASCADE,
  fecha     TEXT NOT NULL,
  origen    TEXT NOT NULL CHECK (origen IN ('WEB','MOVIL','DESKTOP')),
  motivo    TEXT NULL,
  version   INTEGER NOT NULL,
  cambios   TEXT NOT NULL CHECK (json_valid(cambios)),
  UNIQUE (compra_id, version)
);
CREATE INDEX idx_lote_compra_item ON lote (compra_item_id);

-- +goose Down
DROP INDEX IF EXISTS idx_lote_compra_item;
DROP TABLE IF EXISTS compra_edicion;
ALTER TABLE compra DROP COLUMN updated_at;
ALTER TABLE compra DROP COLUMN version;
