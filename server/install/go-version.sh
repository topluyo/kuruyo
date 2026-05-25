#!/bin/bash

echo "==========================================="
echo "==            GOLANG INSTALLER           ==="
echo "==========================================="

# Gerekli paketleri kur
apt update
apt install -y git wget

# Değişkenler
GO_VERSION="1.18"
INSTALL_DIR="/program/files/go/${GO_VERSION}"
LINK_PATH="/program/go-${GO_VERSION}"
PROFILE="$HOME/.profile"

# Klasörleri oluştur
mkdir -p "/program/files/go"

# İndirme ve Ayıklama
echo "Downloading Go ${GO_VERSION}..."
wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz

echo "Installing to ${INSTALL_DIR}..."
rm -rf "${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}"
# Tar dosyasının içindeki 'go' klasörünü doğrudan INSTALL_DIR içine çıkarır
tar -xzf go${GO_VERSION}.linux-amd64.tar.gz --strip-components=1 -C "${INSTALL_DIR}"

# Kısa yol (Symlink) oluşturma
echo "Creating symlink at ${LINK_PATH}..."
rm -f "${LINK_PATH}"
ln -s "${INSTALL_DIR}/bin/go" "${LINK_PATH}"

# Geçici dosyayı temizle
rm go${GO_VERSION}.linux-amd64.tar.gz

echo "==========================================="
echo "KURULUM TAMAMLANDI"
echo "Hedef: ${INSTALL_DIR}"
echo "Kısayol: ${LINK_PATH}"
echo "==========================================="
echo "Lütfen şu komutu çalıştırın: source ~/.profile"
echo "Veya oturumu kapatıp açın."