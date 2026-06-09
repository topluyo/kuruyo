package main

import (
	"net"
	"sync"
)

/*
	UnixSocketClient: Otomatik reconnect destekli Unix Socket Client

	USAGE:

	client := UnixSocketClientInit("/web/sockets/21600-LOG.sock")

	response := client.Request("UserCount")

	client.Close()
*/

type UnixSocketClient struct {
	socketPath string
	conn       net.Conn
	mu         sync.Mutex
}

func UnixSocketClientInit(socketPath string) *UnixSocketClient {
	c := &UnixSocketClient{
		socketPath: socketPath,
	}

	_ = c.reconnect()

	return c
}

func (c *UnixSocketClient) reconnect() error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

func (c *UnixSocketClient) request(msg string) (string, error) {
	data := []byte(msg + "\x00")

	_, err := c.conn.Write(data)
	if err != nil {
		return "", err
	}

	buf := make([]byte, 1024)

	n, err := c.conn.Read(buf)
	if err != nil {
		return "", err
	}

	return string(buf[:n]), nil
}

func (c *UnixSocketClient) Request(msg string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Bağlantı yoksa bağlan
	if c.conn == nil {
		if err := c.reconnect(); err != nil {
			return ""
		}
	}

	// İlk deneme
	resp, err := c.request(msg)
	if err == nil {
		return resp
	}

	// Hata varsa yeniden bağlan
	if err := c.reconnect(); err != nil {
		return ""
	}

	// Aynı isteği tekrar gönder
	resp, err = c.request(msg)
	if err != nil {
		return ""
	}

	return resp
}

func (c *UnixSocketClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}
