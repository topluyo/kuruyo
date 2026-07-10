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
)
import "golang.org/x/sys/unix"
//! tsl kapatıldı
//import "app/tslguard"


var cfgPath string
var cfgCert string
var cfgPriv string

//@ Server
type Server struct {
	IP             string                       `json:"ip"`
	HTTP           int                          `json:"http"`
	HTTPS          int                          `json:"https"`
	Log            string                       `json:"log"`
	Healt          bool                         `json:"healt"`
	Cert           string                       `json:"cert"`
	Priv           string                       `json:"priv"`
	Domains        map[string]map[string]*Route
	Levels         map[string]*Level            `json:"levels"`
	Routes         map[string]*Route            `json:"routes"`
	BlockedFile    string                       `json:"blocked"`
	RateSIZE       int                          `json:"rateSIZE"`
	Blocked        map[string]string         
}

//! Burası Çalışmıyor!
//@ Server.LoadBlocked
func (s *Server) LoadBlocked() error {
	data, err := os.ReadFile(s.BlockedFile)
	if err != nil {
		return err
	}
	s.Blocked = make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		s.Blocked[line] = "blocked"
	}
	return nil

}

var server Server 


//@ Level
type Level struct {
	Rates   string             `json:"rates"`
	IPS     []string           `json:"ips"`
	Token   string             `json:"token"`
}

