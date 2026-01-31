// main.go
package main

import (
	"context"
	"encoding/json"
	//"flag"
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
	"runtime"
	"syscall"

)
import "golang.org/x/sys/unix"


var cfgPath string
var cfgCert string
var cfgKey string



type Level struct {
	Rate    []string           `json:"rate"`
	IPS     []string           `json:"ips"`
	Token   string             `json:"token"`
}
type Server struct {
	IP      string                       `json:"ip"`
	HTTP    int                          `json:"http"`
	HTTPS   int                          `json:"https"`
	Domains map[string]map[string]*Route
	Levels  map[string]*Level            `json:"levels"`
	Routes map[string]*Route             `json:"routes"`
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
	Ports          string   `json:"ports"`
	RateLimits     []RateLimit
	IPFilter        bool
	IPS            []string
	AllowedIPS     []*net.IPNet

	proxy          *httputil.ReverseProxy

	UseToken       bool
	Token          string


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
		log.Fatal("[x] config file: ",path," not readed")
		return
	}
	if err := json.Unmarshal(json_content, &server); err != nil {
		log.Println("Parsing json")
		log.Fatal(err)
		log.Fatal("[x] config file: ",path," not readed")
		return
	}
}


func ranges(s string) []string {
	s = strings.TrimSpace(s)
	if _, err := strconv.Atoi(s); err == nil {
		return []string{s}
	}

	var a, b int
	if strings.Contains(s, "-") {
		p := strings.SplitN(s, "-", 2)
		ai, e1 := strconv.Atoi(p[0])
		bi, e2 := strconv.Atoi(p[1])
		if e1 != nil || e2 != nil { return []string{} }
		a, b = ai, bi
	} else if strings.Contains(s, "+") {
		p := strings.SplitN(s, "+", 2)
		ai, e1 := strconv.Atoi(p[0])
		n, e2 := strconv.Atoi(p[1])
		if e1 != nil || e2 != nil { return []string{} }
		a, b = ai, ai+n
	} else {
		return []string{}
	}

	if a > b { return []string{} }

	out := make([]string, 0, b-a+1)
	for i := a; i <= b; i++ {
		out = append(out, strconv.Itoa(i))
	}
	return out
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
				//log.Println(r.Backends[i], "🔴",r.backendUp ) 
			}else{
				//log.Println(r.Backends[i], "🟢",r.backendUp ) 
			}
		}
	}
}

