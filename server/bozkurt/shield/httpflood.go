package shield

import (
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type HTTPFloodConfig struct {
	WindowSize      time.Duration                             // İzleme penceresi
	ChallengeScore  int                                       // Challenge göster skoru
	BanScore        int                                       // Direkt ban skoru
	MaxConnsPerIP   int                                       // IP başına max eşzamanlı bağlantı
	MaxReqPerWindow int                                       // Pencere başına max istek
	CleanupInterval time.Duration                             // Temizleme aralığı
	BanDuration     time.Duration                             // Ban süresi
	IPSetName       string                                    // ipset adı
	OnBan           func(ip string, score int, reason string) // callback
}

type IPProfile struct {
	mu sync.Mutex

	// Request verileri
	requests      []requestRecord
	totalRequests atomic.Int64

	// Eşzamanlı bağlantı sayısı
	activeConns atomic.Int32

	// Tehdit skoru
	lastScore atomic.Int32

	// Ban bilgisi
	bannedUntil atomic.Int64
}

type requestRecord struct {
	timestamp  time.Time
	path       string
	method     string
	userAgent  string
	statusCode int
	bodySize   int64
}

type HTTPFlood struct {
	config   HTTPFloodConfig
	profiles sync.Map // string -> *IPProfile
	stopCh   chan struct{}

	// Metrikler
	totalChecked    atomic.Int64
	totalBlocked    atomic.Int64
	totalChallenged atomic.Int64
}

func NewHTTPFlood(cfg HTTPFloodConfig) *HTTPFlood {
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = 30 * time.Second
	}
	if cfg.ChallengeScore <= 0 {
		cfg.ChallengeScore = 70
	}
	if cfg.BanScore <= 0 {
		cfg.BanScore = 90
	}
	if cfg.MaxConnsPerIP <= 0 {
		cfg.MaxConnsPerIP = 20
	}
	if cfg.MaxReqPerWindow <= 0 {
		cfg.MaxReqPerWindow = 200
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 60 * time.Second
	}
	if cfg.BanDuration <= 0 {
		cfg.BanDuration = 10 * time.Minute
	}
	if cfg.IPSetName == "" {
		cfg.IPSetName = "blacklist"
	}

	hf := &HTTPFlood{
		config: cfg,
		stopCh: make(chan struct{}),
	}

	go hf.cleanup()

	log.Println("[HTTPFlood] Started | Window:", cfg.WindowSize, "| Challenge:", cfg.ChallengeScore, "| Ban:", cfg.BanScore)

	return hf
}

func (hf *HTTPFlood) getProfile(ip string) *IPProfile {
	val, _ := hf.profiles.LoadOrStore(ip, &IPProfile{})
	return val.(*IPProfile)
}

func (hf *HTTPFlood) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractClientIP(r)
		profile := hf.getProfile(ip)

		hf.totalChecked.Add(1)

		bannedUntil := profile.bannedUntil.Load()
		if bannedUntil > 0 && time.Now().Unix() < bannedUntil {
			hf.totalBlocked.Add(1)
			http.Error(w, "Access Denied", http.StatusForbidden)
			return
		}

		conns := profile.activeConns.Add(1)
		defer profile.activeConns.Add(-1)

		if int(conns) > hf.config.MaxConnsPerIP {
			hf.totalBlocked.Add(1)
			log.Printf("[HTTPFlood] MaxConn exceeded: %s (%d)", ip, conns)
			http.Error(w, "Too Many Connections", http.StatusServiceUnavailable)
			return
		}
		record := requestRecord{
			timestamp: time.Now(),
			path:      r.URL.Path,
			method:    r.Method,
			userAgent: r.UserAgent(),
		}

		profile.mu.Lock()
		profile.requests = append(profile.requests, record)
		score := hf.calculateScore(profile)
		profile.lastScore.Store(int32(score))
		profile.mu.Unlock()

		if score >= hf.config.BanScore {
			profile.bannedUntil.Store(time.Now().Add(hf.config.BanDuration).Unix())
			hf.totalBlocked.Add(1)

			log.Printf("[HTTPFlood] BAN: %s (score: %d)", ip, score)

			go banIPWithIPSet(hf.config.IPSetName, ip, int(hf.config.BanDuration.Seconds()))

			if hf.config.OnBan != nil {
				go hf.config.OnBan(ip, score, "HTTP Flood - High Score")
			}

			http.Error(w, "Access Denied", http.StatusForbidden)
			return
		}

		if score >= hf.config.ChallengeScore {
			hf.totalChallenged.Add(1)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(challengeHTML))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (hf *HTTPFlood) calculateScore(profile *IPProfile) int {
	now := time.Now()
	cutoff := now.Add(-hf.config.WindowSize)

	var recent []requestRecord
	for _, r := range profile.requests {
		if r.timestamp.After(cutoff) {
			recent = append(recent, r)
		}
	}
	profile.requests = recent

	if len(recent) == 0 {
		return 0
	}

	score := 0.0

	rateScore := float64(len(recent)) / float64(hf.config.MaxReqPerWindow) * 100
	if rateScore > 100 {
		rateScore = 100
	}
	score += rateScore * 0.30

	pathCounts := make(map[string]int)
	for _, r := range recent {
		pathCounts[r.path]++
	}

	var entropyScore float64
	if len(pathCounts) == 1 && len(recent) > 10 {
		entropyScore = 90
	} else {
		entropy := shannonEntropy(pathCounts, len(recent))
		maxEntropy := math.Log2(float64(max(len(pathCounts), 2)))
		if maxEntropy > 0 {
			normalized := entropy / maxEntropy
			entropyScore = (1 - normalized) * 100
		}
	}
	score += entropyScore * 0.20

	uaScore := hf.analyzeUserAgent(recent)
	score += uaScore * 0.20

	timingScore := hf.analyzeTimingPattern(recent)
	score += timingScore * 0.15

	connRatio := float64(profile.activeConns.Load()) / float64(hf.config.MaxConnsPerIP) * 100
	if connRatio > 100 {
		connRatio = 100
	}
	score += connRatio * 0.15

	return int(score)
}

