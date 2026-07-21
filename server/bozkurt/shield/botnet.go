package shield

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type BotnetConfig struct {
	ClusterThreshold    int           // Cluster boyutu eşiği
	TimeWindow          time.Duration // Analiz penceresi
	CorrelationMs       int64         // Timing korelasyon toleransı ms
	SubnetThreshold     int           // /24 subnet IP eşiği
	FingerprintLifetime time.Duration // Fingerprint kara liste süresi
	BanDuration         int           // Ban süresi saniye
	AnalysisInterval    time.Duration // Analiz çalışma aralığı
	IPSetName           string        // ipset adı
	OnBotnetDetected    func(clusterID string, ips []string, reason string)
}

type requestEvent struct {
	ip        string
	path      string
	method    string
	userAgent string
	ja3Hash   string
	timestamp time.Time
}

// botnet clusterı
type cluster struct {
	fingerprint string
	ips         map[string]bool
	firstSeen   time.Time
	lastSeen    time.Time
	reason      string
}

type BotnetDetector struct {
	config BotnetConfig

	eventsMu sync.Mutex
	events   []requestEvent

	// Fingerprint tabanlı clusterlar
	clustersMu sync.RWMutex
	clusters   map[string]*cluster // fingerprint -> cluster

	// Blacklistteki fingerprint'ler
	blacklistMu sync.RWMutex
	blacklist   map[string]int64 // fingerprint -> expire timestamp

	// Banlı IP adresleri
	bannedIPs sync.Map // ip -> int64 (expire timestamp)

	// Subnet izleme
	subnetMu sync.Mutex
	subnets  map[string]map[string]int64 // /24 -> ip -> lastSeen

	stopCh chan struct{}

	// Metrikler
	totalAnalyzed   atomic.Int64
	totalClusters   atomic.Int64
	totalBotnetBans atomic.Int64
}

func NewBotnetDetector(cfg BotnetConfig) *BotnetDetector {
	if cfg.ClusterThreshold <= 0 {
		cfg.ClusterThreshold = 10
	}
	if cfg.TimeWindow <= 0 {
		cfg.TimeWindow = 60 * time.Second
	}
	if cfg.CorrelationMs <= 0 {
		cfg.CorrelationMs = 500
	}
	if cfg.SubnetThreshold <= 0 {
		cfg.SubnetThreshold = 8
	}
	if cfg.FingerprintLifetime <= 0 {
		cfg.FingerprintLifetime = 30 * time.Minute
	}
	if cfg.BanDuration <= 0 {
		cfg.BanDuration = 1800
	}
	if cfg.AnalysisInterval <= 0 {
		cfg.AnalysisInterval = 5 * time.Second
	}
	if cfg.IPSetName == "" {
		cfg.IPSetName = "blacklist"
	}

	bd := &BotnetDetector{
		config:    cfg,
		clusters:  make(map[string]*cluster),
		blacklist: make(map[string]int64),
		subnets:   make(map[string]map[string]int64),
		stopCh:    make(chan struct{}),
	}

	go bd.analysisLoop()

	go bd.cleanupLoop()

	log.Println("[Botnet] Started", "| ClusterThreshold:", cfg.ClusterThreshold, "| Window:", cfg.TimeWindow, "| CorrelationMs:", cfg.CorrelationMs)

	return bd
}

func (bd *BotnetDetector) Analyze(ip, path, method, userAgent, ja3Hash string) bool {
	bd.totalAnalyzed.Add(1)
	now := time.Now()

	if expireVal, ok := bd.bannedIPs.Load(ip); ok {
		expire := expireVal.(int64)
		if now.Unix() < expire {
			return true
		}
		bd.bannedIPs.Delete(ip)
	}

	fp := bd.generateFingerprint(path, method, userAgent, ja3Hash)

	bd.blacklistMu.RLock()
	if expire, ok := bd.blacklist[fp]; ok && now.Unix() < expire {
		bd.blacklistMu.RUnlock()
		bd.banIP(ip, "blacklisted_fingerprint: "+fp[:16])
		return true
	}
	bd.blacklistMu.RUnlock()

	event := requestEvent{
		ip:        ip,
		path:      path,
		method:    method,
		userAgent: userAgent,
		ja3Hash:   ja3Hash,
		timestamp: now,
	}

	bd.eventsMu.Lock()
	bd.events = append(bd.events, event)
	bd.eventsMu.Unlock()

	bd.trackSubnet(ip, now)

	return false
}

