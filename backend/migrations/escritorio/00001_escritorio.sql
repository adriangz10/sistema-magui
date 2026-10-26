-- Base local de la app de escritorio (SQLite). No es una copia del esquema del servidor:
-- guarda solo lo necesario para vender sin conexión y la cola de ventas por subir.
-- Montos en centavos (INTEGER); fechas en texto RFC 3339 (UTC).

-- +goose Up

-- Copia del catálogo que se descarga del servidor (se reemplaza entera en cada sincronización).
CREATE TABLE copia_producto (
    id            TEXT    PRIMARY KEY,
    sku           TEXT    NOT NULL,
    codigo_barras TEXT,
    nombre        TEXT    NOT NULL,
    tipo          TEXT    NOT NULL CHECK (tipo IN ('SIMPLE', 'COMBO')),
    empresa       TEXT,
    precio_venta  INTEGER NOT NULL,
    stock         INTEGER NOT NULL, -- stock en el servidor al generar la copia
    activo        INTEGER NOT NULL,
    version       INTEGER NOT NULL
);
CREATE INDEX copia_producto_sku ON copia_producto (sku);
CREATE INDEX copia_producto_codigo ON copia_producto (codigo_barras);

CREATE TABLE copia_componente (
    combo_id    TEXT    NOT NULL,
    producto_id TEXT    NOT NULL,
    cantidad    INTEGER NOT NULL,
    PRIMARY KEY (combo_id, producto_id)
);

-- Ventas cobradas en el escritorio. PENDIENTE: falta subirla; ENVIADA: ya está en el servidor;
-- RECHAZADA: el servidor no la aceptó (hay que revisarla).
CREATE TABLE venta_local (
    id         TEXT    PRIMARY KEY,
    fecha      TEXT    NOT NULL,
    total      INTEGER NOT NULL,
    envio      TEXT    NOT NULL, -- JSON listo para POST /api/sync/ventas
    detalle    TEXT    NOT NULL, -- JSON de la venta como la muestra la pantalla
    estado     TEXT    NOT NULL DEFAULT 'PENDIENTE' CHECK (estado IN ('PENDIENTE', 'ENVIADA', 'RECHAZADA')),
    error      TEXT,
    -- en_copia: la copia del catálogo ya refleja esta venta (se bajó después de subirla).
    en_copia   INTEGER NOT NULL DEFAULT 0,
    creada_en  TEXT    NOT NULL,
    enviada_en TEXT
);
CREATE INDEX venta_local_estado ON venta_local (estado, fecha);

-- Unidades que salieron por cada venta local (en combos, de sus componentes).
-- Mientras la copia no las refleje, se restan del stock de la copia.
CREATE TABLE salida_local (
    venta_id    TEXT    NOT NULL,
    producto_id TEXT    NOT NULL,
    cantidad    INTEGER NOT NULL
);
CREATE INDEX salida_local_venta ON salida_local (venta_id);
CREATE INDEX salida_local_producto ON salida_local (producto_id);

-- Datos sueltos: sesión con el servidor, contraseña para ingresar sin conexión, fecha de la copia.
CREATE TABLE ajuste (
    clave TEXT PRIMARY KEY,
    valor TEXT NOT NULL
);

-- +goose Down
DROP TABLE ajuste;
DROP TABLE salida_local;
DROP TABLE venta_local;
DROP TABLE copia_componente;
DROP TABLE copia_producto;
