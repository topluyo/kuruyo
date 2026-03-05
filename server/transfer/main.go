package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
  "strings"
  "unicode"
  "strconv"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}


func uploadHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	uploadDir := "/web/uploads"
	// Klasör yoksa oluştur
	err = os.MkdirAll(uploadDir, os.ModePerm)
	if err != nil {
		log.Println("Failed to create upload directory:", err)
		conn.WriteMessage(websocket.TextMessage, []byte("Server error: cannot create upload directory"))
		return
	}

	var file *os.File
	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		// İlk mesaj dosya adı
		if file == nil {
			filename := filepath.Base(string(msg)) // sadece dosya adı, path değil
			filePath := filepath.Join(uploadDir, filename)

			file, err = os.Create(filePath)
			if err != nil {
				log.Println("File create error:", err)
				conn.WriteMessage(messageType, []byte("Failed to create file"))
				return
			}
			log.Println("Receiving file:", filePath)
			conn.WriteMessage(messageType, []byte("Start sending chunks"))
			continue
		}

		// "EOF" ile dosya tamamlandı
		if string(msg) == "EOF" {
			file.Close()
			log.Println("File upload completed")
			conn.WriteMessage(messageType, []byte("File uploaded successfully"))
			break
		}

		// Chunk yaz
		_, err = file.Write(msg)
		if err != nil {
			log.Println("Write error:", err)
			conn.WriteMessage(messageType, []byte("Failed to write chunk"))
			return
		}
	}
}


func main() {

  table("TRANSFER")
	PORT := argument("port")

	if PORT == "" {
		log.Fatal("port parameters needed: path=8080")
		return
	}

	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)
	http.HandleFunc("/upload", uploadHandler)


	write("Basic Server running on :" + PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))

}






func argument(key string, defaults ...string) string {
	response := ""
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, key+"=") {
			response = strings.SplitN(arg, "=", 2)[1]
			response = strings.Trim(response, "\"")
			return response
		}
	}
	if len(defaults) > 0 {
		return defaults[0]
	}
	return ""
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

