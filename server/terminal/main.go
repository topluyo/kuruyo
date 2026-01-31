package main

import (
	"log"
	"net/http"
	"time"
	"os/exec"
	"os"
	"strings"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	if access(r) {
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	// Read timeout + pong handler
	conn.SetReadLimit(2048)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Keep-alive pings
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	// Start bash PTY
	cmd := exec.Command("bash")
	cmd.Dir = "/"
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Println("PTY start error:", err)
		return
	}
	defer func() {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
	}()

	// SERVER ➜ CLIENT (PTY output → WebSocket)
	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				return
			}
			// IMPORTANT FIX: send as binary
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// CLIENT ➜ SERVER (keystrokes → PTY)
	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// Accept text or binary (browser may send either)
		if mt == websocket.TextMessage || mt == websocket.BinaryMessage {
			_, err = ptmx.Write(msg)
			if err != nil {
				return
			}
		}
	}
}

func access(r *http.Request) bool {
	cookie, err := r.Cookie("SESTERMINAL")
	if err != nil {
		return true
	}
	return cookie.Value != "XXXXXXXXXXXXXXXXXXXXXXXXXX"
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/ws", handleWS)

	port := argument("port")
	if(port==""){
		log.Fatal("[X] server port not defined")
		return
	}

	log.Fatal(http.ListenAndServe(":"+port, nil))
	log.Println("Server started at http://localhost:"+port)
}


func argument(a string) string{
	response := ""
	for _,arg := range os.Args{
		if(strings.HasPrefix(arg, a+"=") ){
			response=strings.Split(arg, "=")[1]
			response=strings.Trim(response, "\"")
		}
	}
	return response
}