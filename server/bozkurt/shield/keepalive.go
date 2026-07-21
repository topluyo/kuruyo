package shield

import (
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type KeepaliveConfig struct {
	MaxRequestsPerConn int           // Bağlantı başına max istek
	DefaultIdleTimeout time.Duration // Normal idle timeout
	SuspectIdleTimeout time.Duration // Şüpheli IP idle timeout
	RiskyIdleTimeout   time.Duration // Riskli IP idle timeout
	MaxIdleConnsPerIP  int           // IP başına max idle bağlantı
	MaxTotalConnsPerIP int           // IP başına max toplam bağlantı
	AbuseTTL           time.Duration // Abuse kaydı süresi
	AbuseThreshold     int           // Abuse eşiği
	CleanupInterval    time.Duration // Temizleme aralığı
}

type RiskLevel int

const (
	RiskClean     RiskLevel = iota // Temiz
	RiskSuspect                    // Şüpheli
	RiskDangerous                  // Tehlikeli
)

type connInfo struct {
	ip           string
	state        http.ConnState
	createdAt    time.Time
	lastActivity time.Time
	requestCount atomic.Int32
	closed       atomic.Bool
}

type ipKeepaliveState struct {
	mu          sync.Mutex
	totalConns  int32
	idleConns   int32
	activeConns int32
	riskLevel   RiskLevel
	abuseCount  int32
	lastAbuse   time.Time
}

type KeepaliveManager struct {
	config KeepaliveConfig

	// Bağlantı izleme
	conns sync.Map // remoteAddr -> *connInfo

	// IP durumları
	ipStates sync.Map // ip -> *ipKeepaliveState

	// Risk score sağlayıcısı
	riskScorer func(ip string) RiskLevel

	// Metrikler
	totalConns     atomic.Int64
	totalClosed    atomic.Int64
	totalAbuse     atomic.Int64
	totalMaxReqHit atomic.Int64
}

func NewKeepaliveManager(cfg KeepaliveConfig) *KeepaliveManager {
	if cfg.MaxRequestsPerConn <= 0 {
		cfg.MaxRequestsPerConn = 1000
	}
	if cfg.DefaultIdleTimeout <= 0 {
		cfg.DefaultIdleTimeout = 120 * time.Second
	}
	if cfg.SuspectIdleTimeout <= 0 {
		cfg.SuspectIdleTimeout = 15 * time.Second
	}
	if cfg.RiskyIdleTimeout <= 0 {
		cfg.RiskyIdleTimeout = 5 * time.Second
	}
	if cfg.MaxIdleConnsPerIP <= 0 {
		cfg.MaxIdleConnsPerIP = 10
	}
	if cfg.MaxTotalConnsPerIP <= 0 {
		cfg.MaxTotalConnsPerIP = 50
	}
	if cfg.AbuseTTL <= 0 {
		cfg.AbuseTTL = 5 * time.Minute
	}
	if cfg.AbuseThreshold <= 0 {
		cfg.AbuseThreshold = 3
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 30 * time.Second
	}

	km := &KeepaliveManager{
		config: cfg,
	}

	go km.cleanupLoop()

	log.Println("[Keepalive] Started",
		"| MaxReq/Conn:", cfg.MaxRequestsPerConn,
		"| DefaultIdle:", cfg.DefaultIdleTimeout,
		"| MaxIdle/IP:", cfg.MaxIdleConnsPerIP)

	return km
}

func (km *KeepaliveManager) SetRiskScorer(scorer func(ip string) RiskLevel) {
	km.riskScorer = scorer
}

func (km *KeepaliveManager) getIPState(ip string) *ipKeepaliveState {
	val, _ := km.ipStates.LoadOrStore(ip, &ipKeepaliveState{})
	return val.(*ipKeepaliveState)
}

func (km *KeepaliveManager) ConnStateHook(conn net.Conn, state http.ConnState) {
	addr := conn.RemoteAddr().String()
	ip := extractIPFromAddr(addr)
	ipState := km.getIPState(ip)

	switch state {
	case http.StateNew:
		km.totalConns.Add(1)

		info := &connInfo{
			ip:           ip,
			state:        state,
			createdAt:    time.Now(),
			lastActivity: time.Now(),
		}
		km.conns.Store(addr, info)

		ipState.mu.Lock()
		ipState.totalConns++
		ipState.activeConns++

		if int(ipState.totalConns) > km.config.MaxTotalConnsPerIP {
			ipState.mu.Unlock()
			km.recordAbuse(ip, "max_total_conns")
			conn.Close()
			return
		}
		ipState.mu.Unlock()

	case http.StateActive:
		if val, ok := km.conns.Load(addr); ok {
			info := val.(*connInfo)
			info.state = state
			info.lastActivity = time.Now()

			ipState.mu.Lock()
			ipState.activeConns++
			if ipState.idleConns > 0 {
				ipState.idleConns--
			}
			ipState.mu.Unlock()
		}

	case http.StateIdle:
		if val, ok := km.conns.Load(addr); ok {
			info := val.(*connInfo)
			info.state = state
			info.lastActivity = time.Now()

			ipState.mu.Lock()
			ipState.idleConns++
			if ipState.activeConns > 0 {
				ipState.activeConns--
			}

			if int(ipState.idleConns) > km.config.MaxIdleConnsPerIP {
				ipState.mu.Unlock()
				km.recordAbuse(ip, "max_idle_conns")
				conn.Close()
				return
			}
			ipState.mu.Unlock()

			go km.enforceIdleTimeout(conn, addr, ip)
		}

	case http.StateClosed, http.StateHijacked:
		km.totalClosed.Add(1)

		if val, ok := km.conns.Load(addr); ok {
			info := val.(*connInfo)
			info.closed.Store(true)

			ipState.mu.Lock()
			ipState.totalConns--
			if ipState.totalConns < 0 {
				ipState.totalConns = 0
			}
			switch info.state {
			case http.StateActive:
				ipState.activeConns--
				if ipState.activeConns < 0 {
					ipState.activeConns = 0
				}
			case http.StateIdle:
				ipState.idleConns--
				if ipState.idleConns < 0 {
					ipState.idleConns = 0
				}
			}
			ipState.mu.Unlock()
		}

		km.conns.Delete(addr)
	}
}

func (km *KeepaliveManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addr := r.RemoteAddr
		ip := extractClientIP(r)

		if val, ok := km.conns.Load(addr); ok {
			info := val.(*connInfo)
			count := info.requestCount.Add(1)
			info.lastActivity = time.Now()

			if int(count) > km.config.MaxRequestsPerConn {
				km.totalMaxReqHit.Add(1)
				log.Printf("[Keepalive] MaxReq/Conn exceeded: %s (%d)", ip, count)

				w.Header().Set("Connection", "close")
				next.ServeHTTP(w, r)
				return
			}
		}

		risk := km.getIPRisk(ip)
		switch risk {
		case RiskDangerous:
			w.Header().Set("Connection", "close")
		case RiskSuspect:
			w.Header().Set("Keep-Alive", "timeout=15, max=50")
		default:
			w.Header().Set("Keep-Alive", "timeout=120, max=1000")
		}

		next.ServeHTTP(w, r)
	})
}

