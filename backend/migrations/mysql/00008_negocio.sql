-- Nombre, logo y tickets del negocio (US-003). Una sola fila (id = 1).

-- +goose Up
CREATE TABLE negocio (
  id                TINYINT NOT NULL PRIMARY KEY,
  nombre            VARCHAR(60)  NOT NULL,
  datos_ticket      VARCHAR(300) NULL COMMENT 'Dirección, teléfono, CUIT... (una por línea)',
  pie_ticket        VARCHAR(200) NULL,
  ticket_habilitado BOOLEAN NOT NULL DEFAULT FALSE,
  ticket_al_cobrar  BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Imprimir apenas se cobra',
  ticket_ancho      TINYINT NOT NULL DEFAULT 80 COMMENT 'Ancho del papel en mm (58 u 80)',
  ticket_logo       BOOLEAN NOT NULL DEFAULT TRUE,
  logo              MEDIUMBLOB NULL,
  logo_tipo         VARCHAR(30) NULL,
  logo_version      INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Cambia con cada logo nuevo (evita la caché)',
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  CONSTRAINT ck_negocio_unico CHECK (id = 1),
  CONSTRAINT ck_negocio_ancho CHECK (ticket_ancho IN (58, 80))
);
INSERT INTO negocio (id, nombre, pie_ticket) VALUES (1, 'Magui', '¡Gracias por su compra!');

-- +goose Down
DROP TABLE IF EXISTS negocio;
