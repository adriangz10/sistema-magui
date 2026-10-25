-- Esquema v2 del Sistema Magui (MySQL, servidor central).
-- IDs: UUID v7 generados por la aplicación (CHAR(36)), para poder crear registros offline.
-- Dinero: DECIMAL(12,2). Fechas: DATETIME(3) en UTC.
-- Stock: no se guarda como número; es la suma de movimiento_stock.

-- +goose Up

CREATE TABLE empresa (
  id          CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  nombre      VARCHAR(100) NOT NULL,
  activo      BOOLEAN NOT NULL DEFAULT TRUE,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  version     INT UNSIGNED NOT NULL DEFAULT 1,
  CONSTRAINT uq_empresa_nombre UNIQUE (nombre)
);

CREATE TABLE categoria (
  id          CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  nombre      VARCHAR(100) NOT NULL,
  padre_id    CHAR(36) CHARACTER SET ascii NULL,
  -- Columna auxiliar para que el UNIQUE funcione también con categorías raíz (padre NULL)
  padre_clave CHAR(36) CHARACTER SET ascii AS (COALESCE(padre_id, '')) STORED,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  version     INT UNSIGNED NOT NULL DEFAULT 1,
  CONSTRAINT fk_categoria_padre FOREIGN KEY (padre_id) REFERENCES categoria(id),
  CONSTRAINT uq_categoria_nombre UNIQUE (padre_clave, nombre),
  CONSTRAINT ck_categoria_no_ciclo CHECK (padre_id IS NULL OR padre_id <> id)
);

CREATE TABLE producto (
  id                CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  sku               VARCHAR(40) NOT NULL,
  codigo_barras     VARCHAR(64) NULL,
  nombre            VARCHAR(255) NOT NULL,
  descripcion       TEXT NULL,
  empresa_id        CHAR(36) CHARACTER SET ascii NULL,
  tipo              ENUM('SIMPLE','COMBO') NOT NULL DEFAULT 'SIMPLE',
  precio_venta      DECIMAL(12,2) NOT NULL,
  costo             DECIMAL(12,2) NULL COMMENT 'Último costo de compra conocido',
  stock_minimo      INT NOT NULL DEFAULT 0,
  peso_kg           DECIMAL(8,3) NULL,
  alto_cm           DECIMAL(8,2) NULL,
  ancho_cm          DECIMAL(8,2) NULL,
  profundidad_cm    DECIMAL(8,2) NULL,
  mostrar_en_tienda BOOLEAN NOT NULL DEFAULT TRUE,
  activo            BOOLEAN NOT NULL DEFAULT TRUE,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  version           INT UNSIGNED NOT NULL DEFAULT 1,
  CONSTRAINT uq_producto_sku UNIQUE (sku),
  CONSTRAINT uq_producto_codigo_barras UNIQUE (codigo_barras),
  CONSTRAINT fk_producto_empresa FOREIGN KEY (empresa_id) REFERENCES empresa(id),
  CONSTRAINT ck_producto_precio CHECK (precio_venta >= 0),
  CONSTRAINT ck_producto_costo CHECK (costo IS NULL OR costo >= 0),
  CONSTRAINT ck_producto_stock_minimo CHECK (stock_minimo >= 0),
  INDEX idx_producto_nombre (nombre),
  INDEX idx_producto_updated (updated_at)
);

CREATE TABLE producto_categoria (
  producto_id  CHAR(36) CHARACTER SET ascii NOT NULL,
  categoria_id CHAR(36) CHARACTER SET ascii NOT NULL,
  PRIMARY KEY (producto_id, categoria_id),
  CONSTRAINT fk_pc_producto  FOREIGN KEY (producto_id)  REFERENCES producto(id) ON DELETE CASCADE,
  CONSTRAINT fk_pc_categoria FOREIGN KEY (categoria_id) REFERENCES categoria(id),
  INDEX idx_pc_categoria (categoria_id)
);

