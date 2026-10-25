-- Base de datos normalizada para "STOCK VENCIMIENTOS"
-- Datos calculados (precio oferta, días restantes, estado) NO se guardan:
-- se obtienen desde la vista v_stock_vencimientos.

CREATE DATABASE IF NOT EXISTS stock_vencimientos
  CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
USE stock_vencimientos;

-- Categorías jerárquicas: "PERFUMERÍA > Perfumería femenina" => hijo con id_padre
CREATE TABLE categoria (
  id_categoria INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  nombre       VARCHAR(100) NOT NULL,
  id_padre     INT UNSIGNED NULL,
  CONSTRAINT fk_categoria_padre FOREIGN KEY (id_padre)
    REFERENCES categoria(id_categoria) ON DELETE RESTRICT,
  CONSTRAINT uq_categoria UNIQUE (nombre, id_padre)
);

CREATE TABLE producto (
  id_producto       INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  sku               VARCHAR(20)  NOT NULL UNIQUE,
  nombre            VARCHAR(255) NOT NULL,
  precio            DECIMAL(12,2) NOT NULL CHECK (precio >= 0),
  descripcion_html  TEXT NULL,
  peso_kg           DECIMAL(8,3) NULL,
  alto_cm           DECIMAL(8,2) NULL,
  ancho_cm          DECIMAL(8,2) NULL,
  profundidad_cm    DECIMAL(8,2) NULL,
  mostrar_en_tienda BOOLEAN NOT NULL DEFAULT TRUE
);

-- Un producto puede estar en varias categorías ("A | B | C" en el Excel)
CREATE TABLE producto_categoria (
  id_producto  INT UNSIGNED NOT NULL,
  id_categoria INT UNSIGNED NOT NULL,
  PRIMARY KEY (id_producto, id_categoria),
  CONSTRAINT fk_pc_producto  FOREIGN KEY (id_producto)  REFERENCES producto(id_producto)  ON DELETE CASCADE,
  CONSTRAINT fk_pc_categoria FOREIGN KEY (id_categoria) REFERENCES categoria(id_categoria) ON DELETE RESTRICT
);

-- Atributos opcionales (reemplaza las columnas "Nombre/Valor atributo N")
CREATE TABLE atributo (
  id_atributo INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  nombre      VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE producto_atributo (
  id_producto INT UNSIGNED NOT NULL,
  id_atributo INT UNSIGNED NOT NULL,
  valor       VARCHAR(255) NOT NULL,
  PRIMARY KEY (id_producto, id_atributo),
  CONSTRAINT fk_pa_producto FOREIGN KEY (id_producto) REFERENCES producto(id_producto) ON DELETE CASCADE,
  CONSTRAINT fk_pa_atributo FOREIGN KEY (id_atributo) REFERENCES atributo(id_atributo) ON DELETE RESTRICT
);

-- Stock por lote: permite tener el mismo producto con distintas fechas de vencimiento
CREATE TABLE lote (
  id_lote           INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  id_producto       INT UNSIGNED NOT NULL,
  cantidad          INT NOT NULL DEFAULT 0 CHECK (cantidad >= 0),
  fecha_vencimiento DATE NULL,
  CONSTRAINT fk_lote_producto FOREIGN KEY (id_producto) REFERENCES producto(id_producto) ON DELETE CASCADE,
  INDEX idx_lote_vencimiento (fecha_vencimiento)
);

-- Configuración de reglas que en el Excel estaban fijas en las fórmulas
CREATE TABLE configuracion (
  clave VARCHAR(50) PRIMARY KEY,
  valor DECIMAL(10,4) NOT NULL
);
INSERT INTO configuracion VALUES
  ('factor_precio_oferta', 0.60),   -- Precio oferta = Precio * 0.6
  ('dias_alerta_vencimiento', 90);  -- ESTADO = "POR VENCER" si faltan < 90 días

-- Vista que reproduce la hoja original con los campos calculados
CREATE VIEW v_stock_vencimientos AS
SELECT
  p.sku,
  p.nombre,
  l.cantidad AS stock,
  p.precio,
  ROUND(p.precio * (SELECT valor FROM configuracion WHERE clave = 'factor_precio_oferta'), 2) AS precio_oferta,
  l.fecha_vencimiento,
  DATEDIFF(l.fecha_vencimiento, CURDATE()) AS dias_restantes,
  CASE
    WHEN l.fecha_vencimiento IS NULL THEN NULL
    WHEN DATEDIFF(l.fecha_vencimiento, CURDATE()) < 0 THEN 'VENCIDO'
    WHEN DATEDIFF(l.fecha_vencimiento, CURDATE())
         < (SELECT valor FROM configuracion WHERE clave = 'dias_alerta_vencimiento') THEN 'POR VENCER'
    ELSE 'OK'
  END AS estado,
  (SELECT GROUP_CONCAT(
            CONCAT_WS(' > ', cp.nombre, c.nombre) ORDER BY c.id_categoria SEPARATOR ' | ')
     FROM producto_categoria pc
     JOIN categoria c ON c.id_categoria = pc.id_categoria
     LEFT JOIN categoria cp ON cp.id_categoria = c.id_padre
    WHERE pc.id_producto = p.id_producto) AS categorias
FROM producto p
LEFT JOIN lote l ON l.id_producto = p.id_producto;
