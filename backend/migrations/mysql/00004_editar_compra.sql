-- Editar compras (US-001a): control de versión y auditoría de ediciones.

-- +goose Up
ALTER TABLE compra
  ADD COLUMN version    INT UNSIGNED NOT NULL DEFAULT 1,
  ADD COLUMN updated_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3);

-- Las compras existentes nunca se editaron: updated_at = created_at (la aplicación siempre informa updated_at al editar).
UPDATE compra SET updated_at = created_at;

CREATE TABLE compra_edicion (
  id        CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  compra_id CHAR(36) CHARACTER SET ascii NOT NULL,
  fecha     DATETIME(3) NOT NULL,
  origen    ENUM('WEB','MOVIL','DESKTOP') NOT NULL,
  motivo    VARCHAR(255) NULL,
  version   INT UNSIGNED NOT NULL COMMENT 'Versión de la compra que dejó esta edición',
  cambios   JSON NOT NULL COMMENT 'Documento v1: cabecera y renglones con antes/después',
  CONSTRAINT fk_ce_compra FOREIGN KEY (compra_id) REFERENCES compra(id) ON DELETE CASCADE,
  CONSTRAINT uq_ce_compra_version UNIQUE (compra_id, version)
);

-- +goose Down
DROP TABLE IF EXISTS compra_edicion;
ALTER TABLE compra
  DROP COLUMN updated_at,
  DROP COLUMN version;
