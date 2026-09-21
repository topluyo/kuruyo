package main


import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	//"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)


type Route struct {
	Backend string

	CertFile string
	KeyFile  string

	Proxy *httputil.ReverseProxy
}


type CertificateStore struct {
	mu    sync.RWMutex
	certs map[string]*tls.Certificate
	routes map[string]Route
}

func NewCertificateStore(routes map[string]Route) *CertificateStore {
	return &CertificateStore{
		certs:  make(map[string]*tls.Certificate),
		routes: routes,
	}
}

func (cs *CertificateStore) Reload() error {
	newCerts := make(map[string]*tls.Certificate)

	for domain, route := range cs.routes {
		cert, err := tls.LoadX509KeyPair(
			route.CertFile,
			route.KeyFile,
		)

		if err != nil {
			return errors.New(
				"failed loading certificate for " + domain + ": " + err.Error(),
			)
		}

		newCerts[domain] = &cert
	}

	cs.mu.Lock()
	cs.certs = newCerts
	cs.mu.Unlock()

	slog.Info("TLS certificates reloaded", "count", len(newCerts))

	return nil
}

func (cs *CertificateStore) GetCertificate(hello *tls.ClientHelloInfo,) (*tls.Certificate, error) {
	domain := strings.ToLower(hello.ServerName)
	cs.mu.RLock()
	cert := cs.certs[domain]
	cs.mu.RUnlock()

	if cert == nil {
		return nil, errors.New("no certificate for " + domain)
	}

	return cert, nil
}

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {
		return r.RemoteAddr
	}

	return host
}



//@ Server
func serverHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "Altay")
		next.ServeHTTP(w, r)
	})
}

//@ Gzip

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter  *gzip.Writer
	wroteHeader bool
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}

	w.wroteHeader = true

	// Body olmayan response'larda gzip header koyma.
	if statusCode == http.StatusNoContent ||
		statusCode == http.StatusNotModified {
		w.ResponseWriter.WriteHeader(statusCode)
		return
	}

	w.Header().Del("Content-Length")
	w.Header().Del("Content-Encoding")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Add("Vary", "Accept-Encoding")

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(p))
		}

		w.WriteHeader(http.StatusOK)
	}

	return w.gzipWriter.Write(p)
}

func (w *gzipResponseWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	_ = w.gzipWriter.Flush()

	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Upgrade / WebSocket
		if r.Header.Get("Upgrade") != "" {
			next.ServeHTTP(w, r)
			return
		}

		// Client gzip desteklemiyorsa normal response.
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Backend'in gönderdiği gzip'i kaldır.
		// Böylece double-gzip olmaz.
		r.Header.Del("Accept-Encoding")

		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			gzipWriter:     gz,
		}

		next.ServeHTTP(gzw, r)
	})
}






var PORT string
var client *UnixSocketClient

