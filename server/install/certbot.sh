#!/bin/bash

echo "==========================================="
echo "==           CERTBOT SSL                 =="
echo "==========================================="


apt update
apt install certbot -y



####################   Kullanım   #######################
# certbot certonly --standalone --preferred-challenges http -d yourdomain.com --register-unsafely-without-email
# /etc/letsencrypt/live/yourdomain.com/ -> içine dosyalar yüklenir
# cert.pem
# privkey.pem
#########################################################


