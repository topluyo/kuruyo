#!/bin/bash
# Cloudflare only access – IPv4 + IPv6
# Hasan Delibas use-case 😎

### === GÜVENLİK ===
# SSH kesilmesin diye önce izin veriyoruz
SSH_PORT=22

### === TEMİZLE ===
iptables -F
iptables -X
iptables -t nat -F
iptables -t mangle -F

ip6tables -F
ip6tables -X

### === DEFAULT POLICY ===
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

ip6tables -P INPUT DROP
ip6tables -P FORWARD DROP
ip6tables -P OUTPUT ACCEPT

### === LOOPBACK ===
iptables -A INPUT -i lo -j ACCEPT
ip6tables -A INPUT -i lo -j ACCEPT

### === ESTABLISHED ===
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
ip6tables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

### === CLOUDFLARE IPv4 ===
CF_IPV4=(
173.245.48.0/20
103.21.244.0/22
103.22.200.0/22
103.31.4.0/22
141.101.64.0/18
108.162.192.0/18
190.93.240.0/20
188.114.96.0/20
197.234.240.0/22
198.41.128.0/17
162.158.0.0/15
104.16.0.0/13
104.24.0.0/14
172.64.0.0/13
131.0.72.0/22
88.235.214.15
)

for ip in "${CF_IPV4[@]}"; do
  iptables -A INPUT -p tcp -m multiport --dports 80,443,8080,8443,2096,2086 -s $ip -j ACCEPT
done

### === CLOUDFLARE IPv6 ===
CF_IPV6=(
2400:cb00::/32
2606:4700::/32
2803:f800::/32
2405:b500::/32
2405:8100::/32
2a06:98c0::/29
2c0f:f248::/32
)

for ip in "${CF_IPV6[@]}"; do
  ip6tables -A INPUT -p tcp -m multiport --dports 80,443,8080,8443,2096,2086 -s $ip -j ACCEPT
done

### === ICMP (opsiyonel ama önerilir) ===
iptables -A INPUT -p icmp -j ACCEPT
ip6tables -A INPUT -p ipv6-icmp -j ACCEPT

echo "✅ Cloudflare-only iptables rules applied successfully."

