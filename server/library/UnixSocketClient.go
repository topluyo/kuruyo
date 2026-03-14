package main
import(
	"net"
)
/*
	UnixSocketClient: Basit UnixSocketClient uygulaması

	USAGE:
	____________________________
	var client *UnixSocketClientStruct
	func main() {
		client = UnixSocketClient("/web/sockets/21600-LOG.sock")
		var response string = client.Request("UserCount")
	}
*/

type UnixSocketClientStruct struct {
	socketPath string
	conn       net.Conn
}

func UnixSocketClient(socketPath string) *UnixSocketClientStruct {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil
	}
	c := &UnixSocketClientStruct{
		socketPath: socketPath,
		conn:       conn,
	}

	return c
}

func (c *UnixSocketClientStruct) Request(msg string) string {
	if c.conn == nil {
		return ""
	}

	data := []byte(msg + "\x00")

	_, err := c.conn.Write(data)
	if err != nil {
		return ""
	}

	buf := make([]byte, 1024)
	n, err := c.conn.Read(buf)
	if err != nil {
		return ""
	}

	return string(buf[:n])
}

func (c *UnixSocketClientStruct) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}
//--------------------------------------------------------------
