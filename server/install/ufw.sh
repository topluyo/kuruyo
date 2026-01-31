#!/bin/bash

# UFW KURULUMU
sudo apt update
sudo apt install -y ufw
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw enable
