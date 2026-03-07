package main

import (
	"log"
	"net/http"
	"time"
	"os/exec"
	"os"
	"strings"
	"unicode"
	"strconv"
	"path/filepath"
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
	return false
	cookie, err := r.Cookie("SESTERMINAL")
	if err != nil {
		return true
	}
	return cookie.Value != "18247829059349869010294823857835893475878123781"
}

func main() {

	
	port := argument("port")
	base := enviroment("KURUYO_BASE")
	if(port==""){
		log.Fatal("[X] server port not defined")
		return
	}

	
	write("port=",port)
	write("base=",base)

	
	
	write(base+"/ws")
	http.HandleFunc(base+"/ws", handleWS)

	
	ROOT := "/web/server/terminal/static"
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqPath := filepath.Clean("/" + r.URL.Path)
		reqPath = strings.TrimPrefix(reqPath, base)
		filePath := filepath.Join(ROOT, reqPath)

		write(filePath)

		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")

		// Path traversal koruması
		if !strings.HasPrefix(filePath, ROOT) {
			http.NotFound(w, r)
			return
		}

		info, err := os.Stat(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Eğer dizinse → index.html ekle
		if info.IsDir() {
			indexPath := filepath.Join(filePath, "index.html")

			indexInfo, err := os.Stat(indexPath)
			if err != nil {
				http.NotFound(w, r)
				return
			}

			filePath = indexPath
			info = indexInfo
		}

		file, err := os.Open(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	})

	

	
	
	table("TERMINAL STARTED:"+port)
	log.Fatal(http.ListenAndServe(":"+port, nil))

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

func enviroment(a string) string{
	return os.Getenv(a)
}

func write(values ...interface{}) {
	originalFlags := log.Flags()
	log.SetFlags(0)
	log.Println(values...)
	log.SetFlags(originalFlags)
}

func center(text string) string {
	const width = 48
	if len(text) >= width {
		return text // return as-is if longer than width
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-len(text)-padding)
}

func table(name string){
	// █
	write("╔════════════════════════════════════════════════╗") // 50 Char
	write("║"+center(name)+"║")
	write("╚════════════════════════════════════════════════╝") // 50 Char
}

func row(name string){
	// █
	write("┌────────────────────────────────────────────────┐") // 50 Char
	write("│"+center(name)+"│")
	write("└────────────────────────────────────────────────┘") // 50 Char
}

func ToNumber(number string) int {
	numStr := ""
	for _, r := range number {
		if unicode.IsDigit(r) {
			numStr += string(r)
		} else {
			break
		}
	}
	if numStr == "" { return 0 }
	num, err := strconv.Atoi(numStr)
	if err != nil { return 0 }
	return num
}
