package shield

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type IoTConfig struct {
	ScoreThreshold int    // Ban skoru eşiği
	BanDuration    int    // Ban süresi saniye
	IPSetName      string // ipset adı
	OnDetection    func(ip string, score int, reasons []string)
}

type IoTDetector struct {
	config IoTConfig

	// Banlı IP adresleri
	bannedIPs sync.Map // ip -> int64 (expire timestamp)

	ipHistory sync.Map // ip -> *iotIPHistory

	// Metrikler
	totalChecked  atomic.Int64
	totalDetected atomic.Int64
	totalBanned   atomic.Int64
}

type iotIPHistory struct {
	mu          sync.Mutex
	scores      []int
	lastChecked int64
}

func NewIoTDetector(cfg IoTConfig) *IoTDetector {
	if cfg.ScoreThreshold <= 0 {
		cfg.ScoreThreshold = 50
	}
	if cfg.BanDuration <= 0 {
		cfg.BanDuration = 3600
	}
	if cfg.IPSetName == "" {
		cfg.IPSetName = "blacklist"
	}

	iot := &IoTDetector{
		config: cfg,
	}

	go iot.cleanupLoop()

	log.Println("[IoTDetect] Started", "| Threshold:", cfg.ScoreThreshold, "| BanDuration:", cfg.BanDuration)

	return iot
}

func (iot *IoTDetector) Check(r *http.Request) (score int, reasons []string) {
	iot.totalChecked.Add(1)

	path := r.URL.Path
	pathLower := strings.ToLower(path)
	query := r.URL.RawQuery
	ua := r.UserAgent()
	uaLower := strings.ToLower(ua)
	method := r.Method

	for _, ep := range exploitPaths {
		if strings.Contains(pathLower, ep) {
			score += 30
			reasons = append(reasons, "exploit_path: "+ep)
			break
		}
	}

	fullPath := path
	if query != "" {
		fullPath += "?" + query
	}
	fullPathLower := strings.ToLower(fullPath)

	for _, sig := range malwareSignatures {
		if strings.Contains(fullPathLower, sig.pattern) {
			score += sig.score
			reasons = append(reasons, "payload: "+sig.name)
			break
		}
	}

	if ua == "" {
		score += 15
		reasons = append(reasons, "empty_user_agent")
	}

	for _, botUA := range iotBotUserAgents {
		if strings.Contains(uaLower, botUA.pattern) {
			score += botUA.score
			reasons = append(reasons, "iot_ua: "+botUA.name)
			break
		}
	}

	suspiciousMethods := map[string]bool{
		"CONNECT":  true,
		"TRACE":    true,
		"TRACK":    true,
		"DEBUG":    true,
		"PROPFIND": true,
	}
	if suspiciousMethods[method] {
		score += 15
		reasons = append(reasons, "suspicious_method: "+method)
	}

	if r.Header.Get("Accept") == "" && method == "GET" {
		score += 10
		reasons = append(reasons, "missing_accept_header")
	}

	host := r.Host
	if host == "" || isIPAddress(host) {
		score += 10
		reasons = append(reasons, "ip_as_host")
	}

	for _, cmd := range commandInjectionPatterns {
		if strings.Contains(fullPathLower, cmd) {
			score += 40
			reasons = append(reasons, "cmd_injection: "+cmd)
			break
		}
	}

	if score > 0 {
		iot.totalDetected.Add(1)
	}

	return score, reasons
}

