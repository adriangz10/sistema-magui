-- Nombre, logo y tickets del negocio (US-003). Una sola fila (id = 1).

-- +goose Up
CREATE TABLE negocio (
  id                INTEGER NOT NULL PRIMARY KEY CHECK (id = 1),
  nombre            TEXT NOT NULL,
  datos_ticket      TEXT NULL,
  pie_ticket        TEXT NULL,
  ticket_habilitado INTEGER NOT NULL DEFAULT 0,
  ticket_al_cobrar  INTEGER NOT NULL DEFAULT 0,
  ticket_ancho      INTEGER NOT NULL DEFAULT 80 CHECK (ticket_ancho IN (58, 80)),
  ticket_logo       INTEGER NOT NULL DEFAULT 1,
  logo              BLOB NULL,
  logo_tipo         TEXT NULL,
  logo_version      INTEGER NOT NULL DEFAULT 0,
  updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
INSERT INTO negocio (id, nombre, pie_ticket) VALUES (1, 'Magui', '¡Gracias por su compra!');

-- +goose Down
DROP TABLE IF EXISTS negocio;
