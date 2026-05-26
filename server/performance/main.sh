#!/bin/bash

echo "===== System Resource Usage ====="



echo "Internet Speed (per interface):"
for iface in $(ls /sys/class/net | grep -v lo); do
    rx1=$(cat /sys/class/net/$iface/statistics/rx_bytes)
    tx1=$(cat /sys/class/net/$iface/statistics/tx_bytes)

    sleep 1

    rx2=$(cat /sys/class/net/$iface/statistics/rx_bytes)
    tx2=$(cat /sys/class/net/$iface/statistics/tx_bytes)

    rx_rate=$(awk "BEGIN {printf \"%.2f\", ($rx2 - $rx1) * 8 / 1024 / 1024}")
    tx_rate=$(awk "BEGIN {printf \"%.2f\", ($tx2 - $tx1) * 8 / 1024 / 1024}")

    echo "$iface: RX ${rx_rate} Mbps, TX ${tx_rate} Mbps"
done

# --- CPU Usage / Max ---
cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print 100 - $8"%"}')
cpu_cores=$(nproc)
echo "CPU Usage: $cpu_usage"
echo "CPU Max Cores: $cpu_cores"
echo ""

# --- RAM Usage / Max ---
total_ram=$(free -h | awk '/^Mem:/ {print $2}')
used_ram=$(free -h | awk '/^Mem:/ {print $3}')
echo "RAM Usage: $used_ram / $total_ram"

echo "================================="
