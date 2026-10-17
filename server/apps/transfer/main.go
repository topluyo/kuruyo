package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const htmlContent = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Transfer</title>
    <style>
        body{font-family:Arial,sans-serif;background:#f5f5f5;display:flex;flex-direction:column;align-items:center;justify-content:center;height:100vh;margin:0}
        h1{color:#333;margin-top:0}
        .uploader{background:#fff;padding:2em;border-radius:.5em;box-shadow:0 4px 10px rgba(0,0,0,.1);text-align:center;width:300px}
        input[type="file"]{margin-bottom:1em;width:100%}
        button{padding:.5em 1.5em;border:none;border-radius:.5em;background:#007bff;color:#fff;cursor:pointer;font-size:1em;width:100%;transition:background .2s}
        button:hover{background:#0056b3}
        button:disabled{background:#ccc;cursor:not-allowed}
        #status{margin-top:1em;color:#555;font-size:.9em}
        .progress-container{width:100%;background:#eee;border-radius:.5em;overflow:hidden;margin-top:1em;height:1em}
        .progress-bar{width:0%;height:100%;background:#28a745;transition:width .05s ease-in-out}
    </style>
</head>
<body>
    <div class="uploader">
        <h1>Transfer</h1>
        <input type="file" id="fileInput">
        <button id="uploadBtn" onclick="uploadFile()">Upload</button>
        <div class="progress-container">
            <div class="progress-bar" id="progressBar"></div>
        </div>
        <p id="status"></p>
    </div>

<script>
    const ws = new WebSocket("ws://" + location.hostname + (location.port ? ":"+location.port : "") + "/upload");
    ws.binaryType = "arraybuffer";

    let file, offset, chunkSize, reader, totalSize;
    let isUploading = false;

    ws.onmessage = (event) => {
        const msg = event.data;

        if (msg === "READY" || msg === "ACK") {
            if (offset < totalSize) {
                sendNextChunk();
            } else {
                ws.send("EOF");
            }
        } else if (msg === "SUCCESS") {
            document.getElementById("status").innerText = "Upload complete successfully!";
            isUploading = false;
            document.getElementById("uploadBtn").disabled = false;
        } else if (msg === "FAIL") {
            document.getElementById("status").innerText = "Server write error occurred.";
            isUploading = false;
            document.getElementById("uploadBtn").disabled = false;
        } else {
            document.getElementById("status").innerText = msg;
        }
    };

    function uploadFile() {
        if (isUploading) return;

        const fileInput = document.getElementById("fileInput");
        file = fileInput.files[0];
        if (!file) {
            alert("Please select a file");
            return;
        }

        isUploading = true;
        document.getElementById("uploadBtn").disabled = true;

        const progressBar = document.getElementById("progressBar");
        progressBar.style.width = "0%";
        document.getElementById("status").innerText = "Connecting & starting upload...";

        if (ws.readyState !== WebSocket.OPEN) {
            alert("WebSocket bağlantısı kurulamadı. Lütfen sayfayı yenileyin.");
            isUploading = false;
            document.getElementById("uploadBtn").disabled = false;
            return;
        }

        totalSize = file.size;
        offset = 0;
        chunkSize = 64 * 1024;
        reader = new FileReader();

        reader.onload = (e) => {
            ws.send(e.target.result);
            offset += e.target.result.byteLength;

            const percent = Math.min((offset / totalSize) * 100, 100);
            progressBar.style.width = percent + "%";
            document.getElementById("status").innerText = 'Uploading... ' + Math.round(percent) + '%';
        };

        reader.onerror = (err) => {
            console.error("File read error:", err);
            document.getElementById("status").innerText = "Dosya okunurken bir hata oluştu.";
            isUploading = false;
            document.getElementById("uploadBtn").disabled = false;
        };

        ws.send(file.name);
    }

    function sendNextChunk() {
        const slice = file.slice(offset, offset + chunkSize);
        reader.readAsArrayBuffer(slice);
    }
</script>
</body>
</html>`

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	uploadDir := "./uploads" // Geçerli dizin altında uploads klasörü
	err = os.MkdirAll(uploadDir, os.ModePerm)
	if err != nil {
		log.Println("Failed to create upload directory:", err)
		conn.WriteMessage(websocket.TextMessage, []byte("Server error: cannot create upload directory"))
		return
	}

	var file *os.File
	var bufferedWriter *bufio.Writer
	var totalReceived int64

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		if file == nil {
			filename := filepath.Base(string(msg))
			filePath := filepath.Join(uploadDir, filename)

			file, err = os.Create(filePath)
			if err != nil {
				log.Println("File create error:", err)
				conn.WriteMessage(messageType, []byte("Failed to create file"))
				return
			}
			
			bufferedWriter = bufio.NewWriterSize(file, 64*1024)
			totalReceived = 0

			log.Println("Receiving file:", filePath)
			conn.WriteMessage(websocket.TextMessage, []byte("READY"))
			continue
		}

		if string(msg) == "EOF" {
			if bufferedWriter != nil {
				bufferedWriter.Flush()
			}
			file.Close()
			log.Println("File upload completed. Total bytes:", totalReceived)
			conn.WriteMessage(websocket.TextMessage, []byte("SUCCESS"))
			break
		}

		_, err = bufferedWriter.Write(msg)
		if err != nil {
			log.Println("Write error:", err)
			conn.WriteMessage(websocket.TextMessage, []byte("FAIL"))
			return
		}
		
		totalReceived += int64(len(msg))
		conn.WriteMessage(websocket.TextMessage, []byte("ACK"))
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] %s - %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	table("TRANSFER")
	PORT := argument("port", "8080")

	mux := http.NewServeMux()
	
	// Ana sayfa için HTML sunumu
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(htmlContent))
	})

	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP/WS] %s - %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		uploadHandler(w, r)
	})

	write("Basic Server running on :" + PORT)
	
	// IP adresini bulup start for link formatında konsola yazdıralım
	ip := getLocalIP()
	write("start for link: http://" + ip + ":" + PORT)

	log.Fatal(http.ListenAndServe(":"+PORT, mux))
}

// Yerel IP adresini bulan yardımcı fonksiyon
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}
	for _, address := range addrs {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "localhost"
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
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-len(text)-padding)
}

func table(name string) {
	write("╔════════════════════════════════════════════════╗")
	write("║" + center(name) + "║")
	write("╚════════════════════════════════════════════════╝")
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
	if numStr == "" {
		return 0
	}
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0
	}
	return num
}
