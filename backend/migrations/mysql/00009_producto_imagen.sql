-- Foto opcional de cada producto (US-004).

-- +goose Up
ALTER TABLE producto ADD COLUMN imagen_version INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Sube con cada foto nueva y nunca se reutiliza (caché). Hay foto si existe producto_imagen';

CREATE TABLE producto_imagen (
  producto_id CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  datos       MEDIUMBLOB NOT NULL,
  tipo        VARCHAR(30) NOT NULL,
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_pi_producto FOREIGN KEY (producto_id) REFERENCES producto(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS producto_imagen;
ALTER TABLE producto DROP COLUMN imagen_version;
