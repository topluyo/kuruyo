#!/bin/bash

export DEBIAN_FRONTEND=noninteractive

echo "🔧  Updating system..."
apt-get update -y
apt-get install -y wget gnupg lsb-release

echo "📦  Downloading MySQL APT repository..."
wget https://repo.mysql.com/mysql-apt-config_0.8.30-1_all.deb -O mysql-apt-config.deb

echo "⚙️  Installing MySQL repo (non-interactive)..."
dpkg -i mysql-apt-config.deb

echo "🗑️  Cleaning MySQL repo installer..."
rm -f mysql-apt-config.deb

echo "🔄  Updating APT repo..."
apt-get update -y

echo "🐬  Installing MySQL server..."
apt-get install -y mysql-server

echo "🔒  Disabling remote connections..."
# bind-address MUST be localhost
sed -i "s/^bind-address.*/bind-address = 127.0.0.1/" /etc/mysql/mysql.conf.d/mysqld.cnf

# If bind-address does not exist, add it
if ! grep -q "bind-address" /etc/mysql/mysql.conf.d/mysqld.cnf; then
    echo "bind-address = 127.0.0.1" >> /etc/mysql/mysql.conf.d/mysqld.cnf
fi

systemctl restart mysql

echo "👤  Creating LOCAL-ONLY MySQL superuser: master@localhost"
mysql <<EOF
CREATE USER IF NOT EXISTS 'master'@'localhost' IDENTIFIED BY 'master';
GRANT ALL PRIVILEGES ON *.* TO 'master'@'localhost' WITH GRANT OPTION;
FLUSH PRIVILEGES;
EOF

echo "🛑  Ensuring users cannot connect remotely..."
mysql <<EOF
DROP USER IF EXISTS 'master'@'%';
FLUSH PRIVILEGES;
EOF

echo "🔒  MySQL remote access is fully disabled."
echo "   • Port 3306 only listens on 127.0.0.1"
echo "   • User 'master' allowed ONLY from localhost"
echo "   • No external machine can connect"
echo "   • Firewall changes NOT required"

echo "✅  Local-only MySQL setup complete!"
echo "👉  To connect: mysql -u master -p"