//@ Select Back End its upped
func (r *Route) RunningBackEnd() *url.URL {
	n := len(r.Backends)
	if n == 0 {
		return nil
	}
	if n == 1 {
		if len(r.backendUp) == n && !r.backendUp[0] {
			log.Println("Backend is down")
			return nil
		}
		return r.Backends[0]
	}
	start := atomic.AddUint64(&r.rrCounter, 1)
	for i := 0; i < n; i++ {
		idx := int((start + uint64(i)) % uint64(n))
		if len(r.backendUp) == n {
			if r.backendUp[idx] {
				return r.Backends[idx]
			}
			log.Println("Backend", r.Backends[idx], "is down")
			continue
		}
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


//@ AllowedIPS define
func (r *Route) AllowedIPSDefine() {

	ips := make(map[string]string)
	for _, routeLevels := range r.Levels {
    level, ok := server.Levels[routeLevels]
    if ok {
			for _,ip := range level.IPS {
				ips[ip] = ip
			}
    } else {
			log.Println("ERROR: `"+routeLevels+"` level not found in", r.Name)
    }
	}

	for _, ip := range ips {
		_, cidr, err := net.ParseCIDR(ip)
		if err != nil {
			ipAddr := net.ParseIP(ip)
			if ipAddr != nil {
				r.AllowedIPS = append(r.AllowedIPS, &net.IPNet{IP: ipAddr, Mask: net.CIDRMask(32, 32)})
			} else {
				log.Printf("Invalid IP or CIDR: %s", ip)
			}
		} else {
			r.AllowedIPS = append(r.AllowedIPS, cidr)
		}
	}

	if(len(ips)>0){
		r.IPFilter = true
	}else{
		r.IPFilter = false
	}

}

//@ UseTokenDefine define
func (r *Route) UseTokenDefine() {
	token := ""
	for _, routeLevels := range r.Levels {
    level, ok := server.Levels[routeLevels]
    if ok {
			if(level.Token!="") {
				token = level.Token
			}
    } else {
			log.Println("ERROR: `"+routeLevels+"` level not found in", r.Name)
    }
	}
	if(token!=""){
		r.UseToken = true
		r.Token = url.QueryEscape(token)
	}else{
		r.UseToken = false
	}
}

// Check if the client's IP is allowed
func (r *Route) IsAllowedIP(clientIP string) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false // Invalid IP
	}
	for _, cidr := range r.AllowedIPS {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}


func fast(){
	server.Domains = make(map[string]map[string]*Route)
	for name , _ := range server.Routes {
		host := name
		row("installing  " + name )

		//! Burada İşle
		//if( routers )
		write("├─── Check Domain");
		domain := UrlDomain(host)
		path   := UrlPath(host)

		_, ok := server.Domains[domain]
		if(!ok){
			server.Domains[domain] = make(map[string]*Route)
		}
		write("   └──",domain,"/",path)
		server.Domains[domain][path] = server.Routes[name]

		write("├─── Ports");

		if(server.Routes[host].Ports!=""){
			server.Routes[host].Proxies = []string{}
			for  _,port := range ranges(server.Routes[host].Ports) {
				write("   └──",port)
				_url := "http://127.0.0.1:"+port
				server.Routes[host].Proxies = append(server.Routes[host].Proxies , _url)
			}
		}


		write("├─── Proxies");

		write(server.Routes[host].Proxies)
		server.Routes[host].Name = name
		server.Routes[host].Backends = ParseURLs(server.Routes[host].Proxies)
		server.Routes[host].transport = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			MaxIdleConns:          2000,
			MaxIdleConnsPerHost:   1000,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

		server.Routes[host].proxy = &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				backend := server.Routes[host].RunningBackEnd()
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
			Transport:  server.Routes[host].transport,
			ModifyResponse: func(resp *http.Response) error {
				return nil
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				log.Printf("proxy error for %s -> %v", r.Host, err)
				http.Error(w, "upstream error", http.StatusBadGateway)
			},
			BufferPool: ProxyBufferPool{}, // our adapter to use bufPool
		}
		go server.Routes[host].HealthChecker()

		write("\n├─── AllowedIPS")
		server.Routes[host].AllowedIPSDefine()
		write(server.Routes[host].AllowedIPS)
		
		write("\n├─── Levels")
		write(server.Routes[host].Levels)

		write("\n├─── UseToken")
		server.Routes[host].UseTokenDefine()
		write(server.Routes[host].UseToken)

		//server.Routes[host] = route
	}
}



// Main Loop
func listenReusePort(network, address string) (net.Listener, error) {
    lc := net.ListenConfig{
        Control: func(network, address string, c syscall.RawConn) error {
            var err error
            c.Control(func(fd uintptr) {
                err = unix.SetsockoptInt(
                    int(fd),
                    unix.SOL_SOCKET,
                    unix.SO_REUSEPORT,
                    1,
                )
                if err != nil {
                    return
                }
            })
            return err
        },
    }
    return lc.Listen(context.Background(), network, address)
}



func SafeClientIP(r *http.Request) string {
	addr := strings.TrimSpace(r.RemoteAddr)
	if addr == "" {
			return ""
	}

	host, _, err := net.SplitHostPort(addr)
	if err == nil {
			return host
	}

	// If it's IPv6 without port
	if strings.Count(addr, ":") >= 2 {
			return addr
	}

	return addr
}

