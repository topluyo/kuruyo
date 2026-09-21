#!/bin/bash

set -Eeuo pipefail

echo "========================================"
echo " Erlik Installer"
echo "========================================"

# ------------------------------------------------------------
# Hata yakalama
# ------------------------------------------------------------

trap 'echo; echo "Hata: Kurulum sırasında bir problem oluştu."; echo "Satır: $LINENO"; exit 1' ERR

# ------------------------------------------------------------
# Root kontrolü
# ------------------------------------------------------------

if [ "$EUID" -ne 0 ]; then
    echo "Hata: Bu script root olarak çalıştırılmalıdır."
    echo "Örnek: sudo bash install.sh"
    exit 1
fi

# ------------------------------------------------------------
# /etc/os-release kontrolü
# ------------------------------------------------------------

if [ ! -f /etc/os-release ]; then
    echo "Hata: /etc/os-release bulunamadı."
    exit 1
fi

. /etc/os-release

ARCH="$(dpkg --print-architecture)"
KERNEL="$(uname -r)"
CODENAME="${VERSION_CODENAME:-}"

echo "OS: ${PRETTY_NAME}"
echo "Architecture: ${ARCH}"
echo "Kernel: ${KERNEL}"

if [ -n "$CODENAME" ]; then
    echo "Codename: ${CODENAME}"
fi

echo

# ------------------------------------------------------------
# İşletim sistemi kontrolü
# ------------------------------------------------------------

case "$ID" in
    debian)
        case "${VERSION_ID%%.*}" in
            12)
                echo "Debian 12 Bookworm tespit edildi."
                ;;
            13)
                echo "Debian 13 Trixie tespit edildi."
                ;;
            *)
                echo "Uyarı: Desteklenen Debian sürümleri 12 ve 13'tür."
                ;;
        esac
        ;;

    ubuntu)
        echo "Ubuntu tespit edildi."
        ;;

    *)
        echo "Hata: Desteklenmeyen işletim sistemi: $ID"
        echo "Desteklenen sistemler: Debian 12, Debian 13 ve Ubuntu"
        exit 1
        ;;
esac

# ------------------------------------------------------------
# APT repository yardımcı fonksiyonları
# ------------------------------------------------------------

disable_mysql_repositories() {
    echo
    echo "MySQL repository kontrol ediliyor..."

    local mysql_files

    mysql_files=$(
        grep -RIl "repo.mysql.com" \
            /etc/apt/sources.list \
            /etc/apt/sources.list.d/ \
            2>/dev/null || true
    )

    if [ -z "$mysql_files" ]; then
        echo "MySQL repository bulunamadı."
        return 0
    fi

    echo "MySQL repository bulundu:"
    echo "$mysql_files"

    while IFS= read -r file; do
        [ -z "$file" ] && continue
        [ ! -f "$file" ] && continue

        case "$file" in
            *.list)
                if [ ! -f "${file}.disabled" ]; then
                    echo "Geçici olarak devre dışı bırakılıyor: $file"
                    mv "$file" "${file}.disabled"
                fi
                ;;

            *.sources)
                if [ ! -f "${file}.disabled" ]; then
                    echo "Geçici olarak devre dışı bırakılıyor: $file"
                    mv "$file" "${file}.disabled"
                fi
                ;;
        esac
    done <<< "$mysql_files"
}

setup_sury_repository() {
    echo
    echo "PHP Sury repository kontrol ediliyor..."

    local sury_found

    sury_found=$(
        grep -RIl "packages.sury.org/php" \
            /etc/apt/sources.list \
            /etc/apt/sources.list.d/ \
            2>/dev/null || true
    )

    # Sury repository hiç yoksa hiçbir şey yapma.
    if [ -z "$sury_found" ]; then
        echo "Sury PHP repository bulunamadı."
        return 0
    fi

    echo "Sury PHP repository bulundu."

    echo "Gerekli GPG paketleri kuruluyor..."

    apt-get install -y \
        ca-certificates \
        curl \
        gnupg

    local keyring="/etc/apt/keyrings/sury-php.gpg"

    install -d -m 0755 /etc/apt/keyrings

    echo "Sury PHP GPG anahtarı indiriliyor..."

    curl -fsSL \
        https://packages.sury.org/php/apt.gpg \
        -o "$keyring"

    chmod 0644 "$keyring"

    # --------------------------------------------------------
    # Eski Sury kaynaklarını devre dışı bırak
    # --------------------------------------------------------

    while IFS= read -r file; do
        [ -z "$file" ] && continue
        [ ! -f "$file" ] && continue

        case "$file" in
            /etc/apt/sources.list)
                echo "Uyarı: Sury kaynağı ana sources.list içinde bulunuyor."
                ;;

            *.list)
                echo "Eski Sury repository dosyası devre dışı bırakılıyor: $file"

                if [ ! -f "${file}.disabled" ]; then
                    mv "$file" "${file}.disabled"
                fi
                ;;

            *.sources)
                echo "Eski Sury repository dosyası devre dışı bırakılıyor: $file"

                if [ ! -f "${file}.disabled" ]; then
                    mv "$file" "${file}.disabled"
                fi
                ;;
        esac
    done <<< "$sury_found"

    # --------------------------------------------------------
    # Yeni ve doğru Sury repository tanımı
    # --------------------------------------------------------

    if [ -z "$CODENAME" ]; then
        echo "Hata: Debian/Ubuntu codename tespit edilemedi."
        exit 1
    fi

    cat > /etc/apt/sources.list.d/php-sury.list <<EOF
