package main

import (
	"log"
	"net"
	"strings"
	"time"
	"github.com/valyala/fasthttp"
)

var allows = map[string]struct{}{
	"topluyo.com":        {},
	"sigma.topluyo.com":  {},
	"status.topluyo.com": {},
}

const banDuration = 900

// ============================================================
// HTTP REQUEST HANDLER
// ============================================================

func redirectToHTTPS(ctx *fasthttp.RequestCtx) {
	host := strings.ToLower(string(ctx.Host()))

	// Host:port ayrıştır
	if hostName, _, err := net.SplitHostPort(host); err == nil {
		host = hostName
	} else {
		// Normal hostname için fallback
		if i := strings.LastIndex(host, ":"); i != -1 {
			host = host[:i]
		}
	}

	host = strings.TrimPrefix(host, "www.")

	// Whitelist kontrolü
	if _, ok := allows[host]; !ok {
		ip := remoteIP(ctx)

		log.Printf(
			"BLOCK host=%q ip=%s",
			host,
			ip,
		)

		Drop(ip, banDuration)
		return
	}

	targetURL := "https://" + host + string(ctx.URI().RequestURI())

	log.Printf(
		"REDIRECT host=%q -> %s",
		host,
		targetURL,
	)

	ctx.Redirect(
		targetURL,
		fasthttp.StatusTemporaryRedirect,
	)
}

// ============================================================
// REMOTE IP
// ============================================================

func remoteIP(ctx *fasthttp.RequestCtx) string {
	addr := ctx.RemoteAddr().String()

	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host
	}

	return addr
}

// ============================================================
// FASTHTTP LOGGER
// ============================================================

// fasthttp request header'larını parse edemeden önce hata oluşursa
// redirectToHTTPS() çalışmaz.
//
// Bu nedenle parser hatasını Logger üzerinden yakalıyoruz.

type securityLogWriter struct{}

func (w securityLogWriter) Write(p []byte) (int, error) {
	msg := string(p)

	// Sadece request header parse hatalarını yakala.
	if strings.Contains(msg, "error when reading request headers") {

		log.Printf(
			"HTTP HEADER ERROR: %s",
			strings.TrimSpace(msg),
		)

		// Log içerisinden saldırgan/client IP'sini çıkar.
		ip := extractClientIP(msg)

		if ip == "" {
			log.Printf(
				"WARNING: client IP could not be extracted",
			)

			return len(p), nil
		}

		log.Printf(
			"HEADER ATTACK DETECTED ip=%s action=DROP duration=%ds",
			ip,
			banDuration,
		)

		// Tek seferde DROP
		Drop(ip, banDuration)

		return len(p), nil
	}

	// Diğer fasthttp logları
	log.Print(msg)

	return len(p), nil
}

// ============================================================
// CLIENT IP EXTRACTION
// ============================================================


func extractClientIP(msg string) string {
	const separator = `"<->"`

	start := strings.Index(msg, separator)

	if start == -1 {
		return ""
	}

	start += len(separator)

	end := strings.IndexByte(msg[start:], '"')

	if end == -1 {
		return ""
	}

	addr := msg[start : start+end]

	host, _, err := net.SplitHostPort(addr)

	if err != nil {
		return ""
	}

	// Gerçek bir IP olduğundan emin ol.
	if net.ParseIP(host) == nil {
		return ""
	}

	return host
}

// ============================================================
// MAIN
// ============================================================

func main() {
	port := argument("port", "")

	if port == "" {
		return
	}

	table("AKAY:" + port)

	InitDrop()

	server := &fasthttp.Server{
		Handler: redirectToHTTPS,

		Logger: log.New(
			securityLogWriter{},
			"",
			0,
		),

		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf(
		"HTTP server listening on :%s",
		port,
	)

	if err := server.ListenAndServe(":" + port); err != nil {
		log.Fatalf(
			"HTTP sunucu hatası: %s",
			err,
		)
	}
}