func run(){
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		host := HostDomain(r.Host)
		path := UrlPath(r.URL.String())

		//@ Domain Yakalama
		domain, ok := server.Domains[host]
		if !ok {
			http.Error(w, "no route configured for host", http.StatusBadGateway)
			return
		}

		//@ Path Yakalama
		var route *Route
		for prefix, _route := range domain {
			if prefix != "" && strings.HasPrefix(path, prefix) {
				route = _route
				break
			}
		}
		if(route==nil){
			if _route, ok := domain[""]; ok {
				route = _route
			}
		}


		ip := SafeClientIP(r)

		if(route==nil){
			log.Println("ERROR WHEN","\t", ip, "\t", "\t", host + r.URL.String() )
			http.Error(w, "no route configured for host", http.StatusBadGateway)
			return	
		}

		

		
		log.Println("\t", ip, "\t", "\t",
				host + func() string {
						if r != nil && r.URL != nil { return r.URL.String() }
						return ""
				}(),
				func() string {
						if route != nil { return route.Name }
						return ""
				}(),
		)



		//log.Println("\t", ip, "\t", "\t", host + r.URL.String(), route.Name )
		
		if(route.IPFilter){
			if !route.IsAllowedIP(ip) {
				w.WriteHeader(http.StatusForbidden)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(FORBIDDEN_HTML)
				//http.Error(w, FORBIDDEN_HTML, http.StatusForbidden)
				return
			}
		}

		// UseToken
		if(route.UseToken){
			cookie, err := r.Cookie("token")
			if(err==nil && cookie.Value==route.Token){
				// pass
			}else{
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(LOGIN_HTML)
				return
			}
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

	//++   HTTP listening
	go func() {
    workers := runtime.NumCPU()
    log.Printf("HTTP SO_REUSEPORT workers: %d", workers)
    for i := 0; i < workers; i++ {
			go func(id int) {
				ln, err := listenReusePort("tcp", httpAddr)
				if err != nil {
					log.Fatalf("HTTP reuseport error: %v", err)
				}
				log.Printf("HTTP worker %d listening on %s", id, httpAddr)
				if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
					log.Fatalf("http worker %d: %v", id, err)
				}
			}(i)
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
		workers := runtime.NumCPU()
		log.Printf("HTTPS SO_REUSEPORT workers: %d", workers)

		for i := 0; i < workers; i++ {
			go func(id int) {
				ln, err := listenReusePort("tcp", httpsAddr)
				if err != nil {
					log.Fatalf("HTTPS reuseport error: %v", err)
				}
				log.Printf("HTTPS worker %d listening on %s", id, httpsAddr)
				if err := srv2.ServeTLS(ln, cfgCert, cfgKey); err != nil && err != http.ErrServerClosed {
					log.Fatalf("https worker %d: %v", id, err)
				}
			}(i)
		}

	} else {
		if server.HTTPS != 0 {
			log.Printf("HTTPS not started: provide -tls-cert and -tls-key to enable TLS on port %d", server.HTTPS)
		}
	}

	// block forever (or implement graceful shutdown)
	select {}
}




func ready(){
	exec.Command("bash", "-c", "ufw allow " + ToString(server.HTTP)).Output()
	exec.Command("bash", "-c", "ufw allow " + ToString(server.HTTPS)).Output()
	exec.Command("bash", "-c", "fuser -k " + ToString(server.HTTP) + "/tcp").Output()
	exec.Command("bash", "-c", "fuser -k " + ToString(server.HTTPS) + "/tcp").Output()
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

var LOGIN_HTML []byte
var FORBIDDEN_HTML []byte


func main(){
	LOGIN_HTML,_ = os.ReadFile("login.html")
	FORBIDDEN_HTML,_ = os.ReadFile("forbidden.html")

	table("KURUYO STARTING")
	
	cfgPath    = argument("config")
	cfgCert    = "origin.pem"
	cfgKey     = "origin.key"

	if(cfgPath==""){
		write("[X] config=XXXX parameter needed.")
		return
	}
	write("[.]",cfgPath,"reading...")
	load(cfgPath)
	
	ready()

	fast()

	run()
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








///+ REQUIREDS
func HostDomain(hostport string) string {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return h
}

func UrlPath(s string) string {
	i := strings.Index(s, "/")
	if i == -1 {
		return ""
	}
	p := s[i+1:]
	return strings.TrimPrefix(p, "/")
}
func UrlDomain(hostport string) string {
	if i := strings.Index(hostport, "/"); i != -1 {
		hostport = hostport[:i]
	}
	return hostport
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