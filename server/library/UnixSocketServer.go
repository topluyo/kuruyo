package main
import(
	"fmt"
	"os"
	"net"
	"bufio"
	"strings"
  "time"
)

/*

	UnixSocketServer: Basic UnixSocket Server

	for test:
	___________________________
	printf 'UserCount\0' | socat - UNIX-CONNECT:/web/sockets/SOCKET.sock 
	
	Usage:
	___________________________
	go UnixSocketServer("/web/sockets/"+PORT+"-LOG.sock", func(request string) string {
		if(request=="UserCount"){
			return "1"
		}
		return "NOT FOUND ACTION"
	})

*/
func UnixSocketServer(socketPath string, callback func(string) string) {
	
	
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Println("[X] UnixSocketServer:: Listen error:", err)
	}
	fmt.Println("[+] Initing",socketPath)
	defer listener.Close()


	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("[X] UnixSocketServer:: Accept error:", err)
			continue
		}

		go func(conn net.Conn) {
		    defer conn.Close()
		    conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		    reader := bufio.NewReader(conn)
		    for {
		        message, err := reader.ReadString('\x00')
		        if err != nil {
		            return
		        }
		        message = strings.TrimSuffix(message, "\x00")
		        response := callback(message)
		        _, err = conn.Write([]byte(response))
		        if err != nil {
		            fmt.Println("[X] UnixSocketServer:: Write error:", err)
		            return
		        }
		        conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		    }
		}(conn)
		
	}
}
