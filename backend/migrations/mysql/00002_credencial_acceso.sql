-- Contraseña única de la dueña (fila única id = 1). Se guarda solo el hash bcrypt.
-- version aumenta al cambiar la contraseña e invalida las sesiones anteriores.

-- +goose Up
CREATE TABLE credencial_acceso (
  id             TINYINT UNSIGNED NOT NULL PRIMARY KEY,
  hash           VARCHAR(100) NOT NULL,
  version        INT UNSIGNED NOT NULL DEFAULT 1,
  actualizada_en DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT ck_credencial_unica CHECK (id = 1)
);

-- +goose Down
DROP TABLE IF EXISTS credencial_acceso;
