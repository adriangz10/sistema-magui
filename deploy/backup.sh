#!/usr/bin/env bash
# Copias de seguridad de MySQL. Corre dentro del servicio "backup" de docker-compose.prod.yml.
#
#   backup.sh            queda corriendo: una copia por día a la hora BACKUP_HORA (hora de Argentina)
#   backup.sh ahora      hace una copia ya y termina (ej. antes de actualizar)
#
# Cada copia es /backups/magui-AAAAMMDD-HHMMSS.sql.gz. Se guardan las de los últimos BACKUP_DIAS días.
# Variables: DB_HOST, DB_NAME, DB_USER, MYSQL_PWD (contraseña), BACKUP_HORA (03:00), BACKUP_DIAS (14).
set -euo pipefail

DESTINO=/backups
HORA="${BACKUP_HORA:-03:00}"
DIAS="${BACKUP_DIAS:-14}"
export TZ="${TZ:-America/Argentina/Buenos_Aires}"

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') backup: $*"; }

copiar() {
	local nombre tmp
	nombre="$DESTINO/magui-$(date +%Y%m%d-%H%M%S).sql.gz"
	tmp="$nombre.parcial"
	# --single-transaction: copia consistente sin frenar las ventas (InnoDB).
	if ! mysqldump --host="$DB_HOST" --user="$DB_USER" --single-transaction --quick --routines --triggers \
		--no-tablespaces --set-gtid-purged=OFF --default-character-set=utf8mb4 "$DB_NAME" | gzip -9 >"$tmp"; then
		rm -f "$tmp"
		log "ERROR: falló la copia"
		return 1
	fi
	# Una copia cortada a la mitad no sirve: se verifica antes de darla por buena.
	if ! gzip -t "$tmp" || ! gzip -dc "$tmp" | tail -n 1 | grep -q 'Dump completed'; then
		rm -f "$tmp"
		log "ERROR: la copia quedó incompleta"
		return 1
	fi
	mv "$tmp" "$nombre"
	log "copia lista: $(basename "$nombre") ($(du -h "$nombre" | cut -f1))"
	# Rotación: se borran las copias más viejas que BACKUP_DIAS días.
	find "$DESTINO" -maxdepth 1 -name 'magui-*.sql.gz' -mtime +"$((DIAS - 1))" -print -delete | sed 's/^/backup: borrada copia vieja /'
}

# Segundos hasta la próxima BACKUP_HORA.
espera() {
	local ahora prox
	ahora=$(date +%s)
	prox=$(date -d "today $HORA" +%s)
	if [ "$prox" -le "$ahora" ]; then
		prox=$(date -d "tomorrow $HORA" +%s)
	fi
	echo $((prox - ahora))
}

mkdir -p "$DESTINO"
if [ "${1:-}" = "ahora" ]; then
	copiar
	exit
fi

log "copias diarias a las $HORA (hora de Argentina), se guardan $DIAS días"
# Al arrancar, si no hay ninguna copia de las últimas 24 horas, se hace una enseguida.
if [ -z "$(find "$DESTINO" -maxdepth 1 -name 'magui-*.sql.gz' -mmin -1440 -print -quit)" ]; then
	copiar || true
fi
while true; do
	sleep "$(espera)"
	copiar || true # si falla se vuelve a intentar mañana; el chequeo de salud lo marca
done
