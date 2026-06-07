package tslguard

import (
	"io"
	"log"
	"net"
	"os/exec"
	"sync"
	"sync/atomic"
)

var (
	ipQueue = make(chan string, 10000)
	once    sync.Once
	workers = 4

	ipCount sync.Map // map[string]*int64
)

// =======================
// EXPORT: INIT & WRAP
// =======================

// Init TLS Guard (start workers, ipset)
func Init() {
	once.Do(func() {
		ensureInstalled()
		setupIPSet()
		for i := 0; i < workers; i++ {
			go worker()
		}
	})
}

// WrapListener: reuseport veya standart listener'ı tracking listener ile sarar
func WrapListener(ln net.Listener) net.Listener {
	return trackingListener{Listener: ln}
}

// ReportIP: manuel olarak IP eklemek istersen
func ReportIP(ip string) {
	select {
	case ipQueue <- ip:
	default:
		// queue doluysa drop
	}
}

// =======================
// TRACKING LISTENER & CONN
// =======================

type trackingListener struct {
	net.Listener
}

func (l trackingListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &trackingConn{Conn: c}, nil
}

type trackingConn struct {
	net.Conn
}

func (c *trackingConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if err != nil && err == io.EOF {
		ip := extractIP(c.Conn.RemoteAddr())
		if ip != "" {
			registerEOF(ip)
		}
	}
	return n, err
}

func extractIP(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	ip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return ""
	}
	return ip
}

// =======================
// FLOOD CONTROL
// =======================

func registerEOF(ip string) {
	val, _ := ipCount.LoadOrStore(ip, new(int64))
	counter := val.(*int64)

	n := atomic.AddInt64(counter, 1)

	if n < 20 { // threshold
		return
	}

	ReportIP(ip)
}

// =======================
// WORKER POOL
// =======================

func worker() {
	for ip := range ipQueue {
		banIP(ip)
	}
}

// =======================
// IPSET + IPTABLES
// =======================

func banIP(ip string) {
	cmd := exec.Command("ipset", "add", "blacklist", ip, "timeout", "600", "-exist")
	err := cmd.Run()
	if err != nil {
		log.Println("[x] ipset error:", err)
	}else{
		log.Println("[+] ipset add:", ip)
	}
}

func ensureInstalled() {
	if !commandExists("ipset") || !commandExists("iptables") {
		log.Println("Installing ipset & iptables...")

		run("apt-get", "update")
		run("apt-get", "install", "-y", "ipset", "iptables")
	}
}

func setupIPSet() {
	run("ipset", "create", "blacklist", "hash:ip", "timeout", "600", "-exist")

	// iptables rule kontrol
	err := exec.Command("iptables", "-C", "INPUT",
		"-m", "set", "--match-set", "blacklist", "src", "-j", "DROP").Run()

	if err != nil {
		run("iptables", "-A", "INPUT",
			"-m", "set", "--match-set", "blacklist", "src", "-j", "DROP")
	}
}

func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[%s] error: %v | %s\n", name, err, string(out))
	}
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