func (km *KeepaliveManager) enforceIdleTimeout(conn net.Conn, addr, ip string) {
	risk := km.getIPRisk(ip)

	var timeout time.Duration
	switch risk {
	case RiskDangerous:
		timeout = km.config.RiskyIdleTimeout
	case RiskSuspect:
		timeout = km.config.SuspectIdleTimeout
	default:
		timeout = km.config.DefaultIdleTimeout
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	<-timer.C

	val, ok := km.conns.Load(addr)
	if !ok {
		return
	}

	info := val.(*connInfo)
	if info.closed.Load() {
		return
	}

	if info.state == http.StateIdle {
		elapsed := time.Since(info.lastActivity)
		if elapsed >= timeout {
			conn.Close()
		}
	}
}

func (km *KeepaliveManager) getIPRisk(ip string) RiskLevel {
	if km.riskScorer != nil {
		return km.riskScorer(ip)
	}

	ipState := km.getIPState(ip)
	ipState.mu.Lock()
	defer ipState.mu.Unlock()

	if ipState.abuseCount >= int32(km.config.AbuseThreshold) {
		return RiskDangerous
	}
	if ipState.abuseCount > 0 {
		return RiskSuspect
	}
	return RiskClean
}

func (km *KeepaliveManager) recordAbuse(ip, reason string) {
	km.totalAbuse.Add(1)

	ipState := km.getIPState(ip)
	ipState.mu.Lock()
	ipState.abuseCount++
	ipState.lastAbuse = time.Now()
	count := ipState.abuseCount
	ipState.mu.Unlock()

	log.Printf("[Keepalive] Abuse #%d: %s (%s)", count, ip, reason)
}

func (km *KeepaliveManager) cleanupLoop() {
	ticker := time.NewTicker(km.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		km.conns.Range(func(key, value any) bool {
			info := value.(*connInfo)
			if info.closed.Load() {
				km.conns.Delete(key)
			}
			return true
		})

		km.ipStates.Range(func(key, value any) bool {
			state := value.(*ipKeepaliveState)
			state.mu.Lock()
			idle := state.totalConns <= 0 &&
				(state.abuseCount == 0 || now.Sub(state.lastAbuse) > km.config.AbuseTTL)
			if idle {
				state.mu.Unlock()
				km.ipStates.Delete(key)
			} else {
				if state.abuseCount > 0 && now.Sub(state.lastAbuse) > km.config.AbuseTTL {
					state.abuseCount = 0
				}
				state.mu.Unlock()
			}
			return true
		})
	}
}

func (km *KeepaliveManager) Stats() (totalConns, closed, abuse, maxReqHit int64) {
	return km.totalConns.Load(), km.totalClosed.Load(),
		km.totalAbuse.Load(), km.totalMaxReqHit.Load()
}

func (km *KeepaliveManager) OptimizedServerConfig(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: km.Middleware(handler),

		ReadHeaderTimeout: 5 * time.Second,

		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  km.config.DefaultIdleTimeout,

		MaxHeaderBytes: 1 << 20,
		ConnState:      km.ConnStateHook,
	}
}
