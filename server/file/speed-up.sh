echo "net.core.somaxconn = 65535" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog = 4096" >> /etc/sysctl.conf
echo "fs.file-max = 2097152" >> /etc/sysctl.conf

sudo sysctl -p