func main() {

	/*
	limiter := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    2,
		window:   time.Second * 10,
	}
	*/

	BOZKURT_Init("/sys/fs/bpf/map_block")




	cache := NewCache()

	PORT = argument("port","0")
	if(PORT=="0"){
		write("[X] port is not defined")
		return
	}

	routes := map[string]Route{
		"example.com": {
			Backend: "http://XXX.XXX.XXX.XXX:8080",
			CertFile: "/etc/letsencrypt/live/example.com/fullchain.pem",
			KeyFile:  "/etc/letsencrypt/live/example.com/privkey.pem",
		},
	}

	/*
		Her backend için ReverseProxy oluştur.
	*/

	//@ Proxy
	
	client = UnixSocketClientInit("/web/sockets/erlik.sock")
	for domain, route := range routes {
		target, err := url.Parse(route.Backend)
		if err != nil {
			slog.Error("invalid backend","domain", domain,"backend", route.Backend,"error", err,)
			os.Exit(1)
		}

		proxy := &httputil.ReverseProxy{
			Rewrite: func(pr *httputil.ProxyRequest) {
				originalHost := pr.In.Host
				pr.SetURL(target)
				pr.Out.Host = originalHost
				
				// X-Forwarded-* header'larını ayarla.
				pr.SetXForwarded()
				clientIP, _, err := net.SplitHostPort(pr.In.RemoteAddr)
				if err == nil {
					pr.Out.Header.Set("X-Forwarded-For", clientIP)
					pr.Out.Header.Set("X-Real-IP", clientIP)
				}
			},
			ModifyResponse: func(resp *http.Response) error {				
				if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
					host := resp.Request.Header.Get("X-Forwarded-For")
					BOZKURT_Drop(host, 20)
					//go write(client.Request("DROP," + host + ",20"))
				}
        drop := resp.Header.Get("Asena-Drop")
				if(drop!=""){
					host := resp.Request.Header.Get("X-Forwarded-For")
					BOZKURT_Drop(host, uint64(ToNumber(drop)) )
					//go write(client.Request("DROP,"+host+","+drop))
				}
        pass := resp.Header.Get("Asena-Pass")
				if(pass!=""){
					host := resp.Request.Header.Get("X-Forwarded-For")
					BOZKURT_Pass(host, uint64(ToNumber(drop)) )
					//go write(client.Request("PASS,"+host+","+pass))
				}
        return nil
    	},
			Transport: &CacheTransport{
				Transport: http.DefaultTransport,
				Cache:     cache,
			},
			ErrorHandler: func(w http.ResponseWriter,r *http.Request,err error,) {
				slog.Error("backend error","host", r.Host,"error", err,)
				http.Error(w,"bad gateway",http.StatusBadGateway,)
			},
		}

		route.Proxy = proxy
		routes[domain] = route
	}

	/*
		TLS certificate store.
	*/

	certStore := NewCertificateStore(routes)
	if err := certStore.Reload(); err != nil {
		slog.Error("failed loading certificates", "error", err)
		os.Exit(1)
	}

	/*
		TLS config.
		Client hangi domain'e bağlanıyorsa
		SNI üzerinden doğru certificate seçiliyor.
	*/

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: certStore.GetCertificate,
		NextProtos: []string{"h2","http/1.1",},
	}


	handler := http.HandlerFunc(func(w http.ResponseWriter,r *http.Request,) {

		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.ToLower(host)
		clientIP := getClientIP(r)
		route, exists := routes[host]

		if !exists {
			http.Error(w,"unknown host",http.StatusNotFound,)
			return
		}


		key := host + ":" + r.URL.RequestURI()
		/*
		if( key == "topluyo.com:/72172bed29ef69851ecc9d4820ad2aec/sohbet"){
			BOZKURT_Drop(clientIP,90000)
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			//write(client.Request("DROP,"+clientIP+",90000"))
			return
		}
		*/
	



		//@ Chache:Send
		if r.Method == http.MethodGet {
			
			
			if item, ok := cache.Get(key); ok {
				
				/*
				if !limiter.Allow(clientIP) {
					write("RATE ERROR")
					go write(client.Request("DROP,"+clientIP+",90"))
				}
				*/

				for k, values := range item.Header {
					for _, v := range values {
						w.Header().Add(k, v)
					}
				}
				w.WriteHeader(item.Status)
				w.Write(item.Body)
				return
			}
		}

		fmt.Printf("%-8s,%-19s,%-15s,%s\n",r.Method,time.Now().Format("2006-01-02 15:04:05"),clientIP,key,)
		route.Proxy.ServeHTTP(w, r)
	})

	server := &http.Server{
		Handler: serverHeaderMiddleware(gzipMiddleware(handler)),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadTimeout: 60 * time.Second,
		WriteTimeout: 60 * time.Second,
		ErrorLog: log.New(tlsErrorLogger{},"",0,),
	}

	listener, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	tlsListener := tls.NewListener(
		listener,
		tlsConfig,
	)

	/*
		Certbot renew sonrasında sertifikaları
		otomatik olarak tekrar yükle.

		Bu örnekte 5 dakikada bir kontrol ediyoruz.
	*/

	//@ AutoRenewSSL
	/*
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if err := certStore.Reload(); err != nil {
				slog.Error("certificate reload failed","error", err,)
			}
		}
	}()
	*/

	/*
		Graceful shutdown.
	*/

	stop := make(chan os.Signal, 1)
	signal.Notify(stop,os.Interrupt,syscall.SIGTERM,)

	go func() {
		slog.Info("reverse proxy started", "addr", ":"+PORT)

		if err := server.Serve(tlsListener); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error","error", err,)
			os.Exit(1)
		}
	}()
	<-stop
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(),10*time.Second,)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("shutdown error","error", err,)
	}
}