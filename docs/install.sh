apt update
apt upgrade
apt install curl unzip
curl -L -o kuruyo.zip https://github.com/topluyo/kuruyo/archive/refs/heads/master.zip
unzip kuruyo.zip
mv kuruyo-main/ /web
rm kuruyo.zip