func (bd *BotnetDetector) generateFingerprint(path, method, userAgent, ja3Hash string) string {
	normalizedPath := normalizePath(path)

	raw := method + "|" + normalizedPath + "|" + userAgent
	if ja3Hash != "" {
		raw += "|" + ja3Hash
	}

	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:12])
}

func normalizePath(path string) string {
	if i := strings.Index(path, "?"); i != -1 {
		path = path[:i]
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 3 {
		parts = parts[:3]
	}

	return "/" + strings.Join(parts, "/")
}

func (bd *BotnetDetector) trackSubnet(ip string, now time.Time) {
	subnet := extractSubnet24(ip)
	if subnet == "" {
		return
	}

	bd.subnetMu.Lock()
	if bd.subnets[subnet] == nil {
		bd.subnets[subnet] = make(map[string]int64)
	}
	bd.subnets[subnet][ip] = now.Unix()
	count := len(bd.subnets[subnet])
	bd.subnetMu.Unlock()

	if count >= bd.config.SubnetThreshold {
		bd.handleSubnetCluster(subnet)
	}
}

func extractSubnet24(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ""
	}
	return parts[0] + "." + parts[1] + "." + parts[2] + ".0/24"
}

func (bd *BotnetDetector) handleSubnetCluster(subnet string) {
	bd.subnetMu.Lock()
	ips := make([]string, 0, len(bd.subnets[subnet]))
	for ip := range bd.subnets[subnet] {
		ips = append(ips, ip)
	}
	bd.subnetMu.Unlock()

	if len(ips) < bd.config.SubnetThreshold {
		return
	}

	bd.totalClusters.Add(1)
	log.Printf("[Botnet] Subnet cluster detected: %s (%d IP)", subnet, len(ips))

	for _, ip := range ips {
		bd.banIP(ip, "subnet_cluster: "+subnet)
	}

	if bd.config.OnBotnetDetected != nil {
		go bd.config.OnBotnetDetected(subnet, ips, "subnet_cluster")
	}
}

func (bd *BotnetDetector) analysisLoop() {
	ticker := time.NewTicker(bd.config.AnalysisInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bd.stopCh:
			return
		case <-ticker.C:
			bd.runAnalysis()
		}
	}
}

func (bd *BotnetDetector) runAnalysis() {
	now := time.Now()
	cutoff := now.Add(-bd.config.TimeWindow)

	bd.eventsMu.Lock()
	var recent []requestEvent
	for _, e := range bd.events {
		if e.timestamp.After(cutoff) {
			recent = append(recent, e)
		}
	}
	bd.events = recent
	bd.eventsMu.Unlock()

	if len(recent) < bd.config.ClusterThreshold {
		return
	}

	fpClusters := make(map[string]map[string]bool)

	for _, e := range recent {
		fp := bd.generateFingerprint(e.path, e.method, e.userAgent, e.ja3Hash)
		if fpClusters[fp] == nil {
			fpClusters[fp] = make(map[string]bool)
		}
		fpClusters[fp][e.ip] = true
	}

	for fp, ipSet := range fpClusters {
		if len(ipSet) >= bd.config.ClusterThreshold {
			bd.handleFingerprintCluster(fp, ipSet)
		}
	}

	bd.analyzeTimingCorrelation(recent)
}

