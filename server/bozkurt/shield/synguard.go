package shield

import (
	"bufio"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type SYNGuardConfig struct {
	MaxSYNPerIP    int                           // IP başına max SYN_RECV (50)
	GlobalSYNLimit int                           // Toplam max SYN_RECV (5000)
	CheckInterval  time.Duration                 // Kontrol aralığı (2 yeterli)
	BanDuration    int                           // ban süresi (default 600 saniye yeterli bence)
	IPSetName      string                        // ipset adı (default blacklist olarak ayarladim)
	OnBan          func(ip string, synCount int) // Ban callback (keyfinize kalmis bu arada)
}

type SYNGuard struct {
	config  SYNGuardConfig
	banned  sync.Map // string -> int64 (ban zamanı)
	running atomic.Bool
	stopCh  chan struct{}

	// Metrikler
	totalSYN    atomic.Int64
	totalBanned atomic.Int64
	lastCheckMs atomic.Int64
}

func NewSYNGuard(cfg SYNGuardConfig) *SYNGuard {
	if cfg.MaxSYNPerIP <= 0 {
		cfg.MaxSYNPerIP = 50
	}
	if cfg.GlobalSYNLimit <= 0 {
		cfg.GlobalSYNLimit = 5000
	}
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = 2 * time.Second
	}
	if cfg.BanDuration <= 0 {
		cfg.BanDuration = 600
	}
	if cfg.IPSetName == "" {
		cfg.IPSetName = "blacklist"
	}

	return &SYNGuard{
		config: cfg,
		stopCh: make(chan struct{}),
	}
}

func (sg *SYNGuard) Start() {
	if sg.running.Load() {
		return
	}
	sg.running.Store(true)

	// Kernel parameterları optimize et
	sg.optimizeKernel()

	// goroutine
	go sg.monitor()

	log.Printf("[SYNGuard] Started | MaxSYN/IP: %d | GlobalLimit: %d | Interval: %v", sg.config.MaxSYNPerIP, sg.config.GlobalSYNLimit, sg.config.CheckInterval)
}

func (sg *SYNGuard) Stop() {
	if !sg.running.Load() {
		return
	}
	sg.running.Store(false)
	close(sg.stopCh)
}

func (sg *SYNGuard) Stats() (totalSYN, totalBanned, lastCheckMs int64) {
	return sg.totalSYN.Load(), sg.totalBanned.Load(), sg.lastCheckMs.Load()
}

func (sg *SYNGuard) optimizeKernel() {
	params := map[string]string{
		//Bu kısımı değişebilirsiniz bu arada ben böyle yapmayı uygun gördüm -Andre
		// SYN flood backlog dolmadan cookie ile doğrulama
		"net.ipv4.tcp_syncookies": "1",

		// SYN backlog boyutu yarım açık bağlantı kuyruğu
		"net.ipv4.tcp_max_syn_backlog": "65536",

		// SYN-ACK tekrar deneme düşük tut ki zombi bağlantılar gitsin
		"net.ipv4.tcp_synack_retries": "2",

		// Bağlantı kapanış süresini kısalt
		"net.ipv4.tcp_fin_timeout": "15",

		// Soketleri yeniden kullan
		"net.ipv4.tcp_tw_reuse": "1",

		// Orphan soket limiti
		"net.ipv4.tcp_max_orphans": "65536",

		// Bağlantı izleme tablosu boyutu
		"net.netfilter.nf_conntrack_max": "1000000",
	}

	for key, val := range params {
		cmd := exec.Command("sysctl", "-w", key+"="+val)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("[SYNGuard] sysctl %s=%s error: %v (%s)", key, val, err, string(out))
		}
	}

	log.Println("[SYNGuard] Kernel parameters optimized")
}

// TCP bağlantı durumları (kernel)
const (
	tcpSynRecv = 0x03
)

// parseProcTCP /proc/net/tcp dosyasını parse edip SYN_RECV IP'lerini döndürür
func (sg *SYNGuard) parseProcTCP() map[string]int {
	synCounts := make(map[string]int)

	// IPv4
	sg.parseTCPFile("/proc/net/tcp", synCounts)

	// IPv6
	sg.parseTCPFile("/proc/net/tcp6", synCounts)

	return synCounts
}

func (sg *SYNGuard) parseTCPFile(path string, counts map[string]int) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		stateHex := fields[3]
		state, err := strconv.ParseInt(stateHex, 16, 32)
		if err != nil {
			continue
		}

		if state != tcpSynRecv {
			continue
		}

		remoteAddr := fields[2]
		ip := hexToIP(remoteAddr)
		if ip != "" {
			counts[ip]++
		}
	}
}

func hexToIP(hexAddr string) string {
	parts := strings.Split(hexAddr, ":")
	if len(parts) != 2 {
		return ""
	}

	hexIP := parts[0]

	switch len(hexIP) {
	case 8: // IPv4
		ip := make(net.IP, 4)
		for i := 0; i < 4; i++ {
			b, err := strconv.ParseUint(hexIP[i*2:i*2+2], 16, 8)
			if err != nil {
				return ""
			}
			// Little-endiandan reverse
			ip[3-i] = byte(b)
		}
		return ip.String()

	case 32: // IPv6
		ip := make(net.IP, 16)
		for i := 0; i < 16; i++ {
			b, err := strconv.ParseUint(hexIP[i*2:i*2+2], 16, 8)
			if err != nil {
				return ""
			}
			ip[i] = byte(b)
		}
		return ip.String()
	}

	return ""
}

func (sg *SYNGuard) monitor() {
	ticker := time.NewTicker(sg.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sg.stopCh:
			return
		case <-ticker.C:
			sg.check()
		}
	}
}

func (sg *SYNGuard) check() {
	start := time.Now()

	synCounts := sg.parseProcTCP()

	totalSYN := 0
	for ip, count := range synCounts {
		totalSYN += count

		if count >= sg.config.MaxSYNPerIP {
			if _, ok := sg.banned.Load(ip); ok {
				continue
			}

			sg.ban(ip, count)
		}
	}

	sg.totalSYN.Store(int64(totalSYN))
	sg.lastCheckMs.Store(time.Since(start).Milliseconds())

	if totalSYN >= sg.config.GlobalSYNLimit {
		log.Printf("[SYNGuard] GLOBAL SYN ALARM! Total SYN_RECV: %d (limit: %d)",
			totalSYN, sg.config.GlobalSYNLimit)
		for ip, count := range synCounts {
			if count >= sg.config.MaxSYNPerIP/2 {
				if _, ok := sg.banned.Load(ip); !ok {
					sg.ban(ip, count)
				}
			}
		}
	}
}

func (sg *SYNGuard) ban(ip string, synCount int) {
	sg.banned.Store(ip, time.Now().Unix())
	sg.totalBanned.Add(1)

	banIPWithIPSet(sg.config.IPSetName, ip, sg.config.BanDuration)
	log.Printf("[SYNGuard] BAN: %s (SYN_RECV: %d)", ip, synCount)

	if sg.config.OnBan != nil {
		go sg.config.OnBan(ip, synCount)
	}
	go func() {
		time.Sleep(time.Duration(sg.config.BanDuration) * time.Second)
		sg.banned.Delete(ip)
	}()
}
