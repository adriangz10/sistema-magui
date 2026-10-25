-- Contraseña única de la dueña (fila única id = 1). Se guarda solo el hash bcrypt.

-- +goose Up
CREATE TABLE credencial_acceso (
  id             INTEGER NOT NULL PRIMARY KEY CHECK (id = 1),
  hash           TEXT NOT NULL,
  version        INTEGER NOT NULL DEFAULT 1,
  actualizada_en TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- +goose Down
DROP TABLE IF EXISTS credencial_acceso;
