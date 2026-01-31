package main

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"strconv"
)

// ================= CONFIG =================
const (
	MAX_CACHE_AGE = 31536000 // 1 year
	MAX_RAM_MB    = 256      // RAM limit
)

// ================= STRUCTS =================
type CacheItem struct {
	Data     []byte
	GzipData []byte
	Mime     string
	Size     int64
	Hash     string
	LastUsed time.Time
}

var (
	cache     = make(map[string]*CacheItem)
	cacheSize int64
	mu        sync.RWMutex
)

// ================= MAIN =================
func main() {
	
	row("STATIC SERVER")

	PORT := argument("port")
	ROOT := argument("root")
	BASE := argument("base","")

	if PORT == "" || ROOT == "" {
		log.Fatal("usage: port=8080 root=/path")
	}

	if( !strings.HasPrefix(BASE,"/") ){
		BASE = "/" + BASE
	}

	
	write("├─── PORT: ", PORT);
	write("├─── ROOT: ", ROOT);
	write("├─── BASE: ", BASE);




	ROOT = filepath.Clean(ROOT)
	preload(ROOT)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path  := filepath.Clean(r.URL.Path)
		path   = strings.TrimPrefix(path, BASE)

		mu.RLock()
		item, ok := cache[path]
		mu.RUnlock()

		if !ok {
			http.NotFound(w, r)
			return
		}

		item.LastUsed = time.Now()

		// Headers
		w.Header().Set("Content-Type", item.Mime)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("ETag", item.Hash)

		// Gzip
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && len(item.GzipData) > 0 {
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(item.GzipData)
			return
		}

		w.Write(item.Data)
	})

	log.Println("RAM Static CDN running on :" + PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}

// ================= PRELOAD =================
func preload(root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(path, root)
		data, _ := os.ReadFile(path)

		if exceed(int64(len(data))) {
			return nil
		}

		h := sha1.Sum(data)
		hash := hex.EncodeToString(h[:])

		mimeType := mime.TypeByExtension(filepath.Ext(path))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		gz := gzipData(data)

		mu.Lock()
		cache[rel] = &CacheItem{
			Data:     data,
			GzipData: gz,
			Mime:     mimeType,
			Size:     int64(len(data)),
			Hash:     hash,
			LastUsed: time.Now(),
		}
		cacheSize += int64(len(data))
		mu.Unlock()

		return nil
	})

	log.Printf("Preloaded %d files | RAM %.2f MB\n", len(cache), float64(cacheSize)/1024/1024)
}

// ================= HELPERS =================
func gzipData(b []byte) []byte {
	var buf strings.Builder
	gz := gzip.NewWriter(&buf)
	gz.Write(b)
	gz.Close()
	return []byte(buf.String())
}

func exceed(add int64) bool {
	return (cacheSize+add)/1024/1024 > MAX_RAM_MB
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

