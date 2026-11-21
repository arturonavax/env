#!/usr/bin/env bash
# ~/.local/bin/.
set -uo pipefail

TIMEOUT_SECS=8
POLL_INTERVAL=0.5

notify_low() {
    notify-send -u low -a "KDE Connect" "$1" "$2"
}

notify_normal() {
    notify-send -u normal -a "KDE Connect" "$1" "$2"
}

notify_crit() {
    notify-send -u critical -a "KDE Connect" "$1" "$2"
}

# 1. Obtener lista de todos los pares configurados (~5ms)
ALL_RAW=$(gdbus call --session \
    --dest org.kde.kdeconnect \
    --object-path /modules/kdeconnect \
    --method org.kde.kdeconnect.daemon.devices false false 2>/dev/null || echo "()")

mapfile -t ALL_IDS < <(echo "$ALL_RAW" | grep -oE "'[a-zA-Z0-9_-]+'" | tr -d "'")

if [ ${#ALL_IDS[@]} -eq 0 ]; then
    notify_low "KDE Connect" "No hay dispositivos emparejados."
    exit 0
fi

# 2. Obtener en una sola llamada los IDs que ya están activos (~5ms)
ONLINE_RAW=$(gdbus call --session \
    --dest org.kde.kdeconnect \
    --object-path /modules/kdeconnect \
    --method org.kde.kdeconnect.daemon.devices true false 2>/dev/null || echo "()")

declare -A DEV_NAMES
declare -A PENDING_DEVS

# 3. Clasificación inmediata y resolución limpia de nombres
for id in "${ALL_IDS[@]}"; do
    raw_name=$(gdbus call --session \
        --dest org.kde.kdeconnect \
        --object-path "/modules/kdeconnect/devices/$id" \
        --method org.freedesktop.DBus.Properties.Get "org.kde.kdeconnect.device" "name" 2>/dev/null || echo "")

    clean_name="${raw_name#*\'}"
    clean_name="${clean_name%\'*}"
    [ -z "$clean_name" ] || [ "$clean_name" = "$raw_name" ] && clean_name="Dispositivo"
    DEV_NAMES["$id"]="$clean_name"

    # Comparación en memoria
    if [[ "$ONLINE_RAW" == *"'$id'"* ]]; then
        notify_low "Dispositivo Conectado" "$clean_name ya está sincronizado."
    else
        PENDING_DEVS["$id"]="$clean_name"
        notify_normal "Buscando Enlace..." "Intentando conectar con $clean_name..."
    fi
done

# Si todos estaban online, termina de inmediato sin tocar la red (<25ms)
[ ${#PENDING_DEVS[@]} -eq 0 ] && exit 0

# 4. Emitir broadcast UDP desacoplado
kdeconnect-cli --refresh >/dev/null 2>&1 &
gdbus call --session \
    --dest org.kde.kdeconnect \
    --object-path /modules/kdeconnect \
    --method org.kde.kdeconnect.daemon.forceOnNetworkChange >/dev/null 2>&1 || true

# 5. Sondeo de alta frecuencia (1 llamada D-Bus por ciclo de 500ms)
TICKS=0
MAX_TICKS=$((TIMEOUT_SECS * 2))

while [ "$TICKS" -lt "$MAX_TICKS" ] && [ "${#PENDING_DEVS[@]}" -gt 0 ]; do
    sleep "$POLL_INTERVAL"
    TICKS=$((TICKS + 1))

    CURRENT_ONLINE=$(gdbus call --session \
        --dest org.kde.kdeconnect \
        --object-path /modules/kdeconnect \
        --method org.kde.kdeconnect.daemon.devices true false 2>/dev/null || echo "()")

    for id in "${!PENDING_DEVS[@]}"; do
        if [[ "$CURRENT_ONLINE" == *"'$id'"* ]]; then
            notify_normal "Conexión Reestablecida" "${DEV_NAMES[$id]} conectado exitosamente."
            unset "PENDING_DEVS[$id]"
        fi
    done
done

# 6. Timeout exclusivamente para los que no respondieron
for id in "${!PENDING_DEVS[@]}"; do
    notify_crit "Fallo de Reconexión" "${DEV_NAMES[$id]} no respondió tras ${TIMEOUT_SECS}s."
done
