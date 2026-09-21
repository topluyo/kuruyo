#!/bin/bash

ROOT="/web/server/system/erlik";
SRC="bozkurt.c"
OBJ="bozkurt.o"
cd $ROOT


for iface in $(ip -o link show | awk -F': ' '{print $2}' | cut -d@ -f1); do
  ip link set dev "$iface" xdp off 2>/dev/null
done
echo "[+] Tüm XDP programları kaldırıldı."

set -e

BPFFS="/sys/fs/bpf"

# bpffs mount et
if ! mount | grep -q "on $BPFFS "; then
    mount -t bpf bpf "$BPFFS"
fi

echo "[.] XDP programlarını interface'lerden ayırıyor..."
for iface in /sys/class/net/*; do
    iface=$(basename "$iface")
    ip link set dev "$iface" xdp off 2>/dev/null || true
done

echo "[.] Pinned BPF objelerini siliyor..."

find "$BPFFS" -depth | while read -r obj; do
    [ "$obj" = "$BPFFS" ] && continue

    if [ -f "$obj" ] || [ -L "$obj" ]; then
        echo "  rm $obj"
        rm -f "$obj"
    elif [ -d "$obj" ]; then
        rmdir "$obj" 2>/dev/null || true
    fi
done

echo "[.] Kontrol"

echo
echo "Programs:"
bpftool prog show || true

echo
echo "Maps:"
bpftool map show || true

echo
echo "[✓] Temizleme tamamlandı."