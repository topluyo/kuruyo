#!/bin/bash

set -e

ROOT="/web/server/system/erlik"
cd "$ROOT"

echo "[.] XDP programlarını interface'lerden kaldırıyor..."

# ------------------------------------------------------------
# 1. Önce klasik XDP attach'lerini kaldır
# ------------------------------------------------------------

for iface in $(ip -o link show | awk -F': ' '{print $2}' | cut -d@ -f1); do
    echo "Removing XDP from: $iface"

    # Normal XDP
    ip link set dev "$iface" xdp off 2>/dev/null || true

    # Generic XDP
    ip link set dev "$iface" xdpgeneric off 2>/dev/null || true

    # Driver/native XDP
    ip link set dev "$iface" xdpdrv off 2>/dev/null || true

    # Hardware offload XDP
    ip link set dev "$iface" xdpoffload off 2>/dev/null || true
done

# ------------------------------------------------------------
# 2. BPF link üzerinden attach edilmiş XDP'leri kaldır
# ------------------------------------------------------------

echo
echo "[.] BPF XDP linkleri kontrol ediliyor..."

# bpftool link show çıktısından:
# <link_id>: xdp  prog <prog_id>
#
# Örnek:
# 123: xdp  prog 3782
#
# XDP tipindeki linkleri bul ve detach et.

while read -r link_id; do
    if [ -n "$link_id" ]; then
        echo "Detaching BPF XDP link: $link_id"

        bpftool link detach id "$link_id" 2>/dev/null || true
    fi
done < <(
    bpftool link show 2>/dev/null |
    awk '
        /^[0-9]+:/ && / xdp / {
            gsub(":", "", $1)
            print $1
        }
    '
)

# ------------------------------------------------------------
# 3. Son durumu göster
# ------------------------------------------------------------

echo
echo "[.] XDP durumu:"
bpftool net show || true

echo
echo "[.] Interface durumu:"
ip -details link show || true

echo
echo "[.] Kalan BPF programları:"
bpftool prog show || true

echo
echo "[✓] XDP detach işlemi tamamlandı."