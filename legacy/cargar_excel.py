"""Carga 'STOCK VENCIMIENTOS.xlsx' en la base normalizada stock_vencimientos.

Uso:  python cargar_excel.py <ruta_excel> <ruta_esquema.sql>
Credenciales por variables de entorno: MYSQL_USER, MYSQL_PWD, MYSQL_HOST (opcional).
"""
import os
import re
import sys

import openpyxl
import pymysql

excel_path, schema_path = sys.argv[1], sys.argv[2]


def limpiar(texto):
    """Quita espacios sobrantes (inicio, fin y dobles)."""
    if texto is None:
        return None
    texto = re.sub(r"\s+", " ", str(texto)).strip()
    return texto or None


wb = openpyxl.load_workbook(excel_path, data_only=True)
filas = [r for r in wb["Productos"].iter_rows(min_row=2, values_only=True)
         if any(v is not None for v in r)]
# "Hoja 1" tiene los nombres de los kits que faltan en "Productos" (misma fila)
nombres_hoja1 = [r[0] for r in wb["Hoja 1"].iter_rows(min_row=2, values_only=True)]

conn = pymysql.connect(
    host=os.environ.get("MYSQL_HOST", "localhost"),
    user=os.environ["MYSQL_USER"],
    password=os.environ["MYSQL_PWD"],
    charset="utf8mb4",
    autocommit=False,
)
cur = conn.cursor()

# 1) Crear esquema (desde cero)
cur.execute("DROP DATABASE IF EXISTS stock_vencimientos")
with open(schema_path, encoding="utf-8") as f:
    sql = "\n".join(l for l in f.read().splitlines() if not l.strip().startswith("--"))
for stmt in sql.split(";"):
    if stmt.strip():
        cur.execute(stmt)

# 2) Categorías
categorias = {}  # (nombre, id_padre) -> id


def id_categoria(nombre, id_padre):
    clave = (nombre.upper(), id_padre)
    if clave not in categorias:
        cur.execute("INSERT INTO categoria (nombre, id_padre) VALUES (%s, %s)", (nombre, id_padre))
        categorias[clave] = cur.lastrowid
    return categorias[clave]


cur.execute("INSERT INTO atributo (nombre) VALUES ('Camino olfativo')")
id_aroma = cur.lastrowid

for i, r in enumerate(filas):
    nombre = limpiar(r[0]) or limpiar(nombres_hoja1[i])
    stock, sku, precio = r[1], limpiar(r[2]), r[3]
    vencimiento = r[5].date() if r[5] else None
    cats_txt, peso, alto, ancho, prof, mostrar, desc, aroma = r[11:19]

    cur.execute(
        """INSERT INTO producto (sku, nombre, precio, descripcion_html, peso_kg, alto_cm,
                                 ancho_cm, profundidad_cm, mostrar_en_tienda)
           VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)""",
        (sku, nombre, precio, desc, peso, alto, ancho, prof,
         str(mostrar).strip().lower() in ("si", "sí")),
    )
    id_prod = cur.lastrowid

    cur.execute(
        "INSERT INTO lote (id_producto, cantidad, fecha_vencimiento) VALUES (%s,%s,%s)",
        (id_prod, int(stock or 0), vencimiento),
    )

    ids = set()
    for ruta in (cats_txt or "").split("|"):
        partes = [limpiar(p) for p in ruta.split(">") if limpiar(p)]
        padre = None
        for parte in partes:
            padre = id_categoria(parte, padre)
        if padre:
            ids.add(padre)
    for id_cat in ids:
        cur.execute("INSERT INTO producto_categoria VALUES (%s,%s)", (id_prod, id_cat))

    if limpiar(aroma):
        cur.execute("INSERT INTO producto_atributo VALUES (%s,%s,%s)", (id_prod, id_aroma, limpiar(aroma)))

conn.commit()
print(f"Productos cargados: {len(filas)} | Categorías: {len(categorias)}")
conn.close()
