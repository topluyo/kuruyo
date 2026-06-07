# Usage

printf 'json:/HELLO:U1\0' | socat - UNIX-CONNECT:/web/sockets/hello.sock ;echo
