#!/bin/bash

# PORT DEFINE
set -e  
if [[ "$1" != "--port" || -z "$2" ]]; then
    echo "Usage: $0 --port <port>"
    exit 1
fi
PORT="$2"

# Initialize module if missing
[ ! -f go.mod ] && /usr/local/go/bin/go mod init app
/usr/local/go/bin/go mod tidy
/usr/local/go/bin/go run main.go --port "$PORT"