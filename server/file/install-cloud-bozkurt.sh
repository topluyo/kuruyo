sudo apt-get update
sudo apt install nano
sudo apt-get install nginx -y
sudo nginx -v
sudo apt install php8.2-fpm -y
sudo apt-get install php8.2-zip -y
sudo apt-get install php8.2-mysql -y
sudo apt-get install php8.2-curl -y

sudo apt-get install php-gd -y
service php8.2-fpm restart




sudo apt install -y ufw
sudo ufw allow ssh
sudo ufw allow 1453/tcp
sudo ufw enable
sudo ufw status


wget https://dev.mysql.com/get/mysql-apt-config_0.8.29-1_all.deb
sudo dpkg -i mysql-apt-config_0.8.29-1_all.deb
sudo apt update
sudo apt install mysql-server -y
sudo systemctl start mysql




sudo apt update
sudo apt install jq -y
