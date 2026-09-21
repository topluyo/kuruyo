package main

import (
	"bufio"
	"fmt"
	"os"

	"strings"
	"time"
)

func countEstablished(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	count := 0
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	// İlk satır header
	scanner.Scan()

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		// TCP formatında state alanı 4. field'dır:
		// sl local_address rem_address st ...
		if len(fields) < 4 {
			continue
		}

		// 01 = TCP_ESTABLISHED
		if fields[3] == "01" {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return count, nil
}

func countTCPConnections() (int, error) {
	tcp4, err := countEstablished("/proc/net/tcp")
	if err != nil {
		return 0, err
	}

	tcp6, err := countEstablished("/proc/net/tcp6")
	if err != nil {
		return 0, err
	}

	return tcp4 + tcp6, nil
}

var client *UnixSocketClient 
func main() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()



	client = UnixSocketClientInit("/web/sockets/erlik.sock")
	
	defer client.Close()



	for {
		count, err := countTCPConnections()
		if err != nil {
			fmt.Println("error:", err)
		} else {
			Check(count)
			//fmt.Printf("ESTABLISHED TCP connections: %d\n", count)
		}

		<-ticker.C
	}
}

var white int
var mode int = 0
func Check(num int){
  
	if(num > 6000){
		write( client.Request("MODE,1") )
		white = 0
		mode  = 1
	}else{
		white++
		if(white>6){
			white = 0 
			if(mode!=0){
				write( client.Request("MODE,0") )
			}else{
				client.Request("MODE,0")
			}
			mode  = 0
		}
	}
	fmt.Printf("[%s] Sockets: %d  Mode: %d \n", time.Now().Format("2006-01-02 15:04:05"),num, mode, )

}
