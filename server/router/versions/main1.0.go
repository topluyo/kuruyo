// main.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	//"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	//"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)


var cfgPath string
var cfgCert string
var cfgKey string

type Server struct {
	IP      string           `json:"ip"`
	HTTP    int              `json:"http"`
	HTTPS   int              `json:"https"`
	Routes  map[string]Route `json:"routes"`
}
var server Server 


type RateLimit struct{
	Request int
	Second  int
	Wait    int
}

type Route struct{
	Name           string
	Proxies        []string `json:"proxies"`
	Levels         []string `json:"levels"`
	RateLimits     []RateLimit
	IPS            []string
	proxy          *httputil.ReverseProxy


	// Load balancer için gerekli alanlar:
	Backends    []*url.URL   // backend listesi
	transport   *http.Transport
	backendUp   []bool       // health check sonuçları
	rrCounter   uint64       // round-robin sayacı
}

func load(path string){
	json_content, err := os.ReadFile(path)
	if err != nil { 
		log.Fatal(err)
		log.Fatal("config file: ",path," not readed")
		return
	}
	if err := json.Unmarshal(json_content, &server); err != nil {
		log.Fatal(err)
		return
	}
}





//@ Check is Backend UP
func (r *Route) HealthChecker() {
	n := len(r.Backends)
	r.backendUp = make([]bool, n)
	checkOnce := func(idx int) bool {
		u := r.Backends[idx]
		checkURL := *u
		checkURL.Path = "/"
		client := &http.Client{
			Transport: r.transport,
			Timeout:   3 * time.Second,
		}
		req, _ := http.NewRequestWithContext(context.Background(), "HEAD", checkURL.String(), nil)
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return resp.StatusCode < 500
	}
	for i := 0; i < n; i++ {
		ok := checkOnce(i)
		r.backendUp[i] = ok
		if(ok==false){ 
			log.Println(r.Backends[i], "🔴",r.backendUp ) 
		}else{
			log.Println(r.Backends[i], "🟢",r.backendUp ) 
		}
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		for i := 0; i < n; i++ {
			ok := checkOnce(i)
			r.backendUp[i] = ok
			if(ok==false){ 
				log.Println(r.Backends[i], "🔴",r.backendUp ) 
			}else{
				log.Println(r.Backends[i], "🟢",r.backendUp ) 
			}
		}
	}
}

//@ Select Back End its upped
func (r *Route) RunningBackEnd() *url.URL {
	n := len(r.Backends)
	if n == 0 {
		log.Println("No backends available")
		return nil
	}
	if n == 1 {
		if len(r.backendUp) == n && !r.backendUp[0] {
			log.Println("Backend is down")
			return nil
		}
		log.Println("Using single backend:", r.Backends[0])
		return r.Backends[0]
	}
	start := atomic.AddUint64(&r.rrCounter, 1)
	for i := 0; i < n; i++ {
		idx := int((start + uint64(i)) % uint64(n))
		if len(r.backendUp) == n {
			if r.backendUp[idx] {
				log.Println("Using backend:", r.Backends[idx])
				return r.Backends[idx]
			}
			log.Println("Backend", r.Backends[idx], "is down")
			continue
		}
		log.Println("Using backend:", r.Backends[idx])
		return r.Backends[idx]
	}
	log.Println("No available backend")
	return nil
}





//@ Using For High Speed Proxy
var ( bufPool = sync.Pool{ New: func() interface{} { return make([]byte, 32*1024) } })
type ProxyBufferPool struct{}
func (ProxyBufferPool) Get() []byte {
	return bufPool.Get().([]byte)
}
func (ProxyBufferPool) Put(b []byte) {
	bufPool.Put(b)
}

