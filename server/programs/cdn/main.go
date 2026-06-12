package main

import (
	//"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"strconv"
)



// ---------- MAIN ----------
func main() {

	table("BASIC HTTP")

	PORT := argument("port")
	ROOT := argument("root")
	BASE := argument("base","")

	if PORT == "" {
		log.Fatal("port parameters needed: path=8080")
		return
	}
	if ROOT == "" {
		log.Fatal("root parameters needed: root=/path")
		return
	}
	if( !strings.HasPrefix(BASE,"/") ){
		BASE = "/" + BASE
	}
	
	row("CDN SERVER")
	write("├─── PORT: ", PORT);
	write("├─── ROOT: ", ROOT);
	write("├─── BASE: ", BASE);
	
	


	ROOT = filepath.Clean(ROOT)
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqPath := filepath.Clean("/" + r.URL.Path)
		reqPath = strings.TrimPrefix(reqPath, BASE)
		filePath := filepath.Join(ROOT, reqPath)

		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")

		write(reqPath)

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
		
		// ---------- MIME ----------
		ext := filepath.Ext(filePath)
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")

		
		// ---------- HEADERS (MAX CACHE) ----------
		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Expires", "Fri, 31 Dec 9999 23:59:59 GMT")
		w.Header().Set("Accept-Ranges", "bytes")

		// ETag kapat (immutable dosya için gereksiz)
		w.Header().Del("ETag")


		// OPTIONS preflight isteğini yakala
		if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
		}


		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	})




	write("CDN Server running on :" + PORT)
	write("Root:", ROOT)
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