func (hf *HTTPFlood) analyzeUserAgent(records []requestRecord) float64 {
	if len(records) == 0 {
		return 0
	}

	uaCounts := make(map[string]int)
	emptyUA := 0

	for _, r := range records {
		ua := r.userAgent
		if ua == "" {
			emptyUA++
		}
		uaCounts[ua]++
	}

	score := 0.0

	emptyRatio := float64(emptyUA) / float64(len(records))
	score += emptyRatio * 50

	botPatterns := []string{
		"python-requests", "go-http-client", "curl/",
		"wget/", "httpclient", "java/", "libwww",
		"bot", "spider", "crawler", "scan",
		"nikto", "sqlmap", "nmap", "masscan",
		"hello, world", "mozilla/5.0 (compatible;)",
	}

	for ua := range uaCounts {
		uaLower := strings.ToLower(ua)
		for _, pattern := range botPatterns {
			if strings.Contains(uaLower, pattern) {
				score += 30
				break
			}
		}
	}

	if len(uaCounts) == 1 && len(records) > 20 {
		score += 20
	}

	if score > 100 {
		score = 100
	}
	return score
}

func (hf *HTTPFlood) analyzeTimingPattern(records []requestRecord) float64 {
	if len(records) < 5 {
		return 0
	}

	var intervals []float64
	for i := 1; i < len(records); i++ {
		diff := records[i].timestamp.Sub(records[i-1].timestamp).Milliseconds()
		intervals = append(intervals, float64(diff))
	}

	if len(intervals) < 3 {
		return 0
	}

	var sum float64
	for _, v := range intervals {
		sum += v
	}
	mean := sum / float64(len(intervals))

	var variance float64
	for _, v := range intervals {
		diff := v - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(len(intervals)))

	if mean == 0 {
		return 80
	}

	cv := stddev / mean

	// CV < 0.1 = çok düzenli (bot)
	// CV > 0.5 = düzensiz (insan)
	// Ek olarak şunu ekleyeyim yeni geliştirilecek algoritmalar bunlara göre yazılabilir bunlar değişebilir değerler -Andre

	if cv < 0.1 {
		return 90
	} else if cv < 0.2 {
		return 60
	} else if cv < 0.3 {
		return 30
	}

	return 0
}

