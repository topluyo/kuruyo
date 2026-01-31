#!/bin/bash

echo "===== System Resource Usage ====="

# --- Internet Usage / Bandwidth ---
echo "Internet Usage (per interface):"
for iface in $(ls /sys/class/net | grep -v lo); do
    rx_bytes=$(cat /sys/class/net/$iface/statistics/rx_bytes)
    tx_bytes=$(cat /sys/class/net/$iface/statistics/tx_bytes)
    echo "$iface: RX $(($rx_bytes / 1024)) KB, TX $(($tx_bytes / 1024)) KB"
done
echo ""

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
