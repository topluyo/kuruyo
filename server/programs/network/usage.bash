while true; do 
  a=$(cat /sys/class/net/eth0/statistics/rx_bytes)
  sleep 1
  b=$(cat /sys/class/net/eth0/statistics/rx_bytes)
  awk "BEGIN {printf \"%.2f\n\", ($b-$a)*8/1000000}"
done | socat - UNIX-LISTEN:/tmp/net_speed.sock,fork