func shannonEntropy(counts map[string]int, total int) float64 {
	if total == 0 {
		return 0
	}

	var entropy float64
	for _, count := range counts {
		p := float64(count) / float64(total)
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

func extractClientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i != -1 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if i := strings.LastIndex(r.RemoteAddr, ":"); i != -1 {
		return r.RemoteAddr[:i]
	}
	return r.RemoteAddr
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (hf *HTTPFlood) cleanup() {
	ticker := time.NewTicker(hf.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hf.stopCh:
			return
		case <-ticker.C:
			now := time.Now()
			cutoff := now.Add(-hf.config.WindowSize * 2)

			hf.profiles.Range(func(key, value any) bool {
				profile := value.(*IPProfile)
				profile.mu.Lock()

				var fresh []requestRecord
				for _, r := range profile.requests {
					if r.timestamp.After(cutoff) {
						fresh = append(fresh, r)
					}
				}
				profile.requests = fresh

				isEmpty := len(fresh) == 0 &&
					profile.activeConns.Load() == 0 &&
					(profile.bannedUntil.Load() == 0 || time.Now().Unix() > profile.bannedUntil.Load())

				profile.mu.Unlock()

				if isEmpty {
					hf.profiles.Delete(key)
				}

				return true
			})
		}
	}
}

func (hf *HTTPFlood) Stats() (checked, blocked, challenged int64) {
	return hf.totalChecked.Load(), hf.totalBlocked.Load(), hf.totalChallenged.Load()
}

const challengeHTML = `<!DOCTYPE html>
<html lang="tr">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="robots" content="noindex, nofollow">
<title>Güvenlik Kontrolü — Topluyo</title>
<style>
  @import url('https://fonts.googleapis.com/css2?family=Nunito:ital,wght@0,200..1000;1,200..1000&display=swap');
  *{box-sizing:border-box;margin:0;padding:0}
  html,body{height:100%}
  body{
    font-family:"Nunito",sans-serif;
    font-optical-sizing:auto;
    --primary:#DE11BA;
    --primary-dark:#AB22ED;
    display:flex;
    flex-direction:column;
    align-items:center;
    justify-content:center;
    gap:1em;
    text-align:center;
    padding:1.5rem;
  }
  html.theme-dark body{background:#0a0a0c;color:#fff}
  html.theme-light body{background:#f1f3f4;color:#202124}

  .mascot{width:88px;max-width:20vh}
  h1{font-size:1.3rem;font-weight:800}
  p{font-size:.9rem;opacity:.6;max-width:300px}

  .spinner{
    width:36px;height:36px;
    border:3px solid rgba(222,17,186,0.15);
    border-top-color:var(--primary);
    border-radius:50%;
    animation:spin .9s linear infinite;
  }
  @keyframes spin{to{transform:rotate(360deg)}}

  .bar{width:200px;height:4px;border-radius:99px;overflow:hidden;background:rgba(222,17,186,0.12)}
  .bar-fill{height:100%;width:35%;border-radius:99px;
    background:linear-gradient(90deg,var(--primary),var(--primary-dark));
    animation:slide 1.1s ease-in-out infinite}
  @keyframes slide{0%{transform:translateX(-120%)}100%{transform:translateX(390%)}}
  body.verified .bar-fill{width:100%;animation:none}
  body.verified .spinner{display:none}

  .meta{font-size:.75rem;opacity:.35;font-variant-numeric:tabular-nums}

  @media (prefers-reduced-motion:reduce){.spinner,.bar-fill{animation:none!important}}
</style>
</head>
<body id="body">
<script>document.documentElement.classList.add(localStorage._theme=="theme-light"?"theme-light":"theme-dark")</script>

<img class="mascot" src="https://cdn.topluyo.com/kanka/kanka-color-solid.svg" alt="Topluyo">
<div class="spinner" id="spinner"></div>
<h1 id="title">Güvenlik Kontrolü</h1>
<p id="desc">İsteğiniz doğrulanıyor, lütfen bekleyin...</p>
<div class="bar"><div class="bar-fill" id="bar-fill"></div></div>
<p class="meta" id="attempt"></p>

<noscript><p>Bu kontrol için JavaScript gerekli. Lütfen etkinleştirip sayfayı yenileyin.</p></noscript>

<script>
(function(){
  "use strict";
  var SEED = window.__TLY_SEED__ || (new URLSearchParams(location.search)).get("seed") || (crypto.randomUUID ? crypto.randomUUID() : String(Date.now()+Math.random()));
  var DIFFICULTY = window.__TLY_DIFFICULTY__ || 4;
  var body = document.getElementById("body");
  var desc = document.getElementById("desc");
  var title = document.getElementById("title");
  var attemptEl = document.getElementById("attempt");

  function sha256Hex(str){
    var buf = new TextEncoder().encode(str);
    return crypto.subtle.digest("SHA-256", buf).then(function(hash){
      var bytes = new Uint8Array(hash), hex = "";
      for (var i = 0; i < bytes.length; i++) hex += bytes[i].toString(16).padStart(2, "0");
      return hex;
    });
  }

  function solve(seed, difficulty){
    var prefix = "0".repeat(difficulty);
    var nonce = 0;
    function attempt(){
      return sha256Hex(seed + nonce).then(function(hash){
        if (hash.indexOf(prefix) === 0) return { nonce: nonce, hash: hash };
        nonce++;
        if (nonce % 400 === 0){
          attemptEl.textContent = "Deneme: " + nonce.toLocaleString("tr-TR");
          return new Promise(function(r){ setTimeout(r, 0); }).then(attempt);
        }
        return attempt();
      });
    }
    return attempt();
  }

  if (!window.crypto || !window.crypto.subtle){
    title.textContent = "Doğrulanamadı";
    desc.textContent = "Tarayıcınız bu kontrolü desteklemiyor. Lütfen güncelleyin.";
    return;
  }

  solve(SEED, DIFFICULTY).then(function(result){
    document.cookie = "tly_chk=" + result.hash + "." + result.nonce + "; path=/; max-age=300; SameSite=Lax";
    body.classList.add("verified");
    title.textContent = "Doğrulandı";
    desc.textContent = "Yönlendiriliyorsunuz...";
    setTimeout(function(){
      var params = new URLSearchParams(location.search);
      location.replace(params.get("redirect") || document.referrer || "/");
    }, 500);
  }).catch(function(){
    desc.textContent = "Bir hata oluştu, yeniden deneniyor...";
    setTimeout(function(){ location.reload(); }, 3000);
  });
})();
</script>
</body>
</html>`

// Bu kısım AI ile oluşturuldu challenge sayfası bilginize
