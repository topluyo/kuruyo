#!/bin/bash

# Maks değerler
MAX_NET=10     # Mbps
MAX_CPU=100    # %
MAX_RAM=100    # %

BOLD="\033[1;97m"
END="\033[0m"


declare -A rx_hist tx_hist
declare -A rx_time tx_time

while true; do

    # CPU
    cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{printf "%6.2f", 100 - $8}')
    cpu_cores=$(nproc)

    # RAM
    total_ram=$(free -h | awk '/^Mem:/ {print $2}')
    used_ram=$(free -h | awk '/^Mem:/ {print $3}')
    used_ram_percent=$(free | awk '/^Mem:/ {printf "%.2f", $3/$2*100}')

    # Ağ hızları için ön değerleri al
    declare -A rx1 tx1
    for iface in $(ls /sys/class/net | grep -v lo); do
        rx1[$iface]=$(cat /sys/class/net/$iface/statistics/rx_bytes)
        tx1[$iface]=$(cat /sys/class/net/$iface/statistics/tx_bytes)
    done

    sleep 1  # 1 saniye bekle

    # Çıktıyı string olarak hazırla
    output="\033[H\033[2J"
    output+="\033[1;44m                                          \033[0m\n"
    output+="\033[1;44m              SYSTEM MONITOR              \033[0m\n"
    output+="\033[1;44m                                          \033[0m\n\n"
    output+=" Time: \033[1m$(date)\033[0m\n"
    output+="──────────────────────────────────────────\n"

    for iface in $(ls /sys/class/net | grep -v lo); do
        rx2=$(cat /sys/class/net/$iface/statistics/rx_bytes)
        tx2=$(cat /sys/class/net/$iface/statistics/tx_bytes)

        rx_rate=$(awk "BEGIN {printf \"%.2f\", (${rx2} - ${rx1[$iface]}) * 8 / 1024 / 1024}")
        tx_rate=$(awk "BEGIN {printf \"%.2f\", (${tx2} - ${tx1[$iface]}) * 8 / 1024 / 1024}")

        cpu_color=$(awk -v cpu="$cpu_usage" 'BEGIN {if(cpu>80) print "\033[1;31m"; else print "\033[1;32m"}')

        # Çubuk uzunlukları (0-20) – float hesaplayıp round ve sınırla
        rx_bar_len=$(awk -v val="$rx_rate" -v max="$MAX_NET" 'BEGIN {l=int(val/max*20+0.5); if(l>20) l=20; if(l<0) l=0; print l}')
        tx_bar_len=$(awk -v val="$tx_rate" -v max="$MAX_NET" 'BEGIN {l=int(val/max*20+0.5); if(l>20) l=20; if(l<0) l=0; print l}')
        cpu_bar_len=$(awk -v val="$cpu_usage" -v max="$MAX_CPU" 'BEGIN {l=int(val/max*20+0.5); if(l>20) l=20; if(l<0) l=0; print l}')
        ram_bar_len=$(awk -v val="$used_ram_percent" -v max="$MAX_RAM" 'BEGIN {l=int(val/max*20+0.5); if(l>20) l=20; if(l<0) l=0; print l}')

        # Çubukları oluştur
        rx_bar=$(printf "%0.s█" $(seq 1 $rx_bar_len))
        tx_bar=$(printf "%0.s█" $(seq 1 $tx_bar_len))
        cpu_bar=$(printf "%0.s█" $(seq 1 $cpu_bar_len))
        ram_bar=$(printf "%0.s█" $(seq 1 $ram_bar_len))

        # Boşlukları ekle
        rx_bar+=$(printf "%0.s " $(seq 1 $((20 - rx_bar_len))))
        tx_bar+=$(printf "%0.s " $(seq 1 $((20 - tx_bar_len))))
        cpu_bar+=$(printf "%0.s " $(seq 1 $((20 - cpu_bar_len))))
        ram_bar+=$(printf "%0.s " $(seq 1 $((20 - ram_bar_len))))


        # ---- SON 10 SANİYE BUFFER ----
        rx_hist[$iface]="${rx_hist[$iface]} $rx2"
        tx_hist[$iface]="${tx_hist[$iface]} $tx2"

        # 10 değer sınırı (FIFO)
        rx_hist[$iface]=$(echo "${rx_hist[$iface]}" | awk '{for(i=NF-9;i<=NF;i++) if(i>0) printf $i" "}')
        tx_hist[$iface]=$(echo "${tx_hist[$iface]}" | awk '{for(i=NF-9;i<=NF;i++) if(i>0) printf $i" "}')

        # Son 10 saniye RX/TX Mbps hesapla
        rx_10s=$(echo "${rx_hist[$iface]}" | awk '
        {
            for(i=2;i<=NF;i++) sum += ($i - $(i-1))
            print (sum * 8 / 1024 / 1024 / 10)
        }')

        tx_10s=$(echo "${tx_hist[$iface]}" | awk '
        {
            for(i=2;i<=NF;i++) sum += ($i - $(i-1))
            print (sum * 8 / 1024 / 1024 / 10)
        }')


        output+="  RX(10s): ${BOLD}${rx_10s:-0.00} Mbps${END}\n"
        output+="  TX(10s): ${BOLD}${tx_10s:-0.00} Mbps${END}\n"
        output+="  RX:   ${BOLD}${rx_rate} Mbps${END}  ${rx_bar}\n"
        output+="  TX:   ${BOLD}${tx_rate} Mbps${END}  ${tx_bar}\n"
        output+=" CPU: ${cpu_color}${cpu_usage} %     ${cpu_bar} \033[0m\n"
        output+=" RAM:  ${BOLD}${used_ram_percent} %${END}     ${ram_bar}\n"

    done

    output+="──────────────────────────────────────────\n"
    output+=" Interface: \033[1m$iface\033[0m\n"
    output+=" CPU Cores: ${cpu_cores} cores\n"
    output+=" RAM Usage: ${used_ram}/${total_ram}\n"
    output+="──────────────────────────────────────────\n"

    echo -e "$output"

done