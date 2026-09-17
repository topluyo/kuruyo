package shield

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type GuardConfig struct {
	EnableSYN    bool
	EnableHTTP   bool
	EnableSlow   bool
	EnableIoT    bool
	EnableBotnet bool
	EnableKeep   bool

	// Reverse Proxy
	Listen   string
	Upstream string

	// Ortak
	IPSetName  string
	OnCloudBan func(ip, reason string)

	SYNBanSec    int
	HTTPBanSec   int
	IoTBanSec    int
	BotnetBanSec int
	SlowBanSec   int

	SYNMaxPerIP    int
	SYNGlobalLimit int
}

type GuardManager struct {
	config  GuardConfig
	started bool

	syn  *SYNGuard
	http *HTTPFlood
	slow *SlowlorisGuard
	iot  *IoTDetector
	bot  *BotnetDetector
	keep *KeepaliveManager

	proxy *httputil.ReverseProxy
}

func NewGuardManager(cfg GuardConfig) *GuardManager {
	if cfg.IPSetName == "" {
		cfg.IPSetName = "blacklist"
	}
	g := &GuardManager{config: cfg}

	cloudBan := func(ip, reason string) {
		if cfg.OnCloudBan != nil {
			cfg.OnCloudBan(ip, reason)
		}
	}

	// SYN guard
	if cfg.EnableSYN {
		g.syn = NewSYNGuard(SYNGuardConfig{
			MaxSYNPerIP:    cfg.SYNMaxPerIP,
			GlobalSYNLimit: cfg.SYNGlobalLimit,
			CheckInterval:  2 * time.Second,
			BanDuration:    firstNonZero(cfg.SYNBanSec, 600),
			IPSetName:      cfg.IPSetName,
			OnBan:          func(ip string, _ int) { cloudBan(ip, "syn_flood") },
		})
	}

	// HTTP Flood
	if cfg.EnableHTTP {
		g.http = NewHTTPFlood(HTTPFloodConfig{
			WindowSize:      30 * time.Second,
			ChallengeScore:  70,
			BanScore:        90,
			MaxConnsPerIP:   20,
			MaxReqPerWindow: 200,
			BanDuration:     time.Duration(firstNonZero(cfg.HTTPBanSec, 600)) * time.Second,
			IPSetName:       cfg.IPSetName,
			OnBan:           func(ip string, _ int, _ string) { cloudBan(ip, "http_flood") },
		})
	}

	// Slowloris
	if cfg.EnableSlow {
		g.slow = NewSlowlorisGuard(SlowlorisConfig{
			HeaderTimeout:     5 * time.Second,
			BodyTimeout:       30 * time.Second,
			MinBytesPerSec:    100,
			MaxConnsPerIP:     20,
			MaxSlowViolations: 3,
			BanDuration:       firstNonZero(cfg.SlowBanSec, 600),
			IPSetName:         cfg.IPSetName,
			OnBan:             func(ip, _ string) { cloudBan(ip, "slowloris") },
		})
	}

	// IoT
	if cfg.EnableIoT {
		g.iot = NewIoTDetector(IoTConfig{
			ScoreThreshold: 50,
			BanDuration:    firstNonZero(cfg.IoTBanSec, 3600),
			IPSetName:      cfg.IPSetName,
			OnDetection:    func(ip string, _ int, _ []string) { cloudBan(ip, "iot_malware") },
		})
	}

	// Botnet
	if cfg.EnableBotnet {
		g.bot = NewBotnetDetector(BotnetConfig{
			ClusterThreshold:    10,
			TimeWindow:          60 * time.Second,
			CorrelationMs:       500,
			SubnetThreshold:     8,
			FingerprintLifetime: 30 * time.Minute,
			BanDuration:         firstNonZero(cfg.BotnetBanSec, 1800),
			IPSetName:           cfg.IPSetName,
			OnBotnetDetected: func(_ string, ips []string, _ string) {
				for _, ip := range ips {
					cloudBan(ip, "botnet_cluster")
				}
			},
		})
	}

	// Keepalive
	if cfg.EnableKeep {
		g.keep = NewKeepaliveManager(KeepaliveConfig{
			MaxRequestsPerConn: 1000,
			DefaultIdleTimeout: 120 * time.Second,
			SuspectIdleTimeout: 15 * time.Second,
			RiskyIdleTimeout:   5 * time.Second,
			MaxIdleConnsPerIP:  10,
			MaxTotalConnsPerIP: 50,
		})
		g.keep.SetRiskScorer(g.riskScorer)
	}

	// Reverse-proxy
	if cfg.Upstream != "" {
		u, err := url.Parse(cfg.Upstream)
		if err != nil {
			log.Printf("[Guard] upstream parse error %s: %v", cfg.Upstream, err)
		} else {
			g.proxy = httputil.NewSingleHostReverseProxy(u)
		}
	}

	return g
}

