-- Esquema v2 del Sistema Magui (SQLite, app de escritorio offline).
-- Mismas tablas y columnas que MySQL, con estas diferencias:
--   * Dinero: INTEGER en CENTAVOS (59200.00 -> 5920000) para que las sumas sean exactas.
--   * Fechas: TEXT ISO-8601 en UTC ('2026-10-25T18:00:00.000Z'); fechas sin hora 'YYYY-MM-DD'.
--   * Booleanos: INTEGER 0/1.  ENUM: TEXT + CHECK.  updated_at lo actualiza la aplicación.

-- +goose Up

CREATE TABLE empresa (
  id         TEXT NOT NULL PRIMARY KEY,
  nombre     TEXT NOT NULL COLLATE NOCASE UNIQUE,
  activo     INTEGER NOT NULL DEFAULT 1 CHECK (activo IN (0,1)),
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  version    INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE categoria (
  id         TEXT NOT NULL PRIMARY KEY,
  nombre     TEXT NOT NULL COLLATE NOCASE,
  padre_id   TEXT NULL REFERENCES categoria(id),
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  version    INTEGER NOT NULL DEFAULT 1,
  CHECK (padre_id IS NULL OR padre_id <> id)
);
CREATE UNIQUE INDEX uq_categoria_nombre ON categoria (COALESCE(padre_id, ''), nombre);

CREATE TABLE producto (
  id                TEXT NOT NULL PRIMARY KEY,
  sku               TEXT NOT NULL UNIQUE,
  codigo_barras     TEXT NULL UNIQUE,
  nombre            TEXT NOT NULL,
  descripcion       TEXT NULL,
  empresa_id        TEXT NULL REFERENCES empresa(id),
  tipo              TEXT NOT NULL DEFAULT 'SIMPLE' CHECK (tipo IN ('SIMPLE','COMBO')),
  precio_venta      INTEGER NOT NULL CHECK (precio_venta >= 0),
  costo             INTEGER NULL CHECK (costo IS NULL OR costo >= 0),
  stock_minimo      INTEGER NOT NULL DEFAULT 0 CHECK (stock_minimo >= 0),
  peso_kg           REAL NULL,
  alto_cm           REAL NULL,
  ancho_cm          REAL NULL,
  profundidad_cm    REAL NULL,
  mostrar_en_tienda INTEGER NOT NULL DEFAULT 1 CHECK (mostrar_en_tienda IN (0,1)),
  activo            INTEGER NOT NULL DEFAULT 1 CHECK (activo IN (0,1)),
  created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  version           INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX idx_producto_nombre ON producto (nombre);
CREATE INDEX idx_producto_updated ON producto (updated_at);

CREATE TABLE producto_categoria (
  producto_id  TEXT NOT NULL REFERENCES producto(id) ON DELETE CASCADE,
  categoria_id TEXT NOT NULL REFERENCES categoria(id),
  PRIMARY KEY (producto_id, categoria_id)
);
CREATE INDEX idx_pc_categoria ON producto_categoria (categoria_id);

CREATE TABLE combo_item (
  combo_id    TEXT NOT NULL REFERENCES producto(id) ON DELETE CASCADE,
  producto_id TEXT NOT NULL REFERENCES producto(id),
  cantidad    INTEGER NOT NULL CHECK (cantidad > 0),
  PRIMARY KEY (combo_id, producto_id),
  CHECK (combo_id <> producto_id)
);

CREATE TABLE precio_historial (
  id              TEXT NOT NULL PRIMARY KEY,
  producto_id     TEXT NOT NULL REFERENCES producto(id) ON DELETE CASCADE,
  precio_anterior INTEGER NOT NULL,
  precio_nuevo    INTEGER NOT NULL,
  origen          TEXT NOT NULL CHECK (origen IN ('MANUAL','CASCADA','COMPRA')),
  motivo          TEXT NULL,
  fecha           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_ph_producto_fecha ON precio_historial (producto_id, fecha);

CREATE TABLE compra (
  id         TEXT NOT NULL PRIMARY KEY,
  fecha      TEXT NOT NULL,
  empresa_id TEXT NULL REFERENCES empresa(id),
  total      INTEGER NOT NULL CHECK (total >= 0),
  notas      TEXT NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_compra_fecha ON compra (fecha);

CREATE TABLE compra_item (
  id                 TEXT NOT NULL PRIMARY KEY,
  compra_id          TEXT NOT NULL REFERENCES compra(id) ON DELETE CASCADE,
  producto_id        TEXT NOT NULL REFERENCES producto(id),
  cantidad           INTEGER NOT NULL CHECK (cantidad > 0),
  costo_unitario     INTEGER NOT NULL CHECK (costo_unitario >= 0),
  precio_venta_nuevo INTEGER NULL,
  fecha_vencimiento  TEXT NULL
);
CREATE INDEX idx_ci_producto ON compra_item (producto_id);

CREATE TABLE lote (
  id                TEXT NOT NULL PRIMARY KEY,
  producto_id       TEXT NOT NULL REFERENCES producto(id),
  fecha_vencimiento TEXT NULL,
  costo_unitario    INTEGER NULL,
  compra_item_id    TEXT NULL REFERENCES compra_item(id),
  created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_lote_producto_venc ON lote (producto_id, fecha_vencimiento);
CREATE INDEX idx_lote_vencimiento ON lote (fecha_vencimiento);

CREATE TABLE venta (
  id               TEXT NOT NULL PRIMARY KEY,
  fecha            TEXT NOT NULL,
  total            INTEGER NOT NULL CHECK (total >= 0),
  costo_total      INTEGER NOT NULL DEFAULT 0,
  estado           TEXT NOT NULL DEFAULT 'CONFIRMADA' CHECK (estado IN ('CONFIRMADA','ANULADA')),
  origen           TEXT NOT NULL CHECK (origen IN ('WEB','MOVIL','DESKTOP')),
  anulada_en       TEXT NULL,
  motivo_anulacion TEXT NULL,
  created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_venta_fecha ON venta (fecha);
CREATE INDEX idx_venta_estado_fecha ON venta (estado, fecha);

CREATE TABLE venta_item (
  id              TEXT NOT NULL PRIMARY KEY,
  venta_id        TEXT NOT NULL REFERENCES venta(id) ON DELETE CASCADE,
  producto_id     TEXT NOT NULL REFERENCES producto(id),
  cantidad        INTEGER NOT NULL CHECK (cantidad > 0),
  precio_unitario INTEGER NOT NULL,
  costo_unitario  INTEGER NULL,
  subtotal        INTEGER NOT NULL
);
CREATE INDEX idx_vi_venta ON venta_item (venta_id);
CREATE INDEX idx_vi_producto ON venta_item (producto_id);

CREATE TABLE venta_pago (
  id             TEXT NOT NULL PRIMARY KEY,
  venta_id       TEXT NOT NULL REFERENCES venta(id) ON DELETE CASCADE,
  metodo         TEXT NOT NULL CHECK (metodo IN ('EFECTIVO','TRANSFERENCIA','QR','TARJETA')),
  monto          INTEGER NOT NULL CHECK (monto > 0),
  monto_recibido INTEGER NULL,
  vuelto         INTEGER NULL
);
CREATE INDEX idx_vp_venta ON venta_pago (venta_id);

CREATE TABLE movimiento_stock (
  id            TEXT NOT NULL PRIMARY KEY,
  producto_id   TEXT NOT NULL REFERENCES producto(id),
  lote_id       TEXT NULL REFERENCES lote(id),
  cantidad      INTEGER NOT NULL CHECK (cantidad <> 0),
  tipo          TEXT NOT NULL CHECK (tipo IN ('INICIAL','COMPRA','VENTA','ANULACION','AJUSTE')),
  referencia_id TEXT NULL,
  nota          TEXT NULL,
  fecha         TEXT NOT NULL,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_ms_producto ON movimiento_stock (producto_id);
CREATE INDEX idx_ms_lote ON movimiento_stock (lote_id);
CREATE INDEX idx_ms_referencia ON movimiento_stock (referencia_id);
CREATE INDEX idx_ms_fecha ON movimiento_stock (fecha);
CREATE INDEX idx_ms_created ON movimiento_stock (created_at);

CREATE TABLE gasto_caja (
  id         TEXT NOT NULL PRIMARY KEY,
  fecha      TEXT NOT NULL,
  monto      INTEGER NOT NULL CHECK (monto > 0),
  motivo     TEXT NOT NULL,
  categoria  TEXT NULL,
  metodo     TEXT NOT NULL DEFAULT 'EFECTIVO' CHECK (metodo IN ('EFECTIVO','TRANSFERENCIA','QR','TARJETA')),
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_gasto_fecha ON gasto_caja (fecha);

CREATE TABLE configuracion (
  clave TEXT NOT NULL PRIMARY KEY,
  valor TEXT NOT NULL
);

INSERT INTO configuracion (clave, valor) VALUES
  ('dias_alerta_vencimiento', '90'),
  ('zona_horaria', 'America/Argentina/Buenos_Aires');

CREATE VIEW v_stock_lote AS
SELECT l.id AS lote_id, l.producto_id, l.fecha_vencimiento, l.costo_unitario,
       SUM(m.cantidad) AS stock
FROM lote l
JOIN movimiento_stock m ON m.lote_id = l.id
GROUP BY l.id;

CREATE VIEW v_stock_producto AS
SELECT p.id AS producto_id, COALESCE(SUM(m.cantidad), 0) AS stock
FROM producto p
LEFT JOIN movimiento_stock m ON m.producto_id = p.id
WHERE p.tipo = 'SIMPLE'
GROUP BY p.id;

-- +goose Down
DROP VIEW IF EXISTS v_stock_producto;
DROP VIEW IF EXISTS v_stock_lote;
DROP TABLE IF EXISTS configuracion;
DROP TABLE IF EXISTS gasto_caja;
DROP TABLE IF EXISTS movimiento_stock;
DROP TABLE IF EXISTS venta_pago;
DROP TABLE IF EXISTS venta_item;
DROP TABLE IF EXISTS venta;
DROP TABLE IF EXISTS lote;
DROP TABLE IF EXISTS compra_item;
DROP TABLE IF EXISTS compra;
DROP TABLE IF EXISTS precio_historial;
DROP TABLE IF EXISTS combo_item;
DROP TABLE IF EXISTS producto_categoria;
DROP TABLE IF EXISTS producto;
DROP TABLE IF EXISTS categoria;
DROP TABLE IF EXISTS empresa;