-- Componentes de un combo (producto tipo COMBO)
CREATE TABLE combo_item (
  combo_id    CHAR(36) CHARACTER SET ascii NOT NULL,
  producto_id CHAR(36) CHARACTER SET ascii NOT NULL,
  cantidad    INT NOT NULL,
  PRIMARY KEY (combo_id, producto_id),
  CONSTRAINT fk_combo_combo    FOREIGN KEY (combo_id)    REFERENCES producto(id) ON DELETE CASCADE,
  CONSTRAINT fk_combo_producto FOREIGN KEY (producto_id) REFERENCES producto(id),
  -- "Un combo no se contiene a sí mismo" se valida en el dominio:
  -- MySQL no permite CHECK sobre columnas con ON DELETE CASCADE.
  CONSTRAINT ck_combo_cantidad CHECK (cantidad > 0)
);

CREATE TABLE precio_historial (
  id              CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  producto_id     CHAR(36) CHARACTER SET ascii NOT NULL,
  precio_anterior DECIMAL(12,2) NOT NULL,
  precio_nuevo    DECIMAL(12,2) NOT NULL,
  origen          ENUM('MANUAL','CASCADA','COMPRA') NOT NULL,
  motivo          VARCHAR(255) NULL,
  fecha           DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_ph_producto FOREIGN KEY (producto_id) REFERENCES producto(id) ON DELETE CASCADE,
  INDEX idx_ph_producto_fecha (producto_id, fecha)
);

CREATE TABLE compra (
  id         CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  fecha      DATETIME(3) NOT NULL,
  empresa_id CHAR(36) CHARACTER SET ascii NULL,
  total      DECIMAL(12,2) NOT NULL,
  notas      VARCHAR(500) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_compra_empresa FOREIGN KEY (empresa_id) REFERENCES empresa(id),
  CONSTRAINT ck_compra_total CHECK (total >= 0),
  INDEX idx_compra_fecha (fecha)
);

CREATE TABLE compra_item (
  id                 CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  compra_id          CHAR(36) CHARACTER SET ascii NOT NULL,
  producto_id        CHAR(36) CHARACTER SET ascii NOT NULL,
  cantidad           INT NOT NULL,
  costo_unitario     DECIMAL(12,2) NOT NULL,
  precio_venta_nuevo DECIMAL(12,2) NULL,
  fecha_vencimiento  DATE NULL,
  CONSTRAINT fk_ci_compra   FOREIGN KEY (compra_id)   REFERENCES compra(id) ON DELETE CASCADE,
  CONSTRAINT fk_ci_producto FOREIGN KEY (producto_id) REFERENCES producto(id),
  CONSTRAINT ck_ci_cantidad CHECK (cantidad > 0),
  CONSTRAINT ck_ci_costo CHECK (costo_unitario >= 0),
  INDEX idx_ci_producto (producto_id)
);

-- Lote: unidades de un producto que comparten vencimiento y costo
CREATE TABLE lote (
  id                CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  producto_id       CHAR(36) CHARACTER SET ascii NOT NULL,
  fecha_vencimiento DATE NULL,
  costo_unitario    DECIMAL(12,2) NULL,
  compra_item_id    CHAR(36) CHARACTER SET ascii NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_lote_producto    FOREIGN KEY (producto_id)    REFERENCES producto(id),
  CONSTRAINT fk_lote_compra_item FOREIGN KEY (compra_item_id) REFERENCES compra_item(id),
  INDEX idx_lote_producto_venc (producto_id, fecha_vencimiento),
  INDEX idx_lote_vencimiento (fecha_vencimiento)
);

CREATE TABLE venta (
  id               CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  fecha            DATETIME(3) NOT NULL,
  total            DECIMAL(12,2) NOT NULL,
  costo_total      DECIMAL(12,2) NOT NULL DEFAULT 0,
  estado           ENUM('CONFIRMADA','ANULADA') NOT NULL DEFAULT 'CONFIRMADA',
  origen           ENUM('WEB','MOVIL','DESKTOP') NOT NULL,
  anulada_en       DATETIME(3) NULL,
  motivo_anulacion VARCHAR(255) NULL,
  created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT ck_venta_total CHECK (total >= 0),
  INDEX idx_venta_fecha (fecha),
  INDEX idx_venta_estado_fecha (estado, fecha)
);

