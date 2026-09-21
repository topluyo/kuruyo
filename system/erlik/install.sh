#!/bin/bash

set -e

echo "========================================"
echo " Erlik Installer"
echo "========================================"

# Root kontrolü
if [ "$EUID" -ne 0 ]; then
    echo "Hata: Bu script root olarak çalıştırılmalıdır."
    echo "Örnek: sudo bash install.sh"
    exit 1
fi

# İşletim sistemi kontrolü
if [ ! -f /etc/os-release ]; then
    echo "Hata: /etc/os-release bulunamadı."
    exit 1
fi

. /etc/os-release

echo "OS: ${PRETTY_NAME}"
echo "Architecture: $(dpkg --print-architecture)"
echo "Kernel: $(uname -r)"
echo

case "$ID" in
    debian)
        if [ "${VERSION_ID%%.*}" != "13" ]; then
            echo "Uyarı: Bu script Debian 13 Trixie için hazırlanmıştır."
        fi
        ;;

    ubuntu)
        echo "Ubuntu tespit edildi."
        ;;

    *)
        echo "Hata: Desteklenmeyen işletim sistemi: $ID"
        echo "Desteklenen sistemler: Debian 13 Trixie ve Ubuntu"
        exit 1
        ;;
esac

echo
echo "[1/5] APT cache güncelleniyor..."

# MySQL repository hatası yüzünden apt update'in tamamen
# başarısız olmasını önlemek için MySQL kaynaklarını geçici
# olarak devre dışı bırak.
MYSQL_FILES=$(grep -RIl "repo.mysql.com" \
    /etc/apt/sources.list \
    /etc/apt/sources.list.d/ 2>/dev/null || true)

if [ -n "$MYSQL_FILES" ]; then
    echo "MySQL repository bulundu:"
    echo "$MYSQL_FILES"

    while IFS= read -r file; do
        [ -z "$file" ] && continue

        echo "Geçici olarak devre dışı bırakılıyor: $file"

        if [[ "$file" == *.list ]]; then
            mv "$file" "${file}.disabled"
        elif [[ "$file" == *.sources ]]; then
            mv "$file" "${file}.disabled"
        fi
    done <<< "$MYSQL_FILES"
fi

apt-get update

echo
echo "[2/5] Gerekli paketler kuruluyor..."

PACKAGES=(
    libcapstone-dev
    binutils-dev
    clang
    llvm
    libbpf-dev
    linux-libc-dev
    bpftool
)

apt-get install -y "${PACKAGES[@]}"

echo
echo "[3/5] Erlik executable hazırlanıyor..."

if [ ! -f "./erlik" ]; then
    echo "Hata: $(pwd)/erlik bulunamadı."
    exit 1
fi

chmod +x ./erlik

echo
echo "[4/5] /usr/local/bin/erlik oluşturuluyor..."

ln -sfn "$(pwd)/erlik" /usr/local/bin/erlik

echo
echo "[5/5] BPF JIT ayarlanıyor..."

BPF_JIT="/proc/sys/net/core/bpf_jit_enable"

if [ -f "$BPF_JIT" ]; then
    echo 2 > "$BPF_JIT"
    echo "bpf_jit_enable = $(cat "$BPF_JIT")"
else
    echo "Uyarı: $BPF_JIT bulunamadı."
    echo "Kernel BPF JIT ayarı mevcut değil veya farklı bir yapı kullanılıyor."
fi

echo
echo "========================================"
echo " Erlik kurulumu tamamlandı!"
echo "========================================"
echo
echo "Executable : $(pwd)/erlik"
echo "Symlink    : /usr/local/bin/erlik"
echo

if command -v erlik >/dev/null 2>&1; then
    echo "Erlik:"
    command -v erlik
else
    echo "Uyarı: erlik PATH üzerinde bulunamadı."
fi