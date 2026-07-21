package shield

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type SlowlorisConfig struct {
	HeaderTimeout     time.Duration
	BodyTimeout       time.Duration
	MinBytesPerSec    int64
	MaxConnsPerIP     int
	MaxSlowViolations int
	BanDuration       int
	IPSetName         string
	OnBan             func(ip string, reason string)
}

type connTracker struct {
	ip        string
	startTime time.Time
	bytesRead atomic.Int64
	closed    atomic.Bool
}

type ipConnState struct {
	mu          sync.Mutex
	activeConns int32
	violations  int32
	bannedUntil int64 // unix timestamp
}

type SlowlorisGuard struct {
	config SlowlorisConfig

	ipStates sync.Map // string -> *ipConnState

	conns sync.Map // string -> *connTracker

	totalConns      atomic.Int64
	totalRejected   atomic.Int64
	totalSlowKilled atomic.Int64
	totalBanned     atomic.Int64
}

func NewSlowlorisGuard(cfg SlowlorisConfig) *SlowlorisGuard {
	if cfg.HeaderTimeout <= 0 {
		cfg.HeaderTimeout = 5 * time.Second
	}
	if cfg.BodyTimeout <= 0 {
		cfg.BodyTimeout = 30 * time.Second
	}
	if cfg.MinBytesPerSec <= 0 {
		cfg.MinBytesPerSec = 100
	}
	if cfg.MaxConnsPerIP <= 0 {
		cfg.MaxConnsPerIP = 20
	}
	if cfg.MaxSlowViolations <= 0 {
		cfg.MaxSlowViolations = 3
	}
	if cfg.BanDuration <= 0 {
		cfg.BanDuration = 600
	}
	if cfg.IPSetName == "" {
		cfg.IPSetName = "blacklist"
	}

	sg := &SlowlorisGuard{
		config: cfg,
	}

	go sg.cleanupLoop()

	log.Printf("[Slowloris] Started | HeaderTimeout: %v | MinBPS: %d | MaxConns/IP: %d", cfg.HeaderTimeout, cfg.MinBytesPerSec, cfg.MaxConnsPerIP)

	return sg
}

func (sg *SlowlorisGuard) getIPState(ip string) *ipConnState {
	val, _ := sg.ipStates.LoadOrStore(ip, &ipConnState{})
	return val.(*ipConnState)
}

func (sg *SlowlorisGuard) ConnStateHook(conn net.Conn, state http.ConnState) {
	addr := conn.RemoteAddr().String()
	ip := extractIPFromAddr(addr)

	switch state {
	case http.StateNew:
		sg.onNewConn(ip, addr, conn)

	case http.StateActive:
		// bağlantı aktif header parser tamamlanmış

	case http.StateIdle:
		// keepalive zaman aşımını sıfırla

	case http.StateClosed, http.StateHijacked:
		sg.onClosedConn(ip, addr)
	}
}

func (sg *SlowlorisGuard) onNewConn(ip, addr string, conn net.Conn) {
	sg.totalConns.Add(1)
	ipState := sg.getIPState(ip)

	ipState.mu.Lock()
	if ipState.bannedUntil > 0 && time.Now().Unix() < ipState.bannedUntil {
		ipState.mu.Unlock()
		sg.totalRejected.Add(1)
		conn.Close()
		return
	}

	ipState.activeConns++
	conns := ipState.activeConns
	ipState.mu.Unlock()

	if int(conns) > sg.config.MaxConnsPerIP {
		sg.totalRejected.Add(1)
		log.Printf("[Slowloris] MaxConn exceeded: %s (%d connections)", ip, conns)
		conn.Close()

		sg.addViolation(ip, "max_conn_exceeded")
		return
	}

	tracker := &connTracker{
		ip:        ip,
		startTime: time.Now(),
	}
	sg.conns.Store(addr, tracker)
}

func (sg *SlowlorisGuard) onClosedConn(ip, addr string) {
	sg.conns.Delete(addr)

	ipState := sg.getIPState(ip)
	ipState.mu.Lock()
	ipState.activeConns--
	if ipState.activeConns < 0 {
		ipState.activeConns = 0
	}
	ipState.mu.Unlock()
}

