#!/bin/bash

while true; do
  for iface in $(ls /sys/class/net | grep -v lo); do
      rx1=$(cat /sys/class/net/$iface/statistics/rx_bytes)
      tx1=$(cat /sys/class/net/$iface/statistics/tx_bytes)

      sleep 1

      rx2=$(cat /sys/class/net/$iface/statistics/rx_bytes)
      tx2=$(cat /sys/class/net/$iface/statistics/tx_bytes)

      rx_rate=$(awk "BEGIN {printf \"%.2f\", ($rx2 - $rx1) * 8 / 1024 / 1024}")
      tx_rate=$(awk "BEGIN {printf \"%.2f\", ($tx2 - $tx1) * 8 / 1024 / 1024}")

      cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print 100 - $8}')
      cpu_cores=$(nproc)

      cpu_color=$(awk -v cpu="$cpu_usage" 'BEGIN {if(cpu>80) print "\033[1;31m"; else print "\033[1;32m"}')


      total_ram=$(free -h | awk '/^Mem:/ {print $2}')
      used_ram=$(free -h | awk '/^Mem:/ {print $3}')

      
      echo -e "\
RX: \033[1;32m${rx_rate} Mbps\033[0m\t\
TX: \033[1;36m${tx_rate} Mbps\033[0m\t\
CPU: ${cpu_color}${cpu_usage}%\033[0m (${cpu_cores} cores)\t\
RAM: \033[1;35m${used_ram}/${total_ram}\033[0m"


  done
done