//@ Route
type Route struct{
	Name           string
	Proxies        []string `json:"proxies"`
	Levels         []string `json:"levels"`
	Ports          string   `json:"ports"`
	IPFilter        bool
	IPS            []string
	AllowedIPS     []*net.IPNet

	proxy          *httputil.ReverseProxy

	Balancer       int

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



//@ Route.HealthChecker
func (r *Route) HealthChecker() {
	if(server.Healt==false){
		return
	}
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
		req.Header.Set("Host-Path", UrlPath(r.Name))

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
			write(r.Backends[i], "🔴" ) 
		}else{
			write(r.Backends[i], "🟢" ) 
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



//@ Route.RunningBackEnd
func (r *Route) RunningBackEnd(url string) *url.URL {
	n := len(r.Backends)
	if n == 0 {
		return nil
	}
	if n == 1 {
		if server.Healt && len(r.backendUp) == n && !r.backendUp[0] {
			log.Println("Backend is down")
			return nil
		}
		return r.Backends[0]
	}

	if(r.Balancer>1){
		return r.Backends[BalancerHash(url,r.Balancer)]
	}

	start := atomic.AddUint64(&r.rrCounter, 1)
	for i := 0; i < n; i++ {
		idx := int((start + uint64(i)) % uint64(n))
		if(server.Healt==false){
			return r.Backends[idx]
		}
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

//@ Route.DefineUseToken
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

//@ Route.IsAllowedIP
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





var LOGIN_HTML []byte
var FORBIDDEN_HTML []byte
var RATEOVER_HTML []byte


//@ 0. main
func main(){
	LOGIN_HTML,_ = os.ReadFile("login.html")
	FORBIDDEN_HTML,_ = os.ReadFile("forbidden.html")
	RATEOVER_HTML,_ = os.ReadFile("rateover.html")


	Init_CLOUDFLARENETS()
	Init_LogTime()
	

	table("KURUYO STARTING")


	cfgPath    = argument("config")
	cfgCert    = argument("cert", "/web/config/origin.pem")
	cfgPriv    = argument("key",  "/web/config/origin.key")

	if(cfgPath==""){
		write("[X] config=XXXX parameter needed.")
		return
	}
	write("[.]",cfgPath,"reading...")
	load(cfgPath)
	
	

	ready()

	write("|--- Log")
	Init_Log()

	fast()

	run()

	//@ 0.1 reload
	for {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan,syscall.SIGUSR1)
		s := <-sigChan
		if(s==syscall.SIGUSR1){
			table("RELOADING ROUTER")
			write("|--- Loading")
			load(cfgPath)
			write("|--- Log")
			Init_Log()
			write("|--- Fast")
			fast()
		}
	}
}









//@   1. load                     
//    Using on `load` and `reload`
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
	cfgCert = server.Cert
	cfgPriv = server.Priv
}




//@ 2. fast
// kurulum yapıyor burada
func fast(){

	server.LoadBlocked()
	write("├─── Black List");
	write(server.Blocked)
	

	server.Domains = make(map[string]map[string]*Route)
	for name , _ := range server.Routes {
		host := name
		row("installing  " + name )

		write("├─── Check Domain");
		domain := UrlDomain(host)
		path   := UrlPath(host)

		Balancer := 0
		
		path_balancer := strings.Split(path, "%%")
		if(len(path_balancer)>1){
			path = path_balancer[0]
			Balancer = 1
		}

		



		_, ok := server.Domains[domain]
		if(!ok){
			server.Domains[domain] = make(map[string]*Route)
		}
		write("   └──",domain,"/",path)
		server.Domains[domain][path] = server.Routes[name]

		write("├─── Ports");

		if(server.Routes[host].Ports!=""){
			server.Routes[host].Proxies = []string{}
			for  _,port := range Ranges(server.Routes[host].Ports) {
				write("   └──",port)
				_url := "http://127.0.0.1:"+port
				server.Routes[host].Proxies = append(server.Routes[host].Proxies , _url)
			}
		}

		if(Balancer==1){
			Balancer = len(server.Routes[host].Proxies)
			server.Routes[host].Balancer = Balancer
			write("\n├─── Balancer")
			write("  path     :" , path)
			write("  Balancer :" , Balancer)
		}

		write("├─── Proxies");

		write(server.Routes[host].Proxies)
		server.Routes[host].Name = name
		server.Routes[host].Backends = ParseURLS(server.Routes[host].Proxies)
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
				backend := server.Routes[host].RunningBackEnd(req.URL.String())
				if backend == nil {
					return
				}
				req.URL.Scheme = backend.Scheme
				req.URL.Host = backend.Host
				clientIP := GetIP(req)
				req.Header.Set("X-Forwarded-For", clientIP)
				req.Header.Set("Host-Path","/"+path)
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
			BufferPool: PROXY_BUFFER_POOL{}, // our adapter to use BUF_POOL
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
}







//@ 3. run
func run(){
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		host := HostDomain(r.Host)
		path := UrlPath(r.URL.String())
		

		r, ip := SetIP(r)

    if _, ok := server.Blocked[ip]; ok {
			http.Error(w, "bloked", http.StatusBadGateway)
			
			Log( "BLOCKED, " + Log_Now() + ", " + ip + ", " + host + ", "+ r.URL.String()) 
			return
    }
		
		//- Domain Yakalama
		domain, ok := server.Domains[host]
		if !ok {
			http.Error(w, "no route configured for host", http.StatusBadGateway)
			Log( "NOROUTER, " + Log_Now() + ", " + ip + ", " + host + ", "+ r.URL.String()) 
			return
		}

		//- Path Yakalama
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
			Log( "NOROUTER, " + Log_Now() + ", " + ip + ", " + host + ", "+ r.URL.String()) 
			return	
		}

		write(ip, host, r.URL.String())
		if(route.IPFilter){
			if !route.IsAllowedIP(ip) {
				w.WriteHeader(http.StatusForbidden)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(FORBIDDEN_HTML)
				Log( "FORBIDDEN, " + Log_Now() + ", " + ip + ", " + host + ", "+ r.URL.String()) 
				return
			}
		}

		//- UseToken
		if(route.UseToken){
			cookie, err := r.Cookie("token")
		
			if(err==nil && cookie.Value==route.Token){
				// pass
			}else{
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(LOGIN_HTML)
				Log( "LOGIN, " + Log_Now() + ", " + ip + ", " + host + ", "+ r.URL.String()) 
				return
			}
		}
		
		//- UseLimit
		if(route.UseLimit){
			//for _, limit := range route.Limits{
				if( CheckRateLimiter(ip,route.Limits)==false ) {
					w.WriteHeader(http.StatusTooManyRequests)
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.Write(RATEOVER_HTML)
					Log( "RATEOVER, " + Log_Now() + ", " + ip + ", " + host + ", "+ r.URL.String())
					return
				}
			//}
		}

		route.proxy.ServeHTTP(w, r)
		Log( "SUCCESS, " + Log_Now() + ", " + ip + ", " + host + ", "+ r.URL.String()) 		
	})
	
	httpAddr := net.JoinHostPort("", strconv.Itoa(server.HTTP))
	srv := &http.Server{
		Addr:         httpAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // streaming proxies shouldn't set small write timeout
		IdleTimeout:  120 * time.Second,
	}

	//- HTTP listening
	go func() {
    workers := runtime.NumCPU()
    log.Printf("HTTP SO_REUSEPORT workers: %d", workers)
    for i := 0; i < workers; i++ {
			go func(id int) {
				ln, err := ListenReusePort("tcp", httpAddr)
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

	//tslguard.Init()

	//- HTTPS listening
	if cfgCert != "" && cfgPriv != "" && server.HTTPS != 0 {
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
				ln, err := ListenReusePort("tcp", httpsAddr)
				if err != nil {
					log.Fatalf("HTTPS reuseport error: %v", err)
				}
				//ln = tslguard.WrapListener(ln)
				log.Printf("HTTPS worker %d listening on %s", id, httpsAddr)
				if err := srv2.ServeTLS(ln, cfgCert, cfgPriv); err != nil && err != http.ErrServerClosed {
					write("https worker %d: %v", id, err)
					row("HTTPS Cert ERROR")
					write(center("make https:0 in router.json"))
					os.Exit(1)
				}
			}(i)
		}

	} else {
		if server.HTTPS != 0 {
			log.Printf("HTTPS not started: provide -tls-cert and -tls-key to enable TLS on port %d", server.HTTPS)
		}
	}

}



//@ 3.1 Using For High Speed Proxy
var ( BUF_POOL = sync.Pool{ New: func() interface{} { return make([]byte, 32*1024) } })
type PROXY_BUFFER_POOL struct{}
func (PROXY_BUFFER_POOL) Get() []byte {
	return BUF_POOL.Get().([]byte)
}
func (PROXY_BUFFER_POOL) Put(b []byte) {
	BUF_POOL.Put(b)
}


//@ 3.2 ListenReusePort
func ListenReusePort(network, address string) (net.Listener, error) {
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


//@ 4. ready
func ready(){
	if(server.HTTP>0) {
		exec.Command("bash", "-c", "ufw allow " + ToString(server.HTTP)).Output()
	}
	if(server.HTTPS>0) {
		exec.Command("bash", "-c", "ufw allow " + ToString(server.HTTPS)).Output()
	}
	if(server.HTTP>0) {
		exec.Command("bash", "-c", "fuser -k " + ToString(server.HTTP) + "/tcp").Output()
	}
	if(server.HTTPS>0) {
		exec.Command("bash", "-c", "fuser -k " + ToString(server.HTTPS) + "/tcp").Output()
	}
}
