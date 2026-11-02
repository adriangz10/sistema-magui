-- Foto opcional de cada producto (US-004).

-- +goose Up
ALTER TABLE producto ADD COLUMN imagen_version INTEGER NOT NULL DEFAULT 0;

CREATE TABLE producto_imagen (
  producto_id TEXT NOT NULL PRIMARY KEY REFERENCES producto(id) ON DELETE CASCADE,
  datos       BLOB NOT NULL,
  tipo        TEXT NOT NULL,
  updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- +goose Down
DROP TABLE IF EXISTS producto_imagen;
ALTER TABLE producto DROP COLUMN imagen_version;
