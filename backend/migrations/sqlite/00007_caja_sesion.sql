-- Apertura, cierre y conciliación de caja (US-002).

-- +goose Up
CREATE TABLE caja_sesion (
  id              TEXT NOT NULL PRIMARY KEY,
  abierta_en      TEXT NOT NULL,
  cerrada_en      TEXT NULL,
  monto_inicial   INTEGER NOT NULL CHECK (monto_inicial >= 0),
  notas           TEXT NULL,
  cantidad_ventas INTEGER NOT NULL DEFAULT 0,
  version         INTEGER NOT NULL DEFAULT 1,
  created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
-- Como máximo una caja abierta.
CREATE UNIQUE INDEX uq_cs_abierta ON caja_sesion ((cerrada_en IS NULL)) WHERE cerrada_en IS NULL;
CREATE INDEX idx_cs_abierta_en ON caja_sesion (abierta_en);

CREATE TABLE caja_arqueo (
  sesion_id TEXT NOT NULL REFERENCES caja_sesion(id) ON DELETE CASCADE,
  metodo    TEXT NOT NULL CHECK (metodo IN ('EFECTIVO','TRANSFERENCIA','QR','TARJETA')),
  ventas    INTEGER NOT NULL,
  gastos    INTEGER NOT NULL,
  esperado  INTEGER NOT NULL,
  contado   INTEGER NOT NULL CHECK (contado >= 0),
  PRIMARY KEY (sesion_id, metodo)
);

CREATE TABLE caja_edicion (
  id        TEXT NOT NULL PRIMARY KEY,
  sesion_id TEXT NOT NULL REFERENCES caja_sesion(id) ON DELETE CASCADE,
  fecha     TEXT NOT NULL,
  motivo    TEXT NOT NULL,
  version   INTEGER NOT NULL,
  cambios   TEXT NOT NULL CHECK (json_valid(cambios)),
  UNIQUE (sesion_id, version)
);

-- +goose Down
DROP TABLE IF EXISTS caja_edicion;
DROP TABLE IF EXISTS caja_arqueo;
DROP INDEX IF EXISTS idx_cs_abierta_en;
DROP INDEX IF EXISTS uq_cs_abierta;
DROP TABLE IF EXISTS caja_sesion;
