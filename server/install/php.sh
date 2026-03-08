echo "==========================================="
echo "==              PHP                     ==="
echo "==========================================="

#apt update
#apt install -y php8.2 php8.2-cli php8.2-fpm php8.2-curl php8.2-mbstring php8.2-xml php8.2-zip php8.2-mysql php8.2-gd

apt update
apt install -y lsb-release ca-certificates curl
curl -sSLo /tmp/debsuryorg-archive-keyring.deb https://packages.sury.org/debsuryorg-archive-keyring.deb
dpkg -i /tmp/debsuryorg-archive-keyring.deb
tee /etc/apt/sources.list.d/php.sources <<EOF
Types: deb
URIs: https://packages.sury.org/php/
Suites: $(lsb_release -sc)
Components: main
Signed-By: /usr/share/keyrings/debsuryorg-archive-keyring.gpg
EOF
apt update


apt install -y software-properties-common
LC_ALL=C.UTF-8 add-apt-repository ppa:ondrej/php -y
apt update


# Install PHP.
apt install -y php8.2 php8.2-cli php8.2-fpm php8.2-curl php8.2-mbstring php8.2-xml php8.2-zip php8.2-mysql php8.2-gd