deb [signed-by=/etc/apt/keyrings/sury-php.gpg] https://packages.sury.org/php ${CODENAME} main
EOF

    chmod 0644 /etc/apt/sources.list.d/php-sury.list

    echo "Sury PHP repository yapılandırıldı:"
    echo "  https://packages.sury.org/php ${CODENAME} main"
}

# ------------------------------------------------------------
# [1/5] APT cache güncelleme
# ------------------------------------------------------------

echo
echo "[1/5] APT repository'leri hazırlanıyor..."

# MySQL repository'si bozuksa apt update'i engellemesin.
disable_mysql_repositories

# Sury PHP repository'sinin GPG keyring'ini düzelt.
setup_sury_repository

echo
echo "APT cache güncelleniyor..."

apt-get update

echo "APT cache başarıyla güncellendi."

# ------------------------------------------------------------
# [2/5] Gerekli paketler
# ------------------------------------------------------------

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

echo "Gerekli paketler başarıyla kuruldu."

# ------------------------------------------------------------
# [3/5] Erlik executable kontrolü
# ------------------------------------------------------------

echo
echo "[3/5] Erlik executable hazırlanıyor..."

ERLIK_PATH="$(pwd)/erlik"

if [ ! -f "$ERLIK_PATH" ]; then
    echo "Hata: Erlik executable bulunamadı:"
    echo "$ERLIK_PATH"
    exit 1
fi

chmod +x "$ERLIK_PATH"

echo "Executable:"
echo "$ERLIK_PATH"

# ------------------------------------------------------------
# [4/5] /usr/local/bin/erlik
# ------------------------------------------------------------

echo
echo "[4/5] /usr/local/bin/erlik oluşturuluyor..."

ln -sfn "$ERLIK_PATH" /usr/local/bin/erlik

echo "Symlink oluşturuldu:"
echo "/usr/local/bin/erlik -> $ERLIK_PATH"

# ------------------------------------------------------------
# [5/5] BPF JIT
# ------------------------------------------------------------

echo
echo "[5/5] BPF JIT ayarlanıyor..."

BPF_JIT="/proc/sys/net/core/bpf_jit_enable"

if [ -f "$BPF_JIT" ]; then

    CURRENT_JIT="$(cat "$BPF_JIT")"

    echo "Mevcut bpf_jit_enable: $CURRENT_JIT"

    if echo 2 > "$BPF_JIT"; then
        echo "bpf_jit_enable = $(cat "$BPF_JIT")"
    else
        echo "Uyarı: BPF JIT ayarlanamadı."
    fi

else

    echo "Uyarı: $BPF_JIT bulunamadı."
    echo "Kernel BPF JIT ayarı mevcut değil veya farklı bir yapı kullanılıyor."

fi

# ------------------------------------------------------------
# Kurulum doğrulama
# ------------------------------------------------------------

echo
echo "Kurulum doğrulanıyor..."

if [ -x "$ERLIK_PATH" ]; then
    echo "OK: Erlik executable çalıştırılabilir."
else
    echo "Uyarı: Erlik executable çalıştırılabilir değil."
fi

if [ -L /usr/local/bin/erlik ] || [ -x /usr/local/bin/erlik ]; then
    echo "OK: /usr/local/bin/erlik mevcut."
else
    echo "Uyarı: /usr/local/bin/erlik oluşturulamadı."
fi

# ------------------------------------------------------------
# PATH kontrolü
# ------------------------------------------------------------

echo
echo "========================================"
echo " Erlik kurulumu tamamlandı!"
echo "========================================"
echo
echo "Executable : $ERLIK_PATH"
echo "Symlink    : /usr/local/bin/erlik"
echo

if command -v erlik >/dev/null 2>&1; then
    echo "Erlik PATH üzerinde bulundu:"
    command -v erlik
else
    echo "Uyarı: erlik PATH üzerinde bulunamadı."
    echo "Ancak /usr/local/bin/erlik oluşturulmuş olabilir."
fi

echo
echo "Kurulum tamamlandı."