func fast(){
	for domain , route := range server.Routes {
		route.Name = domain
		route.Backends = ParseURLs(route.Proxies)
		route.transport = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			MaxIdleConns:          2000,
			MaxIdleConnsPerHost:   1000,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

		route.proxy = &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				backend := route.RunningBackEnd()
				if backend == nil {
					return
				}
				req.URL.Scheme = backend.Scheme
				req.URL.Host = backend.Host
				if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
					prior := req.Header.Get("X-Forwarded-For")
					if prior == "" {
						req.Header.Set("X-Forwarded-For", clientIP)
					} else {
						req.Header.Set("X-Forwarded-For", prior+", "+clientIP)
					}
				}
				if req.TLS != nil {
					req.Header.Set("X-Forwarded-Proto", "https")
				} else {
					req.Header.Set("X-Forwarded-Proto", "http")
				}
			},
			Transport:  route.transport,
			ModifyResponse: func(resp *http.Response) error {
				return nil
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				log.Printf("proxy error for %s -> %v", r.Host, err)
				http.Error(w, "upstream error", http.StatusBadGateway)
			},
			BufferPool: ProxyBufferPool{}, // our adapter to use bufPool
		}
		go route.HealthChecker()
		server.Routes[domain] = route
		log.Println(domain , "\t→", route.Proxies)
	}
}




func run(){
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		host := domain(r.Host)
		route, ok := server.Routes[host]
		if !ok {
			http.Error(w, "no route configured for host", http.StatusBadGateway)
			return
		}
		route.proxy.ServeHTTP(w, r)
	})
	
	httpAddr := net.JoinHostPort("", strconv.Itoa(server.HTTP))
	srv := &http.Server{
		Addr:         httpAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // streaming proxies shouldn't set small write timeout
		IdleTimeout:  120 * time.Second,
	}

	// HTTP listening
	go func() {
		log.Printf("HTTP  listening on %s", httpAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	// HTTPS listening
	if cfgCert != "" && cfgKey != "" && server.HTTPS != 0 {
		httpsAddr := net.JoinHostPort("", strconv.Itoa(server.HTTPS))
		srv2 := &http.Server{
			Addr:         httpsAddr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 0,
			IdleTimeout:  120 * time.Second,
		}
		log.Printf("HTTPS listening on %s", httpsAddr)
		if err := srv2.ListenAndServeTLS(cfgCert, cfgKey); err != nil && err != http.ErrServerClosed {
			log.Fatalf("https server: %v", err)
		}
	} else {
		if server.HTTPS != 0 {
			log.Printf("HTTPS not started: provide -tls-cert and -tls-key to enable TLS on port %d", server.HTTPS)
		}
	}

	// block forever (or implement graceful shutdown)
	//select {}
}




func ready(){
	exec.Command("bash", "-c", "ufw allow " + ToString(server.HTTP)).Output()
	exec.Command("bash", "-c", "ufw allow " + ToString(server.HTTPS)).Output()
	exec.Command("bash", "-c", "fuser -k " + ToString(server.HTTP) + "/tcp").Output()
	exec.Command("bash", "-c", "fuser -k " + ToString(server.HTTPS) + "/tcp").Output()
}




func main(){
	cfgPath    = *flag.String("config", "config.json", "path to config.json")
	cfgCert    = *flag.String("cert",   "origin.pem", "TLS certificate file (optional)")
	cfgKey     = *flag.String("key",    "origin.key", "TLS key file (optional)")
	flag.Parse()

	load(cfgPath)
	
	ready()

	fast()

	run()
}













///+ REQUIREDS
func domain(hostport string) string {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return h
}

func ToString(number int) string{
  return strconv.Itoa(number)
}

func ParseURLs(list []string) []*url.URL {
	out := make([]*url.URL, 0, len(list))
	for _, s := range list {
		u, err := url.Parse(s)
		if err != nil {
			if !strings.Contains(s, "://") {
				if u2, err2 := url.Parse("http://" + s); err2 == nil {
					u = u2
				} else {
					log.Printf("bad backend url %q: %v", s, err)
					continue
				}
			} else {
				log.Printf("bad backend url %q: %v", s, err)
				continue
			}
		}
		out = append(out, u)
	}
	return out
}