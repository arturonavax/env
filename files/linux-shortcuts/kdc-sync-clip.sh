#!/usr/bin/env bash
# ~/.local/bin/.
set -uo pipefail

DEVICE_ID="<DEVICE_ID>"

notify_err() {
    notify-send -u critical -a "KDE Connect" "Error de Envío" "$1"
    exit 1
}

notify_warn() {
    notify-send -u normal -a "KDE Connect" "Envío en Tránsito" "$1"
}

notify_ok() {
    notify-send -u low -a "KDE Connect" "$1" "$2"
}

[[ -z "$DEVICE_ID" || "$DEVICE_ID" == "<DEVICE_ID>" ]] && notify_err "No se especificó un DEVICE_ID válido."

# 1. Comprobación instantánea de disponibilidad vía D-Bus (~6ms)
RAW_REACHABLE=$(gdbus call --session \
    --dest org.kde.kdeconnect \
    --object-path "/modules/kdeconnect/devices/$DEVICE_ID" \
    --method org.freedesktop.DBus.Properties.Get "org.kde.kdeconnect.device" "isReachable" 2>/dev/null || echo "false")

if [[ "$RAW_REACHABLE" != *"true"* ]]; then
    notify_err "El teléfono no está disponible en la red local."
fi

# 2. Extracción limpia del nombre sin subprocesos ni regex rotos
RAW_NAME=$(gdbus call --session \
    --dest org.kde.kdeconnect \
    --object-path "/modules/kdeconnect/devices/$DEVICE_ID" \
    --method org.freedesktop.DBus.Properties.Get "org.kde.kdeconnect.device" "name" 2>/dev/null || echo "")

DEV_NAME="${RAW_NAME#*\'}"
DEV_NAME="${DEV_NAME%\'*}"
[ -z "$DEV_NAME" ] || [ "$DEV_NAME" = "$RAW_NAME" ] && DEV_NAME="Teléfono"

# 3. Detección de contenido en portapapeles
TARGETS=$(xclip -selection clipboard -target TARGETS -out 2>/dev/null || true)
[ -z "$TARGETS" ] && notify_err "El portapapeles está vacío."

# 4. Caso A: Archivos desde el explorador
if [[ "$TARGETS" == *"text/uri-list"* ]]; then
    mapfile -t VALID_FILES < <(xclip -selection clipboard -target text/uri-list -out 2>/dev/null | python3 -c '
import sys, urllib.parse
for line in sys.stdin:
    line = line.strip()
    if line.startswith("file://"):
        path = urllib.parse.unquote(urllib.parse.urlsplit(line).path)
        print(path)
' | while IFS= read -r file; do [ -e "$file" ] && echo "$file"; done)

    if [ ${#VALID_FILES[@]} -gt 0 ]; then
        if ERR_MSG=$(kdeconnect-cli --device "$DEVICE_ID" --share "${VALID_FILES[@]}" 2>&1); then
            notify_warn "Transfiriendo ${#VALID_FILES[@]} archivo(s) a $DEV_NAME..."
            exit 0
        else
            notify_err "Fallo al enviar archivos a $DEV_NAME:\n$ERR_MSG"
        fi
    fi
fi

# 5. Caso B: Imágenes en RAM (Screenshots o navegador)
if [[ "$TARGETS" == *"image/png"* ]] || [[ "$TARGETS" == *"image/jpeg"* ]] || [[ "$TARGETS" == *"image/bmp"* ]] || [[ "$TARGETS" == *"image/webp"* ]]; then
    TEMP_IMG="/tmp/Screenshot_$(date +%Y%m%d_%H%M%S).png"
    if xclip -selection clipboard -target image/png -out >"$TEMP_IMG" 2>/dev/null && [ -s "$TEMP_IMG" ]; then
        if ERR_MSG=$(kdeconnect-cli --device "$DEVICE_ID" --share "$TEMP_IMG" 2>&1); then
            notify_warn "Imagen en cola de envío a $DEV_NAME."
            (sleep 30 && rm -f "$TEMP_IMG") >/dev/null 2>&1 &
            disown
            exit 0
        else
            rm -f "$TEMP_IMG"
            notify_err "Error al enviar imagen a $DEV_NAME:\n$ERR_MSG"
        fi
    fi
    rm -f "$TEMP_IMG"
fi

# 6. Caso C: Texto plano o enlaces (disparo D-Bus directo sin esperas)
if gdbus call --session \
    --dest org.kde.kdeconnect \
    --object-path "/modules/kdeconnect/devices/$DEVICE_ID/clipboard" \
    --method org.kde.kdeconnect.device.clipboard.sendClipboard >/dev/null 2>&1; then
    notify_ok "Portapapeles Sincronizado" "Texto enviado a $DEV_NAME."
    exit 0
else
    notify_err "Fallo D-Bus al sincronizar con $DEV_NAME."
fi
