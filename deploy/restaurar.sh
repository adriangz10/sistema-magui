#!/usr/bin/env bash
# Restaura una copia de seguridad en el VPS. REEMPLAZA todos los datos actuales.
# Antes guarda una copia del estado actual ("antes de restaurar") por si hay que volver atrás.
#
#   deploy/restaurar.sh deploy/backups/magui-20261026-030000.sql.gz
#
# Se corre desde la raíz del repositorio, con el sistema levantado.
set -euo pipefail

archivo="${1:-}"
if [ -z "$archivo" ] || [ ! -f "$archivo" ]; then
	echo "Uso: deploy/restaurar.sh deploy/backups/magui-AAAAMMDD-HHMMSS.sql.gz" >&2
	echo "Copias disponibles:" >&2
	ls -1t deploy/backups/magui-*.sql.gz 2>/dev/null | head -n 10 >&2 || true
	exit 1
fi
gzip -t "$archivo" || {
	echo "El archivo está dañado o no es una copia válida." >&2
	exit 1
}

compose=(docker compose -f deploy/docker-compose.prod.yml --env-file deploy/.env.prod)

if [ "${CONFIRMAR:-}" != "RESTAURAR" ]; then
	echo "Esto reemplaza TODOS los datos actuales por los de: $(basename "$archivo")"
	read -r -p "Escribí RESTAURAR para seguir: " respuesta
	[ "$respuesta" = "RESTAURAR" ] || {
		echo "Cancelado."
		exit 1
	}
fi

echo "1/4 Copia del estado actual (por si hay que volver atrás)..."
"${compose[@]}" exec -T backup bash /backup.sh ahora

echo "2/4 Deteniendo la API..."
"${compose[@]}" stop api

echo "3/4 Restaurando $(basename "$archivo")..."
# shellcheck disable=SC2016 # las variables se expanden dentro del contenedor
"${compose[@]}" exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot -e "DROP DATABASE IF EXISTS \`$MYSQL_DATABASE\`; CREATE DATABASE \`$MYSQL_DATABASE\` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"'
gzip -dc "$archivo" | "${compose[@]}" exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot "$MYSQL_DATABASE"'

echo "4/4 Levantando la API..."
"${compose[@]}" start api
echo "Listo. Verificá el sistema en el navegador."
