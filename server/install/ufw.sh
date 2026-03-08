#!/bin/bash
echo "==========================================="
echo "==              UFW                     ==="
echo "==========================================="

# UFW KURULUMU
apt update
apt install -y ufw
ufw default deny incoming
ufw default allow outgoing
ufw enable
