package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// IP ve Port kontrolü yapan ve transferleri loglayan middleware
func securityMiddleware(allowedIP string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Forbidden: Invalid Remote Address", http.StatusForbidden)
			return
		}

		// Belirtilen IP adresi (veya lokal testler için localhost) kontrolü
		if host != allowedIP && host != "127.0.0.1" && host != "::1" {
			fmt.Printf("Yetkisiz IP denemesi: %s\n", host)
			http.Error(w, "Forbidden: IP Not Allowed", http.StatusForbidden)
			return
		}

		// Tüm dosya transferlerini (istepleri) göster
		fmt.Printf("[TRANSFER] IP: %s -> %s\n", host, r.URL.Path)

		next.ServeHTTP(w, r)
	})
}

func main() {
	table("MOVER")

	if len(os.Args) < 2 || !strings.HasPrefix(os.Args[1], "ip=") {
		name := os.Args[0]
		write("Write on remote: \n  " + name + " ip=" + getLocalIP() + ":3003")
		return
	}

	argVal := strings.TrimPrefix(os.Args[1], "ip=")
	allowedIP, port, err := net.SplitHostPort(argVal)
	if err != nil {
		fmt.Printf("Hata: Geçersiz IP ve Port formatı (%s). Örnek format: 192.168.1.50:8040\n", argVal)
		return
	}

	domain := "http://" + getLocalIP() + ":" + port + "/"

	// Share klasörünün mutlak (absolute) yolunu bul
	absSharePath, err := filepath.Abs("./share")
	if err != nil {
		absSharePath = "./share" // Hata olursa göreceli yolu bırak
	}

	write("Access IP:\n\n  " + allowedIP + "\n")
	write("Server Root:\n\n  " + domain + "\n")
	write("Share Path:\n\n  " + absSharePath + "\n") // Klasör yolu konsola yazdırılıyor

	write("Bash:\n\n  curl " + domain + "/file.zip > file.zip \n") // Klasör yolu konsola yazdırılıyor

	write("Files:\n")

	// share klasöründeki dosyaları listele ve yazdır
	entries, err := os.ReadDir("./share")
	if err != nil {
		_ = os.MkdirAll("./share", 0755)
		write("  (Bilgi: ./share klasörü boş veya yeni oluşturuldu)")
	} else {
		for _, entry := range entries {
			if !entry.IsDir() {
				write("  " + domain + entry.Name())
			}
		}
	}
	write("") // Boşluk bırakmak için

	listenAddr := ":" + port
	fileServer := http.FileServer(http.Dir("./share"))

	http.Handle("/", securityMiddleware(allowedIP, fileServer))

	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		fmt.Printf("Sunucu başlatılamadı: %v\n", err)
		os.Exit(1)
	}
}

// Curl ile gelen veya herhangi bir URL'den çekilen dosyayı kaydetme metodu
func downloadFile(fileURL string, outputFilePath string) error {
	resp, err := http.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sunucu hata kodu döndürdü: %s", resp.Status)
	}

	dir := filepath.Dir(outputFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	out, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("Dosya başarıyla kaydedildi: %s\n", outputFilePath)
	return nil
}

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
