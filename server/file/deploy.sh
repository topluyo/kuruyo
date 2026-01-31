#!/bin/bash


SEARCH_DIR="${1:-/}"  # Parametre verilmezse / dizininde ara
DEST_DIR="/etc/nginx/sites-enabled/CB"

echo ":: Removing: sites-enabled/CB* "
rm -f /etc/nginx/sites-enabled/CB*


all_ok=true

find "$SEARCH_DIR" -type f -name "*server.nginx.conf" 2>/dev/null \
  | awk -F/ '{ print NF, $0 }' | sort -n | cut -d' ' -f2- \
  | while IFS= read -r file; do

  # echo ":: Processing file: $file"

  filename=$(basename "$file")
  hash_name="$(echo -n "$file" | sha256sum | cut -c1-8).conf"
  link_path="$DEST_DIR-${filename}-$hash_name"

  # Create symlink
  ln -s "$file" "$link_path"

  # Test nginx with the symlinked config
  if nginx -t &>/dev/null; then
    echo "++ Test passed. $file"
  else
    echo "-- Test failed for $file"
    nginx -t
    rm "$link_path"
    all_ok=false
  fi
done

if $all_ok; then
  echo ":: Nginx sites successfully updated."
  service nginx reload
else
  echo ":: Errors detected. Nginx reload skipped."
fi
