#!/bin/bash
printf 'restart=10\0' | socat - UNIX-CONNECT:/web/sockets/KURUCU.sock