func (bd *BotnetDetector) handleFingerprintCluster(fp string, ipSet map[string]bool) {
	ips := make([]string, 0, len(ipSet))
	for ip := range ipSet {
		ips = append(ips, ip)
	}

	bd.totalClusters.Add(1)
	log.Printf("[Botnet] Fingerprint cluster detected: %s... (%d IP)", fp[:16], len(ips))

	bd.blacklistMu.Lock()
	bd.blacklist[fp] = time.Now().Add(bd.config.FingerprintLifetime).Unix()
	bd.blacklistMu.Unlock()

	for _, ip := range ips {
		bd.banIP(ip, "fingerprint_cluster: "+fp[:16])
	}

	if bd.config.OnBotnetDetected != nil {
		go bd.config.OnBotnetDetected(fp, ips, "fingerprint_cluster")
	}
}

func (bd *BotnetDetector) analyzeTimingCorrelation(events []requestEvent) {
	if len(events) < bd.config.ClusterThreshold {
		return
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].timestamp.Before(events[j].timestamp)
	})

	toleranceMs := bd.config.CorrelationMs

	for i := 0; i < len(events); i++ {
		correlatedIPs := make(map[string]bool)
		correlatedIPs[events[i].ip] = true

		baseTime := events[i].timestamp.UnixMilli()

		for j := i + 1; j < len(events); j++ {
			diff := events[j].timestamp.UnixMilli() - baseTime
			if diff > toleranceMs {
				break
			}
			correlatedIPs[events[j].ip] = true
		}

		if len(correlatedIPs) >= bd.config.ClusterThreshold {
			ips := make([]string, 0, len(correlatedIPs))
			for ip := range correlatedIPs {
				ips = append(ips, ip)
			}

			bd.totalClusters.Add(1)
			ts := time.UnixMilli(baseTime).Format("15:04:05.000")
			log.Printf("[Botnet] Timing cluster detected: %s ±%dms (%d IP)",
				ts, toleranceMs, len(ips))

			for _, ip := range ips {
				bd.banIP(ip, fmt.Sprintf("timing_cluster @%s", ts))
			}

			if bd.config.OnBotnetDetected != nil {
				go bd.config.OnBotnetDetected("timing_"+ts, ips, "timing_correlation")
			}

			i += len(correlatedIPs) - 1
		}
	}
}

func (bd *BotnetDetector) banIP(ip, reason string) {
	if _, ok := bd.bannedIPs.Load(ip); ok {
		return
	}

	expire := time.Now().Unix() + int64(bd.config.BanDuration)
	bd.bannedIPs.Store(ip, expire)
	bd.totalBotnetBans.Add(1)

	log.Printf("[Botnet] BAN: %s (%s)", ip, reason)

	go func() {
		banIPWithIPSet(bd.config.IPSetName, ip, bd.config.BanDuration)
	}()
}

func (bd *BotnetDetector) cleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bd.stopCh:
			return
		case <-ticker.C:
			now := time.Now().Unix()

			bd.bannedIPs.Range(func(key, value any) bool {
				if now > value.(int64) {
					bd.bannedIPs.Delete(key)
				}
				return true
			})

			bd.blacklistMu.Lock()
			for fp, expire := range bd.blacklist {
				if now > expire {
					delete(bd.blacklist, fp)
				}
			}
			bd.blacklistMu.Unlock()

			windowSec := int64(bd.config.TimeWindow.Seconds())
			bd.subnetMu.Lock()
			for subnet, ips := range bd.subnets {
				for ip, lastSeen := range ips {
					if now-lastSeen > windowSec*2 {
						delete(ips, ip)
					}
				}
				if len(ips) == 0 {
					delete(bd.subnets, subnet)
				}
			}
			bd.subnetMu.Unlock()
		}
	}
}

func (bd *BotnetDetector) Stop() {
	close(bd.stopCh)
}

func (bd *BotnetDetector) Stats() (analyzed, clusters, bans int64) {
	return bd.totalAnalyzed.Load(), bd.totalClusters.Load(), bd.totalBotnetBans.Load()
}
