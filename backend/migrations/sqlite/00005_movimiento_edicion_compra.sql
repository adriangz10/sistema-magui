-- Movimiento de stock por edición de una compra (US-001b, ADR 0003).
-- SQLite no permite cambiar un CHECK: se reconstruye movimiento_stock. Nada la referencia por FK;
-- las dos vistas sí la leen, así que se borran y se recrean idénticas a 00001.

-- +goose Up
DROP VIEW IF EXISTS v_stock_producto;
DROP VIEW IF EXISTS v_stock_lote;

CREATE TABLE movimiento_stock_nueva (
  id            TEXT NOT NULL PRIMARY KEY,
  producto_id   TEXT NOT NULL REFERENCES producto(id),
  lote_id       TEXT NULL REFERENCES lote(id),
  cantidad      INTEGER NOT NULL CHECK (cantidad <> 0),
  tipo          TEXT NOT NULL CHECK (tipo IN ('INICIAL','COMPRA','VENTA','ANULACION','AJUSTE','EDICION_COMPRA')),
  referencia_id TEXT NULL,
  nota          TEXT NULL,
  fecha         TEXT NOT NULL,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
INSERT INTO movimiento_stock_nueva (id, producto_id, lote_id, cantidad, tipo, referencia_id, nota, fecha, created_at)
SELECT id, producto_id, lote_id, cantidad, tipo, referencia_id, nota, fecha, created_at FROM movimiento_stock;
DROP TABLE movimiento_stock;
ALTER TABLE movimiento_stock_nueva RENAME TO movimiento_stock;

CREATE INDEX idx_ms_producto ON movimiento_stock (producto_id);
CREATE INDEX idx_ms_lote ON movimiento_stock (lote_id);
CREATE INDEX idx_ms_referencia ON movimiento_stock (referencia_id);
CREATE INDEX idx_ms_fecha ON movimiento_stock (fecha);
CREATE INDEX idx_ms_created ON movimiento_stock (created_at);

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
-- Los movimientos EDICION_COMPRA pasan a AJUSTE marcando la nota (el stock no cambia).
UPDATE movimiento_stock
   SET tipo = 'AJUSTE', nota = substr('[EDICION_COMPRA] ' || COALESCE(nota, ''), 1, 255)
 WHERE tipo = 'EDICION_COMPRA';

DROP VIEW IF EXISTS v_stock_producto;
DROP VIEW IF EXISTS v_stock_lote;

CREATE TABLE movimiento_stock_nueva (
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
INSERT INTO movimiento_stock_nueva (id, producto_id, lote_id, cantidad, tipo, referencia_id, nota, fecha, created_at)
SELECT id, producto_id, lote_id, cantidad, tipo, referencia_id, nota, fecha, created_at FROM movimiento_stock;
DROP TABLE movimiento_stock;
ALTER TABLE movimiento_stock_nueva RENAME TO movimiento_stock;

CREATE INDEX idx_ms_producto ON movimiento_stock (producto_id);
CREATE INDEX idx_ms_lote ON movimiento_stock (lote_id);
CREATE INDEX idx_ms_referencia ON movimiento_stock (referencia_id);
CREATE INDEX idx_ms_fecha ON movimiento_stock (fecha);
CREATE INDEX idx_ms_created ON movimiento_stock (created_at);

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
