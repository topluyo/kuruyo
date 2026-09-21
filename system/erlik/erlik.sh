#!/bin/bash

ROOT="/web/server/system/test";
SRC="bozkurt.c"
OBJ="bozkurt.o"
cd $ROOT


if [ $# -eq 0 ]; then
  echo "╔════════════════════════════════════════════════╗"
  echo "║                     erlik                      ║"
  echo "╚════════════════════════════════════════════════╝"
  echo "  build"
  echo "  enable"
  echo "  disable"
  echo "  delete"
  
  echo ""
  exit 0
fi





if [ "$1" = "build" ]; then
  /app/go run FrontBuild.go Library.go  Bash.go
  exit 0
  if clang -target bpf -O2 -g -Wall -c "$SRC" -o "$OBJ" -I/usr/include/x86_64-linux-gnu; then
    echo "[+] build successful"
  else
    echo "[-] build failed"
    exit 1
  fi
fi


if [ "$1" = "enable" ]; then

  if [ ! -f "$OBJ" ]; then
    echo "[X] $OBJ bulunamadı."
    exit 1
  fi

  # bpffs mount et (gerekliyse)
  if ! mount | grep -q "on /sys/fs/bpf "; then
    echo "[.] /sys/fs/bpf mount ediliyor..."
    mount -t bpf bpf /sys/fs/bpf
  fi

  # Varsayılan arayüzü bul
  IFACE=$(ip route show default | awk '{print $5}' | head -n1)

  if [ -z "$IFACE" ]; then
    echo "[X] Varsayılan ağ arayüzü bulunamadı."
    exit 1
  fi

  for iface in $(ls /sys/class/net); do
    ip link set dev "$iface" xdp off 2>/dev/null
  done

  # Eski XDP programını kaldır
  ip link set dev "$IFACE" xdp off 2>/dev/null || true

  # Yeni XDP programını yükle
  if ip link set dev "$IFACE" xdp obj "$OBJ" sec xdp; then
    echo "[+] Bozkurt XDP başarıyla yüklendi: $IFACE"
  else
    echo "[X] XDP yüklenemedi."
    exit 1
  fi

  exit 0
fi




if [ "$1" = "disable" ]; then
  for iface in $(ip -o link show | awk -F': ' '{print $2}' | cut -d@ -f1); do
    ip link set dev "$iface" xdp off 2>/dev/null
  done
  echo "[+] Tüm XDP programları kaldırıldı."
  exit 0
fi


if [ "$1" = "delete" ]; then

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

  echo "[*] XDP programlarını interface'lerden ayırıyor..."
  for iface in /sys/class/net/*; do
      iface=$(basename "$iface")
      ip link set dev "$iface" xdp off 2>/dev/null || true
  done

  echo "[*] Pinned BPF objelerini siliyor..."

  find "$BPFFS" -depth | while read -r obj; do
      [ "$obj" = "$BPFFS" ] && continue

      if [ -f "$obj" ] || [ -L "$obj" ]; then
          echo "  rm $obj"
          rm -f "$obj"
      elif [ -d "$obj" ]; then
          rmdir "$obj" 2>/dev/null || true
      fi
  done

  echo "[*] Kontrol"

  echo
  echo "Programs:"
  bpftool prog show || true

  echo
  echo "Maps:"
  bpftool map show || true

  echo
  echo "[✓] Temizleme tamamlandı."
fi







if [ "$1" = "info" ]; then


xdp_count=$(bpftool -j prog show | jq '[.[] | select(.type=="xdp")] | length')
if [ "$xdp_count" -eq 0 ]; then
  echo "[X] No XDP program installed."
  exit 0
fi

bpftool -j prog show | jq -c '.[] | select(.type=="xdp")' | while read prog; do

prog_id=$(echo "$prog" | jq -r '.id')
prog_name=$(echo "$prog" | jq -r '.name')
jited=$(echo "$prog" | jq -r '.jited')
mem=$(echo "$prog" | jq -r '.bytes_memlock // 0')
mem_mb=$(awk "BEGIN {printf \"%.3f\",$mem/1024/1024}")

echo ""
echo "Program"
echo "  name: $prog_name"
echo "  id: $prog_id"
[ "$jited" = "true" ] && echo "  jited: yes" || echo "  jited: no"
echo "  mem: ${mem_mb} MB"

prog_pin=$(find /sys/fs/bpf -type f -name "*" 2>/dev/null | while read f; do
id=$(bpftool -j prog show pinned "$f" 2>/dev/null | jq -r 'if type=="array" then .[0].id else .id end // empty')
[ "$id" = "$prog_id" ] && echo "$f" && break
done)

[ -n "$prog_pin" ] && echo "  pinned: yes ($prog_pin)" || echo "  pinned: no"

echo ""
echo "Maps"
printf "  %-20s %-15s %-12s %-10s %-10s\n" "name" "type" "memory" "items" "pinned"
echo "  --------------------------------------------------------------------"

for map_id in $(echo "$prog" | jq -r '.map_ids[]?'); do

map=$(bpftool -j map show id "$map_id" 2>/dev/null)
printf "  "
name=$(echo "$map" | jq -r '.name')
type=$(echo "$map" | jq -r '.type')
bytes=$(echo "$map" | jq -r '.bytes_memlock // 0')
mem_mb=$(awk "BEGIN {printf \"%.2f\",$bytes/1024/1024}")

case "$type" in
ringbuf) item="N/A";;
*) item=$(bpftool -j map dump id "$map_id" 2>/dev/null | jq 'length' 2>/dev/null);;
esac

[[ "$item" =~ ^[0-9]+$ ]] || item="N/A"

map_pin=$(find /sys/fs/bpf -type f -name "map_$name" 2>/dev/null | head -1)

if [ -n "$map_pin" ]; then
pinned="yes"
else
pinned="no"
fi

printf "%-20s %-15s %-12s %-10s %-4s" "$name" "$type" "${mem_mb} MB" "$item" "$pinned"

[ -n "$map_pin" ] && printf "$map_pin"

echo ""
done

echo ""

done

fi