func (sg *SlowlorisGuard) WrapHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractClientIP(r)
		ipState := sg.getIPState(ip)

		ipState.mu.Lock()
		banned := ipState.bannedUntil > 0 && time.Now().Unix() < ipState.bannedUntil
		ipState.mu.Unlock()

		if banned {
			http.Error(w, "Blocked", http.StatusForbidden)
			return
		}

		if r.Body != nil && r.ContentLength != 0 {
			r.Body = &throttledReader{
				reader:    r.Body,
				ip:        ip,
				guard:     sg,
				startTime: time.Now(),
				deadline:  time.Now().Add(sg.config.BodyTimeout),
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (sg *SlowlorisGuard) WrapListener(ln net.Listener) net.Listener {
	return &slowlorisListener{
		Listener: ln,
		guard:    sg,
	}
}

type slowlorisListener struct {
	net.Listener
	guard *SlowlorisGuard
}

func (sl *slowlorisListener) Accept() (net.Conn, error) {
	conn, err := sl.Listener.Accept()
	if err != nil {
		return nil, err
	}

	ip := extractIPFromAddr(conn.RemoteAddr().String())
	ipState := sl.guard.getIPState(ip)

	ipState.mu.Lock()
	banned := ipState.bannedUntil > 0 && time.Now().Unix() < ipState.bannedUntil
	conns := ipState.activeConns
	ipState.mu.Unlock()

	if banned {
		conn.Close()
		sl.guard.totalRejected.Add(1)
		return sl.Accept()
	}

	if int(conns) >= sl.guard.config.MaxConnsPerIP {
		conn.Close()
		sl.guard.totalRejected.Add(1)
		sl.guard.addViolation(ip, "listener_max_conn")
		return sl.Accept()
	}

	return &slowlorisConn{
		Conn:      conn,
		guard:     sl.guard,
		ip:        ip,
		startTime: time.Now(),
	}, nil
}

type slowlorisConn struct {
	net.Conn
	guard      *SlowlorisGuard
	ip         string
	startTime  time.Time
	totalRead  atomic.Int64
	headerDone atomic.Bool
}

func (c *slowlorisConn) Read(b []byte) (int, error) {
	if !c.headerDone.Load() {
		elapsed := time.Since(c.startTime)
		if elapsed > c.guard.config.HeaderTimeout {
			c.guard.totalSlowKilled.Add(1)
			c.guard.addViolation(c.ip, "header_timeout")
			log.Printf("[Slowloris] Header timeout: %s (%v)", c.ip, elapsed)
			return 0, io.EOF
		}
	}

	n, err := c.Conn.Read(b)
	if n > 0 {
		c.totalRead.Add(int64(n))

		elapsed := time.Since(c.startTime).Seconds()
		if elapsed > 2 {
			bps := float64(c.totalRead.Load()) / elapsed
			if bps < float64(c.guard.config.MinBytesPerSec) {
				c.guard.totalSlowKilled.Add(1)
				c.guard.addViolation(c.ip, "slow_throughput")
				log.Printf("[Slowloris] Slow throughput: %s (%.0f B/s < %d B/s)",
					c.ip, bps, c.guard.config.MinBytesPerSec)
				return 0, io.EOF
			}
		}

		if !c.headerDone.Load() && c.totalRead.Load() > 4 {
			c.headerDone.Store(true)
		}
	}

	return n, err
}

func (c *slowlorisConn) Close() error {
	c.guard.onClosedConn(c.ip, c.Conn.RemoteAddr().String())
	return c.Conn.Close()
}

type throttledReader struct {
	reader    io.ReadCloser
	ip        string
	guard     *SlowlorisGuard
	startTime time.Time
	deadline  time.Time
	totalRead int64
}

func (tr *throttledReader) Read(p []byte) (int, error) {
	if time.Now().After(tr.deadline) {
		tr.guard.totalSlowKilled.Add(1)
		tr.guard.addViolation(tr.ip, "body_timeout")
		log.Printf("[Slowloris] Body timeout: %s", tr.ip)
		return 0, io.EOF
	}

	n, err := tr.reader.Read(p)
	tr.totalRead += int64(n)

	elapsed := time.Since(tr.startTime).Seconds()
	if elapsed > 2 && tr.totalRead > 0 {
		bps := float64(tr.totalRead) / elapsed
		if bps < float64(tr.guard.config.MinBytesPerSec) {
			tr.guard.totalSlowKilled.Add(1)
			tr.guard.addViolation(tr.ip, "slow_body")
			log.Printf("[Slowloris] Slow body: %s (%.0f B/s)", tr.ip, bps)
			return 0, io.EOF
		}
	}

	return n, err
}

func (tr *throttledReader) Close() error {
	return tr.reader.Close()
}

func (sg *SlowlorisGuard) addViolation(ip, reason string) {
	ipState := sg.getIPState(ip)

	ipState.mu.Lock()
	ipState.violations++
	violations := ipState.violations

	if int(violations) >= sg.config.MaxSlowViolations {
		ipState.bannedUntil = time.Now().Unix() + int64(sg.config.BanDuration)
		ipState.violations = 0
		ipState.mu.Unlock()

		sg.totalBanned.Add(1)
		log.Printf("[Slowloris] BAN: %s ( violations: %d, last: %s)", ip, violations, reason)

		// ipset kernel yada cekirdek ne dersen de artık o seviye'de ban
		go banIPWithIPSet(sg.config.IPSetName, ip, sg.config.BanDuration)

		if sg.config.OnBan != nil {
			go sg.config.OnBan(ip, reason)
		}
		return
	}

	ipState.mu.Unlock()
	log.Printf("[Slowloris] Violation #%d: %s (%s)", violations, ip, reason)
}

func (sg *SlowlorisGuard) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Unix()
		sg.ipStates.Range(func(key, value any) bool {
			state := value.(*ipConnState)
			state.mu.Lock()
			idle := state.activeConns <= 0 &&
				(state.bannedUntil == 0 || now > state.bannedUntil)
			state.mu.Unlock()

			if idle {
				sg.ipStates.Delete(key)
			}
			return true
		})
	}
}

func (sg *SlowlorisGuard) Stats() (totalConns, rejected, slowKilled, banned int64) {
	return sg.totalConns.Load(), sg.totalRejected.Load(),
		sg.totalSlowKilled.Load(), sg.totalBanned.Load()
}

func extractIPFromAddr(addr string) string {
	if i := strings.LastIndex(addr, ":"); i != -1 {
		return addr[:i]
	}
	return addr
}
