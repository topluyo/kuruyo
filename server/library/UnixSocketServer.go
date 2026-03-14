package main
import(
	"fmt"
	"os"
	"net"
	"bufio"
	"strings"
)

/*

	UnixSocketServer: Basic UnixSocket Server

	for test:
	___________________________
	printf 'HELLO\0' | socat - UNIX-CONNECT:/tmp/socket.sock
	
	Usage:
	___________________________
	go UnixSocketServer("/web/sockets/"+PORT+"-LOG.sock", func(request string) string {
		if(request=="UserCount"){
			return ToString( "1" )
		}
		return "NOT FOUND ACTION"
	})

*/
func UnixSocketServer(socketPath string, callback func(string) string) {
	
	fmt.Println("Initing",socketPath)
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Println("UnixSocketServer:: Listen error:", err)
	}
	defer listener.Close()


	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("UnixSocketServer:: Accept error:", err)
			continue
		}

		go func(conn net.Conn) {
			defer conn.Close()
			reader := bufio.NewReader(conn)
			for {
				message, err := reader.ReadString('\x00')
				if err != nil {
					return
				}
				message = strings.TrimSuffix(message,"\x00")

				response := callback(message)
				_, err = conn.Write([]byte(response))
				if err != nil {
					fmt.Println("UnixSocketServer:: Yazma hatası:", err)
					return
				}
			}
		}(conn)
	}
}
//-----------------------------------------------------------------