CREATE TABLE venta_item (
  id              CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  venta_id        CHAR(36) CHARACTER SET ascii NOT NULL,
  producto_id     CHAR(36) CHARACTER SET ascii NOT NULL,
  cantidad        INT NOT NULL,
  precio_unitario DECIMAL(12,2) NOT NULL,
  costo_unitario  DECIMAL(12,2) NULL COMMENT 'Costo al momento de vender (para el margen)',
  subtotal        DECIMAL(12,2) NOT NULL,
  CONSTRAINT fk_vi_venta    FOREIGN KEY (venta_id)    REFERENCES venta(id) ON DELETE CASCADE,
  CONSTRAINT fk_vi_producto FOREIGN KEY (producto_id) REFERENCES producto(id),
  CONSTRAINT ck_vi_cantidad CHECK (cantidad > 0),
  INDEX idx_vi_producto (producto_id)
);

CREATE TABLE venta_pago (
  id             CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  venta_id       CHAR(36) CHARACTER SET ascii NOT NULL,
  metodo         ENUM('EFECTIVO','TRANSFERENCIA','QR','TARJETA') NOT NULL,
  monto          DECIMAL(12,2) NOT NULL COMMENT 'Monto aplicado a la venta',
  monto_recibido DECIMAL(12,2) NULL COMMENT 'Solo efectivo: lo que entregó el cliente',
  vuelto         DECIMAL(12,2) NULL,
  CONSTRAINT fk_vp_venta FOREIGN KEY (venta_id) REFERENCES venta(id) ON DELETE CASCADE,
  CONSTRAINT ck_vp_monto CHECK (monto > 0),
  INDEX idx_vp_venta (venta_id)
);

-- Cada cambio de stock. Stock actual = SUM(cantidad).
CREATE TABLE movimiento_stock (
  id            CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  producto_id   CHAR(36) CHARACTER SET ascii NOT NULL,
  lote_id       CHAR(36) CHARACTER SET ascii NULL,
  cantidad      INT NOT NULL COMMENT 'Positivo entra, negativo sale',
  tipo          ENUM('INICIAL','COMPRA','VENTA','ANULACION','AJUSTE') NOT NULL,
  referencia_id CHAR(36) CHARACTER SET ascii NULL COMMENT 'Venta, compra, etc. que lo originó',
  nota          VARCHAR(255) NULL,
  fecha         DATETIME(3) NOT NULL,
  created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT fk_ms_producto FOREIGN KEY (producto_id) REFERENCES producto(id),
  CONSTRAINT fk_ms_lote     FOREIGN KEY (lote_id)     REFERENCES lote(id),
  CONSTRAINT ck_ms_cantidad CHECK (cantidad <> 0),
  INDEX idx_ms_producto (producto_id),
  INDEX idx_ms_lote (lote_id),
  INDEX idx_ms_referencia (referencia_id),
  INDEX idx_ms_fecha (fecha),
  INDEX idx_ms_created (created_at)
);

-- Retiros de caja para gastos que no son mercadería
CREATE TABLE gasto_caja (
  id         CHAR(36) CHARACTER SET ascii NOT NULL PRIMARY KEY,
  fecha      DATETIME(3) NOT NULL,
  monto      DECIMAL(12,2) NOT NULL,
  motivo     VARCHAR(255) NOT NULL,
  categoria  VARCHAR(60) NULL COMMENT 'Ej.: servicios, transporte, retiro personal',
  metodo     ENUM('EFECTIVO','TRANSFERENCIA','QR','TARJETA') NOT NULL DEFAULT 'EFECTIVO',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT ck_gasto_monto CHECK (monto > 0),
  INDEX idx_gasto_fecha (fecha)
);

CREATE TABLE configuracion (
  clave VARCHAR(60) NOT NULL PRIMARY KEY,
  valor VARCHAR(255) NOT NULL
);

INSERT INTO configuracion (clave, valor) VALUES
  ('dias_alerta_vencimiento', '90'),
  ('zona_horaria', 'America/Argentina/Buenos_Aires');

-- Stock por lote (solo lotes con movimientos)
CREATE VIEW v_stock_lote AS
SELECT l.id AS lote_id, l.producto_id, l.fecha_vencimiento, l.costo_unitario,
       SUM(m.cantidad) AS stock
FROM lote l
JOIN movimiento_stock m ON m.lote_id = l.id
GROUP BY l.id, l.producto_id, l.fecha_vencimiento, l.costo_unitario;

-- Stock total por producto (los combos no tienen stock propio)
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
