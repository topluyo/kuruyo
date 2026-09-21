#!/bin/bash

set -e

BPFFS="/sys/fs/bpf"

# bpffs mount et
if ! mount | grep -q "on $BPFFS "; then
    mount -t bpf bpf "$BPFFS"
fi

echo "[.] Pinned BPF map ve objelerini siliyor..."

find "$BPFFS" -depth | while read -r obj; do
    [ "$obj" = "$BPFFS" ] && continue

    if [ -f "$obj" ] || [ -L "$obj" ]; then
        echo "  rm $obj"
        rm -f "$obj"
    elif [ -d "$obj" ]; then
        rmdir "$obj" 2>/dev/null || true
    fi
done

echo
echo "Maps:"
bpftool map show || true

echo
echo "[✓] BPF map temizleme tamamlandı."