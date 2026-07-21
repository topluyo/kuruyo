/program/go run main.go shield=on http=:9090 upstream=http://127.0.0.1:8080 \
  mods=syn,http,slow,iot,botnet,keep \
  cf=/web/config/.cloudflare.json \
  ipset=blacklist