func (iot *IoTDetector) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractClientIP(r)

		if expireVal, ok := iot.bannedIPs.Load(ip); ok {
			if time.Now().Unix() < expireVal.(int64) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			iot.bannedIPs.Delete(ip)
		}

		score, reasons := iot.Check(r)

		historicalScore := iot.addToHistory(ip, score)

		if score >= iot.config.ScoreThreshold || historicalScore >= iot.config.ScoreThreshold*2 {
			iot.totalBanned.Add(1)
			expire := time.Now().Unix() + int64(iot.config.BanDuration)
			iot.bannedIPs.Store(ip, expire)

			log.Printf("[IoTDetect] BAN: %s (score: %d, cumulative: %d, reasons: %v)", ip, score, historicalScore, reasons)

			go banIPWithIPSet(iot.config.IPSetName, ip, iot.config.BanDuration)

			if iot.config.OnDetection != nil {
				go iot.config.OnDetection(ip, score, reasons)
			}

			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (iot *IoTDetector) addToHistory(ip string, score int) int {
	if score == 0 {
		return 0
	}

	val, _ := iot.ipHistory.LoadOrStore(ip, &iotIPHistory{})
	history := val.(*iotIPHistory)

	history.mu.Lock()
	defer history.mu.Unlock()

	history.scores = append(history.scores, score)
	history.lastChecked = time.Now().Unix()

	if len(history.scores) > 10 {
		history.scores = history.scores[len(history.scores)-10:]
	}

	total := 0
	for _, s := range history.scores {
		total += s
	}
	return total
}

type signatureEntry struct {
	name    string
	pattern string
	score   int
}

var exploitPaths = []string{
	// Exploitler
	"/cgi-bin/",
	"/setup.cgi",
	"/hnap1",
	"/goform/",
	"/formlogin",
	"/uir/",
	"/stssys.htm",
	"/currentsetting.htm",

	// Kamera exploitleri
	"/onvif/",
	"/dvr/",
	"/system.ini",
	"/live/",

	// Shell veya RCE exploitleri
	"/shell",
	"/cmd",
	"/command",
	"/exec",
	"/console",
	"/debug/pprof",
	"/actuator",
	"/manager/html",
	"/solr/admin",
	"/jenkins",
	"/wp-login.php",
	"/xmlrpc.php",

	// Bilinen IoT firmware pathleri
	"/firmwareupgrade",
	"/cgi-bin/viewlog.cgi",
	"/cgi-bin/wlogin.cgi",
	"/tmunblock.cgi",
	"/diagnostic.php",

	// Router backdoorları
	"/HNAP1",
	"/rom-0",
	"/login.cgi",
	"/webcm",

	// PHPUnit veya Debug
	"/vendor/phpunit",
	"/.env",
	"/.git/config",
	"/.well-known/",
	"/config.json",
	"/wp-config.php",
}

var malwareSignatures = []signatureEntry{
	// Mirai botnet
	{name: "mirai_scan", pattern: "/bin/busybox", score: 40},
	{name: "mirai_shell", pattern: "/bin/sh", score: 40},
	{name: "mirai_wget", pattern: "wget http", score: 35},
	{name: "mirai_curl", pattern: "curl http", score: 35},
	{name: "mirai_tftp", pattern: "tftp ", score: 40},
	{name: "mirai_echo", pattern: "echo -ne", score: 35},

	// Mozi botnet
	{name: "mozi_dht", pattern: "/mozi.a", score: 40},
	{name: "mozi_dht2", pattern: "/mozi.m", score: 40},

	// Hajime botnet
	{name: "hajime_atk", pattern: ".hajime", score: 40},

	// Bashlite / Gafgyt
	{name: "bashlite_arm", pattern: "/arm7", score: 35},
	{name: "bashlite_mips", pattern: "/mips", score: 30},
	{name: "bashlite_x86", pattern: "/x86_64", score: 30},

	// Genel RCE
	{name: "rce_eval", pattern: "eval(", score: 35},
	{name: "rce_system", pattern: "system(", score: 35},
	{name: "rce_passthru", pattern: "passthru(", score: 35},
	{name: "rce_exec", pattern: "exec(", score: 35},
	{name: "rce_popen", pattern: "popen(", score: 35},

	// Directory traversal
	{name: "traversal", pattern: "../../../", score: 30},
	{name: "traversal_etc", pattern: "/etc/passwd", score: 40},
	{name: "traversal_shadow", pattern: "/etc/shadow", score: 40},

	// Log4j
	{name: "log4j", pattern: "${jndi:", score: 40},
	{name: "log4j_ldap", pattern: "jndi:ldap", score: 40},
}

// IoT botnet User-Agent'ları
var iotBotUserAgents = []signatureEntry{
	{name: "hello_world", pattern: "hello, world", score: 20},
	{name: "hello_world2", pattern: "hello world", score: 20},
	{name: "compatible_bot", pattern: "mozilla/5.0 (compatible;)", score: 15},
	{name: "masscan", pattern: "masscan", score: 20},
	{name: "zgrab", pattern: "zgrab", score: 20},
	{name: "nmap", pattern: "nmap", score: 20},
	{name: "nikto", pattern: "nikto", score: 20},
	{name: "sqlmap", pattern: "sqlmap", score: 20},
	{name: "dirbuster", pattern: "dirbuster", score: 20},
	{name: "gobuster", pattern: "gobuster", score: 20},
	{name: "nuclei", pattern: "nuclei", score: 20},
	{name: "httpx", pattern: "projectdiscovery", score: 20},
	{name: "censys", pattern: "censys", score: 15},
	{name: "shodan", pattern: "shodan", score: 15},
	{name: "python_requests", pattern: "python-requests", score: 10},
	{name: "go_http", pattern: "go-http-client", score: 10},
	{name: "curl", pattern: "curl/", score: 5},
	{name: "wget", pattern: "wget/", score: 10},
	{name: "libwww", pattern: "libwww", score: 10},
	{name: "java_runtime", pattern: "java/", score: 10},
	{name: "petalbot", pattern: "petalbot", score: 10},
	{name: "ahrefsbot", pattern: "ahrefsbot", score: 5},
	{name: "semrushbot", pattern: "semrushbot", score: 5},
	{name: "bytespider", pattern: "bytespider", score: 10},
}

var commandInjectionPatterns = []string{
	";ls",
	";cat ",
	";id",
	";whoami",
	";uname",
	";ping ",
	";wget ",
	";curl ",
	";nc ",
	";bash ",
	"|ls",
	"|cat ",
	"|id",
	"|whoami",
	"`ls`",
	"`id`",
	"`whoami`",
	"$(ls)",
	"$(id)",
	"$(whoami)",
	"&&ls",
	"&&cat",
	"||ls",
	"||cat",
}

func isIPAddress(host string) bool {
	if i := strings.LastIndex(host, ":"); i != -1 {
		host = host[:i]
	}

	for _, c := range host {
		if c != '.' && (c < '0' || c > '9') {
			return false
		}
	}

	parts := strings.Split(host, ".")
	return len(parts) == 4
}

func (iot *IoTDetector) cleanupLoop() {
	ticker := time.NewTicker(120 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Unix()

		iot.bannedIPs.Range(func(key, value any) bool {
			if now > value.(int64) {
				iot.bannedIPs.Delete(key)
			}
			return true
		})

		iot.ipHistory.Range(func(key, value any) bool {
			history := value.(*iotIPHistory)
			history.mu.Lock()
			if now-history.lastChecked > 300 {
				history.mu.Unlock()
				iot.ipHistory.Delete(key)
				return true
			}
			history.mu.Unlock()
			return true
		})
	}
}

func (iot *IoTDetector) Stats() (checked, detected, banned int64) {
	return iot.totalChecked.Load(), iot.totalDetected.Load(), iot.totalBanned.Load()
}
