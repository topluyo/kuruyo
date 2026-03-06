#!/bin/bash

echo "==========================================="
echo "==              MARIADB                  =="
echo "==========================================="


export DEBIAN_FRONTEND=noninteractive

echo "🔧  Updating system..."
apt-get update -y
apt-get install -y mariadb-server

echo "🔒  Enforcing local-only access (bind-address = 127.0.0.1)..."
sed -i "s/^bind-address.*/bind-address = 127.0.0.1/" /etc/mysql/mariadb.conf.d/50-server.cnf
if ! grep -q "bind-address" /etc/mysql/mariadb.conf.d/50-server.cnf; then
    echo "bind-address = 127.0.0.1" >> /etc/mysql/mariadb.conf.d/50-server.cnf
fi

systemctl restart mariadb

echo "👤  Creating LOCAL-ONLY MariaDB superuser (master@localhost)..."
mysql <<EOF
CREATE USER IF NOT EXISTS 'master'@'localhost' IDENTIFIED BY 'master';
GRANT ALL PRIVILEGES ON *.* TO 'master'@'localhost' WITH GRANT OPTION;
FLUSH PRIVILEGES;
EOF

echo "🛑  Removing ANY remote user access..."
mysql <<EOF
DROP USER IF EXISTS 'master'@'%';
FLUSH PRIVILEGES;
EOF

echo "🔒  MariaDB remote access fully disabled."
echo "   • Port 3306 listens ONLY on 127.0.0.1"
echo "   • User 'master' allowed ONLY from localhost"
echo "   • No external connections possible"

echo "🚀  MariaDB installation complete!"
echo "👉  To connect locally: mysql -u master -p"
