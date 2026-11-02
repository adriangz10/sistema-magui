-- Apertura, cierre y conciliación de caja (US-002).

-- +goose Up
CREATE TABLE caja_sesion (
  id              CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  abierta_en      DATETIME(3) NOT NULL,
  cerrada_en      DATETIME(3) NULL,
  monto_inicial   DECIMAL(12,2) NOT NULL COMMENT 'Efectivo con el que se abrió',
  notas           VARCHAR(500) NULL,
  cantidad_ventas INT NOT NULL DEFAULT 0 COMMENT 'Ventas confirmadas del período (al cerrar o editar)',
  version         INT UNSIGNED NOT NULL DEFAULT 1,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  -- 1 mientras está abierta y NULL al cerrar: con el índice único hay como máximo una caja abierta.
  abierta         TINYINT AS (IF(cerrada_en IS NULL, 1, NULL)) STORED,
  CONSTRAINT ck_cs_monto_inicial CHECK (monto_inicial >= 0),
  CONSTRAINT uq_cs_abierta UNIQUE (abierta),
  INDEX idx_cs_abierta_en (abierta_en)
);

-- Arqueo por medio de pago, guardado al cerrar (y recalculado al editar).
CREATE TABLE caja_arqueo (
  sesion_id CHAR(36) CHARACTER SET ascii NOT NULL,
  metodo    ENUM('EFECTIVO','TRANSFERENCIA','QR','TARJETA') NOT NULL,
  ventas    DECIMAL(12,2) NOT NULL,
  gastos    DECIMAL(12,2) NOT NULL,
  esperado  DECIMAL(12,2) NOT NULL COMMENT 'Según el sistema (en efectivo incluye el monto inicial)',
  contado   DECIMAL(12,2) NOT NULL COMMENT 'Lo que contó la dueña',
  PRIMARY KEY (sesion_id, metodo),
  CONSTRAINT fk_ca_sesion FOREIGN KEY (sesion_id) REFERENCES caja_sesion(id) ON DELETE CASCADE,
  CONSTRAINT ck_ca_contado CHECK (contado >= 0)
);

-- Historial de ediciones (mismo criterio que compra_edicion, ADR 0005).
CREATE TABLE caja_edicion (
  id        CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  sesion_id CHAR(36) CHARACTER SET ascii NOT NULL,
  fecha     DATETIME(3) NOT NULL,
  motivo    VARCHAR(255) NOT NULL,
  version   INT UNSIGNED NOT NULL COMMENT 'Versión de la caja que dejó esta edición',
  cambios   JSON NOT NULL COMMENT 'Lista de {campo, antes, despues}',
  CONSTRAINT fk_cje_sesion FOREIGN KEY (sesion_id) REFERENCES caja_sesion(id) ON DELETE CASCADE,
  CONSTRAINT uq_cje_sesion_version UNIQUE (sesion_id, version)
);

-- +goose Down
DROP TABLE IF EXISTS caja_edicion;
DROP TABLE IF EXISTS caja_arqueo;
DROP TABLE IF EXISTS caja_sesion;
