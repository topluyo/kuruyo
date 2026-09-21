#!/bin/bash

SRC="bozkurt.c"
OBJ="bozkurt.o"

echo "[.] building: clang -target bpf -O2 -g -Wall -c "$SRC" -o "$OBJ" -I/usr/include/x86_64-linux-gnu"

if clang -target bpf -O2 -g -Wall -c "$SRC" -o "$OBJ" -I/usr/include/x86_64-linux-gnu; then
  echo "[+] build successful"
else
  echo "[-] build failed"
  exit 1
fi