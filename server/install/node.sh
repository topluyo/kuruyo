#!/bin/bash
echo "==========================================="
echo "==              NODE.JS                  =="
echo "==========================================="

set -e

VERSION="v24.11.1"
FILE="node-${VERSION}-linux-x64"
URL="https://nodejs.org/dist/${VERSION}/${FILE}.tar.xz"
INSTALL_DIR="/program/files/node"

echo "📥 Downloading Node.js ${VERSION}..."
curl -fsSL "$URL" -o node.tar.xz

echo "📦 Extracting..."
mkdir -p "$INSTALL_DIR"
tar -xf node.tar.xz -C "$INSTALL_DIR" --strip-components=1

echo "🧹 Cleaning downloaded file..."
rm -f node.tar.xz

echo "🔗 Creating symlinks..."
ln -sf "${INSTALL_DIR}/bin/node" /usr/local/bin/node
ln -sf "${INSTALL_DIR}/bin/npm" /usr/local/bin/npm
ln -sf "${INSTALL_DIR}/bin/npx" /usr/local/bin/npx

echo "✅ Node installation completed!"
echo "Node version: $(node -v)"
echo "NPM version : $(npm -v)"