func (g *GuardManager) Start() {
	if g.started {
		return
	}
	g.started = true
	if g.syn != nil {
		g.syn.Start()
	}
	log.Printf("[Guard] Started | SYN:%v HTTP:%v Slow:%v IoT:%v Botnet:%v Keep:%v | upstream=%s",
		g.syn != nil, g.http != nil, g.slow != nil,
		g.iot != nil, g.bot != nil, g.keep != nil, g.config.Upstream)
}

func (g *GuardManager) Stop() {
	if g.syn != nil {
		g.syn.Stop()
	}
	if g.bot != nil {
		g.bot.Stop()
	}
}

func (g *GuardManager) Handler() http.Handler {
	if g.proxy == nil {
		// upstream yoksa 502 dönsün sadece koruma katmanı çalışır
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "shield active, no upstream configured", http.StatusBadGateway)
		})
	}

	// upstream reverse-proxy
	var h http.Handler = http.HandlerFunc(g.proxy.ServeHTTP)

	// Botnet: her request analiz eder banlıysa 403 degilse devamke
	if g.bot != nil {
		bot := g.bot
		inner := h
		h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractClientIP(r)
			if bot.Analyze(ip, r.URL.Path, r.Method, r.UserAgent(), ja3FromRequest(r)) {
				http.Error(w, "Access Denied", http.StatusForbidden)
				return
			}
			inner.ServeHTTP(w, r)
		})
	}

	// HTTPFlood
	if g.http != nil {
		h = g.http.Middleware(h)
	}
	// IoT
	if g.iot != nil {
		h = g.iot.Middleware(h)
	}
	// Slowloris
	if g.slow != nil {
		h = g.slow.WrapHandler(h)
	}
	// Keepalive
	if g.keep != nil {
		h = g.keep.Middleware(h)
	}
	return h
}

func (g *GuardManager) ConnStateHook(conn net.Conn, state http.ConnState) {
	if g.slow != nil {
		g.slow.ConnStateHook(conn, state)
	}
	if g.keep != nil {
		g.keep.ConnStateHook(conn, state)
	}
}

func (g *GuardManager) riskScorer(ip string) RiskLevel {
	if g.bot != nil {
		if v, ok := g.bot.bannedIPs.Load(ip); ok {
			if exp, _ := v.(int64); time.Now().Unix() < exp {
				return RiskDangerous
			}
		}
	}
	if g.http != nil {
		if p, ok := g.http.profiles.Load(ip); ok {
			prof := p.(*IPProfile)
			if bu := prof.bannedUntil.Load(); bu > 0 && time.Now().Unix() < bu {
				return RiskSuspect
			}
		}
	}
	return RiskClean
}

func ja3FromRequest(r *http.Request) string {
	return r.Header.Get("X-JA3-Hash")
}

func firstNonZero(vals ...int) int {
	for _, v := range vals {
		if v > 0 {
			return v
		}
	}
	return 0
}
