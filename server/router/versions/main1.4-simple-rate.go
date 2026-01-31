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
	"os/signal"
	//"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"runtime"
	"syscall"
	"unicode"
)
import "golang.org/x/sys/unix"


var cfgPath string
var cfgCert string
var cfgKey string



type Level struct {
	Rates   []string           `json:"rates"`
	IPS     []string           `json:"ips"`
	Token   string             `json:"token"`
}
type Server struct {
	IP             string                       `json:"ip"`
	HTTP           int                          `json:"http"`
	HTTPS          int                          `json:"https"`
	Log            string                       `json:"log"`
	Domains        map[string]map[string]*Route
	Levels         map[string]*Level            `json:"levels"`
	Routes         map[string]*Route            `json:"routes"`
}

var server Server 


type Route struct{
	Name           string
	Proxies        []string `json:"proxies"`
	Levels         []string `json:"levels"`
	Ports          string   `json:"ports"`
	IPFilter        bool
	IPS            []string
	AllowedIPS     []*net.IPNet

	proxy          *httputil.ReverseProxy

	Limits         []*Limit
	UseLimit       bool
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

//@ DefineUseToken define
func (r *Route) DefineUseToken() {
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
				clientIP := RealIP(req)
				req.Header.Set("X-Forwarded-For", clientIP)
				/*
				if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
					prior := req.Header.Get("X-Forwarded-For")
					if prior == "" {
						req.Header.Set("X-Forwarded-For", clientIP)
					} else {
						req.Header.Set("X-Forwarded-For", prior+", "+clientIP)
					}
				}
				*/
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
		server.Routes[host].DefineUseToken()
		write(server.Routes[host].UseToken)

		//server.Routes[host] = route
	}

	DefineRateLimit()
	write(limiter)
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



//@ IP Adresi Öğrenme

var cloudflareIPs = []string{
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"172.64.0.0/13",
	"131.0.72.0/22",
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
}

var cloudflareNets []*net.IPNet

func cloudflareNetsInit() {
	for _, cidr := range cloudflareIPs {
		_, n, _ := net.ParseCIDR(cidr)
		cloudflareNets = append(cloudflareNets, n)
	}
}
func isCloudflareFast(ip net.IP) bool {
	for _, n := range cloudflareNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func fastRealIP(r *http.Request) string {
	ipStr, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(ipStr)
	if !isCloudflareFast(ip) {
		return ipStr
	}
	if v := r.Header.Get("CF-Connecting-IP"); v != "" {
		return v
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		if i := strings.IndexByte(v, ','); i != -1 {
			return strings.TrimSpace(v[:i])
		}
		return strings.TrimSpace(v)
	}
	return ipStr
}

type ctxKeyRealIP struct{}

func RealIP(r *http.Request) string {
	if ip, ok := r.Context().Value(ctxKeyRealIP{}).(string); ok {
		return ip
	}
	return ""
}
//===========================================================





func run(){
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		host := HostDomain(r.Host)
		path := UrlPath(r.URL.String())
		ip := fastRealIP(r)
		ctx := context.WithValue(r.Context(), ctxKeyRealIP{}, ip)
		r.WithContext(ctx)

		//@ Domain Yakalama
		domain, ok := server.Domains[host]
		if !ok {
			http.Error(w, "no route configured for host", http.StatusBadGateway)
			Log( "NOROUTER, " + time.Now().Format("2006-01-02 15:04:05") + ", " + ip + ", " + host + ", "+ r.URL.String()) 
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


		if(route==nil){
			http.Error(w, "no route configured for host", http.StatusBadGateway)
			Log( "NOROUTER, " + time.Now().Format("2006-01-02 15:04:05") + ", " + ip + ", " + host + ", "+ r.URL.String()) 
			return	
		}

		
		//write(route)

		
		write(ip, host, r.URL.String())

		

		/*
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
		*/



		//log.Println("\t", ip, "\t", "\t", host + r.URL.String(), route.Name )
		
		if(route.IPFilter){
			if !route.IsAllowedIP(ip) {
				w.WriteHeader(http.StatusForbidden)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(FORBIDDEN_HTML)
				Log( "FORBIDDEN, " + time.Now().Format("2006-01-02 15:04:05") + ", " + ip + ", " + host + ", "+ r.URL.String()) 
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
				Log( "LOGIN, " + time.Now().Format("2006-01-02 15:04:05") + ", " + ip + ", " + host + ", "+ r.URL.String()) 
				return
			}
		}

		if(route.UseLimit){
			for _, limit := range route.Limits{
				if( CheckLimit(limit,ip)==false ) {
					w.WriteHeader(http.StatusForbidden)
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.Write(RATEOVER_HTML)
					Log( "RATEOVER, " + time.Now().Format("2006-01-02 15:04:05") + ", " + ip + ", " + host + ", "+ r.URL.String())
					return
				}
			}
		}


		route.proxy.ServeHTTP(w, r)
		Log( "SUCCESS, " + time.Now().Format("2006-01-02 15:04:05") + ", " + ip + ", " + host + ", "+ r.URL.String()) 
		
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
var RATEOVER_HTML []byte


func main(){
	LOGIN_HTML,_ = os.ReadFile("login.html")
	FORBIDDEN_HTML,_ = os.ReadFile("forbidden.html")
	RATEOVER_HTML,_ = os.ReadFile("rateover.html")

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
	
	cloudflareNetsInit()

	ready()

	write("|--- Log")
	InitLog()

	fast()

	run()

	/*
	manager.StartCleanup(
		2*time.Minute, // pasif IP sil
		10*time.Second,  // cleanup interval
	)
	*/


	
	for {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan,syscall.SIGUSR1)
		s := <-sigChan
		if(s==syscall.SIGUSR1){
			table("RELOADING ROUTER")
			write("|--- Loading")
			load(cfgPath)
			write("|--- Log")
			InitLog()
			write("|--- Fast")
			fast()
		}
	}

	


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





//========================================
// LOG Sistemi
//========================================
var (
	logChan chan string
	logMu   sync.Mutex
	stopLog chan struct{}
)

func InitLog() {
	logMu.Lock()
	defer logMu.Unlock()

	// önce eski logger'ı durdur
	if stopLog != nil {
		close(stopLog)
	}

	logChan = make(chan string, 1024)
	stopLog = make(chan struct{})

	go func(ch chan string, stop chan struct{}) {
		f, err := os.OpenFile(server.Log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			write("[X] Log File Doesnt Readed", server.Log)
			return
		}
		defer f.Close()

		for {
			select {
			case msg := <-ch:
				if _, err := f.WriteString(msg + "\n"); err != nil {
					// disk full vs.
				}
			case <-stop:
				return
			}
		}
	}(logChan, stopLog)
}

func Log(msg string) {
	logMu.Lock()
	ch := logChan
	logMu.Unlock()

	if ch == nil {
		return
	}

	select {
	case ch <- msg:
	default:
		// doluysa düşür
	}
}
//----------------------------------------






//========================================
// Rate Limit Sistemi (High Performance)
//========================================

type RateLimitParameters struct {
	Request       int
	Second        int
	Wait          int
	IPControllers map[string]*IPController
}

type IPController struct {
	WindowStart int64
	BlockedTime int64
	RunCount    int32
	LastSeen    int64
}

type Limit struct {
	Parameters []RateLimitParameters
}

var limiter map[string]*Limit

func DefineRateLimit() {
	table("RateLimit")
	limiter = make(map[string]*Limit)

	for key, level := range server.Levels {
		rates := level.Rates
		if len(rates) > 0 {
			write("\n├─── RateLimit : " + key)

			limiter[key] = &Limit{
				Parameters: make([]RateLimitParameters, 0),
			}

			for _, rate := range rates {
				rateParam := strings.Split(rate, " ")
				if len(rateParam) < 3 {
					write("[X] Rate parameters error on \"" + rate + "\"")
					break
				}

				request := ToNumber(rateParam[0])
				second := ToNumber(rateParam[1])
				wait := ToNumber(rateParam[2])

				limiter[key].Parameters = append(
					limiter[key].Parameters,
					RateLimitParameters{
						Request:       request,
						Second:        second,
						Wait:          wait,
						IPControllers: make(map[string]*IPController),
					},
				)

				write("   └──", ToString(request)+"r", ToString(second)+"s", ToString(wait)+"w")
			}
		}
	}

	write("\n")

	for name := range server.Routes {
		r := server.Routes[name]
		for _, level := range r.Levels {
			if limit, ok := limiter[level]; ok {
				write("├─── Install RateLimit(" + name + " " + level + ")")
				r.Limits = append(r.Limits, limit)
				r.UseLimit = true
			}
		}
	}
}

func RateTime(now int64, period int) int64 {
	return now / int64(period)
}


func CheckLimit(l *Limit, ip string) bool {
	now := time.Now().Unix()

	for i := range l.Parameters {
		p := &l.Parameters[i]

		ctrl, ok := p.IPControllers[ip]
		if !ok {
			ctrl = &IPController{}
			p.IPControllers[ip] = ctrl
		}

		// IP bloklu mu?
		if ctrl.BlockedTime > now {
			return false
		}

		currentWindow := RateTime(now, p.Second)

		// Yeni window
		if ctrl.WindowStart != currentWindow {
			ctrl.WindowStart = currentWindow
			ctrl.RunCount = 0
		}

		ctrl.RunCount++

		// Limit aşıldı mı?
		if int(ctrl.RunCount) > p.Request {
			ctrl.BlockedTime = now + int64(p.Wait)
			return false
		}

		ctrl.LastSeen = now
	}

	return true
}

//----------------------------------------