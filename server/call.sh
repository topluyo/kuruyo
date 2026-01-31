#!/bin/bash

SELF=$(realpath "$0")
SELFDIR=$(dirname "$SELF")
FILE="$1"

find "$SELFDIR" -mindepth 2 -type f -name "$FILE" | while read file; do
    DIR=$(dirname "$file")     # dosyanın klasörü
    BASENAME=$(basename "$file") 

    echo "Running: $file (in $DIR)"
    (
        cd "$DIR" || exit 1
        bash "$BASENAME"
    )
done
