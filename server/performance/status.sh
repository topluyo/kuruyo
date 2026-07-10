#!/usr/bin/env bash

INTERVAL=1

if command -v nethogs >/dev/null 2>&1; then
    HAS_NETHOGS=1
else
    HAS_NETHOGS=0
fi

while true; do
    screen=$(
        printf "KURUYO STATUS MONITOR - %s\n\n" "$(date '+%d.%m.%Y %H:%M:%S')"

        printf "%-35s %-8s %-10s %-8s %-15s %-15s\n" \
            "SERVICE" "PID" "RAM(MB)" "CPU%" "RX(KB/s)" "TX(KB/s)"

        for servicefile in /etc/systemd/system/kuruyo-*; do
            [ -e "$servicefile" ] || continue

            service=$(basename "$servicefile")

            pid=$(systemctl show -p MainPID --value "$service" 2>/dev/null)

            if [[ -z "$pid" || "$pid" == "0" ]]; then
                printf "%-35s %-8s\n" "$service" "STOPPED"
                continue
            fi

            read rss cpu <<<"$(ps -p "$pid" -o rss=,%cpu=)"

            ram=$(awk "BEGIN {printf \"%.1f\", $rss/1024}")

            rx="-"
            tx="-"

            if [[ $HAS_NETHOGS -eq 1 ]]; then
                line=$(timeout 1 nethogs -t -c 1 2>/dev/null | awk -v pid="$pid" '$1 ~ "/" pid "/" {print}')

                if [[ -n "$line" ]]; then
                    rx=$(awk '{printf "%.1f", $2}' <<<"$line")
                    tx=$(awk '{printf "%.1f", $3}' <<<"$line")
                fi
            fi

            printf "%-35s %-8s %-10s %-8s %-15s %-15s\n" \
                "$service" "$pid" "$ram" "$cpu" "$rx" "$tx"
        done
    )

    # Ekranı titreşimsiz güncelle
    printf '\033[H\033[2J'
    printf '%s' "$screen"
    printf '\033[J'

    sleep "$INTERVAL"
done
