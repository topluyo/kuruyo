#!/bin/bash

ROOT="/web/server/system/erlik";
SRC="bozkurt.c"
OBJ="bozkurt.o"
cd $ROOT


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