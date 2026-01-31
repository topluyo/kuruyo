package main

import (
	"encoding/json"
	"github.com/hpcloud/tail"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"fmt"
	"net/http"
	"bytes"
	"io"
	"strconv"
)

var (
	LOG_FILE                 = ""   // log=
	BLOCK_LOG_FILE           = ""   // [log].blocked.log
	BLOCK_JSON_FILE          = ""   // [log].blocked.json
	CLOUDFLARE_CONFIG_FILE   = ""   // cf=
	REQUEST         = 10
	PER_TIME        = 2 * time.Second
	BLOCK_TIME      = 2 * time.Second
)

/* ---------------- TYPES ---------------- */

type IPStat struct {
	mu      sync.Mutex
	events  []time.Time
	blocked bool
}

type BlockInfo struct {
	blockedAt int64 // unix timestamp (atomic)
}

/* ---------------- GLOBALS ---------------- */

var (
	ipStats sync.Map

	blockedMu    sync.Mutex
	Blocked      = make(map[string]*BlockInfo)
	blockedDirty atomic.Bool

	blockLogMu   sync.Mutex
	
)

/* ---------------- MAIN ---------------- */

func main() {
	if( argument("log") == "" && argument("clear")=="" ){
		row("USAGE")
		write(" log=/path/to/file.log")
		write(" cf=/web/config/.cloudflare.log")
		write(" clear=yes")
		write(" r=100  request")
		write(" s=120  in last period")
		write(" b=3600 block")
		
		write()
		return
	}





	LOG_FILE := argument("log")
	BLOCK_LOG_FILE = LOG_FILE + ".block.log"
	CLOUDFLARE_CONFIG_FILE := argument("cf","/web/config/.cloudflare.json")
	BLOCK_JSON_FILE = LOG_FILE + ".block.json"

	loadCloudflareConfig(CLOUDFLARE_CONFIG_FILE)


	if(argument("clear")!=""){
		ClearAllBlockedIPs(cfConfig.CFZoneID,cfConfig.CFAuthToken)
	}

	if(argument("log")!=""){

		table(" BozKurt Started")
		write(" * log        :", LOG_FILE)
		write(" * block log  :", BLOCK_LOG_FILE)
		write(" * block json :", BLOCK_JSON_FILE)

		r := argument("r","100")
		s := argument("s","120")
		b := argument("b","3600")

		REQUEST    = toInt(r)
		PER_TIME   = time.Duration(toInt(s)) * time.Second
		BLOCK_TIME = time.Duration(toInt(b)) * time.Second

		write(" = r REQUEST    :", REQUEST)
		write(" = s PER_TIME   :", PER_TIME)
		write(" = b BLOCK_TIME :", BLOCK_TIME)
		
		write("")

		loadBlockedFromFile(BLOCK_JSON_FILE)
		startUnblockTimer()
		startBlockedWriter()

		t, err := tail.TailFile(LOG_FILE, tail.Config{
			Follow:    true,
			ReOpen:   true,
			MustExist: true,
			Poll:     true,
			Location: &tail.SeekInfo{Offset: 0, Whence: os.SEEK_END},
		})
		if err != nil {
			log.Fatalf("tail error: %v", err)
		}

		checkUnblock()

		for line := range t.Lines {
			handleLine(line.Text)
		}
	}
}

/* ---------------- LOG HANDLING ---------------- */

func handleLine(line string) {
	if !strings.HasPrefix(line, "RATEOVER") {
		return
	}

	parts := strings.Split(line, ",")
	if len(parts) < 3 {
		return
	}

	ip := strings.TrimSpace(parts[2])

	val, _ := ipStats.LoadOrStore(ip, &IPStat{})
	stat := val.(*IPStat)

	stat.mu.Lock()
	defer stat.mu.Unlock()

	// 🔥 HALA ATAK VARSA BLOCK SÜRESİNİ UZAT
	if stat.blocked {
		registerBlock(ip)
		return
	}

	now := time.Now()
	stat.events = append(stat.events, now)

	// window temizliği
	cutoff := now.Add(-PER_TIME)
	idx := 0
	for _, t := range stat.events {
		if t.After(cutoff) {
			stat.events[idx] = t
			idx++
		}
	}
	stat.events = stat.events[:idx]

	if len(stat.events) >= REQUEST {
		stat.blocked = true
		registerBlock(ip)
	}
}

/* ---------------- BLOCK / UNBLOCK ---------------- */

func registerBlock(ip string) {
	now := time.Now().Unix()

	blockedMu.Lock()
	defer blockedMu.Unlock()

	// zaten blockluysa sadece süre uzat
	if info, ok := Blocked[ip]; ok {
		atomic.StoreInt64(&info.blockedAt, now)
		blockedDirty.Store(true)
		return
	}

	Blocked[ip] = &BlockInfo{
		blockedAt: now,
	}

	blockedDirty.Store(true)

	// ⛔ BLOCK LOG
	writeBlockLog("BLOCK", ip)

	go BlockAPI(ip)
}

func startUnblockTimer() {
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for range ticker.C {
			checkUnblock()
		}
	}()
}

func checkUnblock() {
	now := time.Now().Unix()

	blockedMu.Lock()
	defer blockedMu.Unlock()

	for ip, info := range Blocked {
		blockTime := atomic.LoadInt64(&info.blockedAt)

		if time.Duration(now-blockTime)*time.Second >= BLOCK_TIME{
			go func(ip string) {
				if UnBlockAPI(ip) {
					blockedMu.Lock()
					delete(Blocked, ip)
					blockedDirty.Store(true)
					blockedMu.Unlock()
					writeBlockLog("UNBLOCK", ip)
				} else {
					write("[UNBLOCK FAILED]", ip)
				}
			}(ip)
			blockedDirty.Store(true)
		}
	}
}

/* ---------------- BLOCK LOG FILE ---------------- */

func writeBlockLog(action, ip string) {
	blockLogMu.Lock()
	defer blockLogMu.Unlock()

	f, err := os.OpenFile(BLOCK_LOG_FILE, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	ts := time.Now().Format("2006-01-02 15:04:05")
	line := action + ", " + ts + ", " + ip + "\n"
	_, _ = f.WriteString(line)
}

/* ---------------- blocked.json ---------------- */

func loadBlockedFromFile(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	raw := map[string]int64{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}

	blockedMu.Lock()
	defer blockedMu.Unlock()

	for ip, ts := range raw {
		Blocked[ip] = &BlockInfo{
			blockedAt: ts,
		}
	}
}

func startBlockedWriter() {
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for range ticker.C {
			if !blockedDirty.Load() {
				continue
			}
			writeBlockedToFile(BLOCK_JSON_FILE)
		}
	}()
}

func writeBlockedToFile(path string) {
	blockedMu.Lock()
	defer blockedMu.Unlock()

	data := make(map[string]int64, len(Blocked))
	for ip, info := range Blocked {
		data[ip] = atomic.LoadInt64(&info.blockedAt)
	}

	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}

	tmp := path + ".tmp"
	if err := ioutil.WriteFile(tmp, buf, 0644); err != nil {
		return
	}

	_ = os.Rename(tmp, path)
	blockedDirty.Store(false)
}

/* ---------------- API PLACEHOLDERS ---------------- */


type CFConfig struct {
	CFZoneID    string `json:"CFZoneID"`
	CFAuthToken string `json:"CFAuthToken"`
}

var cfConfig CFConfig

func loadCloudflareConfig(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &cfConfig)
	if err != nil {
		return err
	}

	if cfConfig.CFZoneID == "" || cfConfig.CFAuthToken == "" {
		return fmt.Errorf("cloudflare config missing fields")
	}

	return nil
}
var (
	HTTPTimeout  = 10 * time.Second
)

// ---------------- BLOCK ----------------



func BlockAPI(ip string) {
	write("[BLOCK] ",ip)
	BlockOnCloudFlare(ip,cfConfig.CFZoneID,cfConfig.CFAuthToken)
}

func UnBlockAPI(ip string) bool {
	write("[UNBLOCK TRY] ", ip)
	return UnBlockOnCloudFlare(ip, cfConfig.CFZoneID, cfConfig.CFAuthToken)
}


type IPRule struct {
    Mode          string `json:"mode"`
    Notes         string `json:"notes"`
    Configuration struct {
        Target string `json:"target"`
        Value  string `json:"value"`
    } `json:"configuration"`
}

type ListResponse struct {
    Result []struct {
        ID string `json:"id"`
    } `json:"result"`
}


func BlockOnCloudFlare(ip string, zoneID string, apiToken string) {
    url := fmt.Sprintf(
        "https://api.cloudflare.com/client/v4/zones/%s/firewall/access_rules/rules",
        zoneID,
    )

    rule := IPRule{
        Mode:  "block",
        Notes: "Blocked via API",
    }
    rule.Configuration.Target = "ip"
    rule.Configuration.Value = ip

    payload, _ := json.Marshal(rule)

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
    if err != nil {
        fmt.Println("Request hatası:", err)
        return
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+apiToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        write("[APIERROR]", err)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        write("[BLOCKED]", ip)
    } else {
        write("[ERROR]", resp.StatusCode)
    }
}

func UnBlockOnCloudFlare(ip string, zoneID string, apiToken string) bool {
    client := &http.Client{}

    listURL := fmt.Sprintf(
        "https://api.cloudflare.com/client/v4/zones/%s/firewall/access_rules/rules?configuration.value=%s",
        zoneID,
        ip,
    )

    req, _ := http.NewRequest("GET", listURL, nil)
    req.Header.Set("Authorization", "Bearer "+apiToken)

    resp, err := client.Do(req)
    if err != nil {
        write("[LISTERROR]", err)
        return false
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    var listResp ListResponse
    json.Unmarshal(body, &listResp)

    if len(listResp.Result) == 0 {
        write("[NONBLOCKED]", ip)
        return true // zaten block yok → temiz say
    }

    ruleID := listResp.Result[0].ID

    deleteURL := fmt.Sprintf(
        "https://api.cloudflare.com/client/v4/zones/%s/firewall/access_rules/rules/%s",
        zoneID,
        ruleID,
    )

    delReq, _ := http.NewRequest("DELETE", deleteURL, nil)
    delReq.Header.Set("Authorization", "Bearer "+apiToken)

    delResp, err := client.Do(delReq)
    if err != nil {
        write("[DELETEERROR]", err)
        return false
    }
    defer delResp.Body.Close()

    if delResp.StatusCode >= 200 && delResp.StatusCode < 300 {
        write("[UNBLOCKED]", ip)
        return true
    }

    write("[UNBLOCKERROR]", delResp.StatusCode)
    return false
}





func ClearAllBlockedIPs(zoneID string, apiToken string) {
	client := &http.Client{Timeout: HTTPTimeout}

	page := 1
	perPage := 50

	for {
		listURL := fmt.Sprintf(
			"https://api.cloudflare.com/client/v4/zones/%s/firewall/access_rules/rules?page=%d&per_page=%d",
			zoneID,
			page,
			perPage,
		)

		req, err := http.NewRequest("GET", listURL, nil)
		if err != nil {
			write("[LIST REQUEST ERROR]", err)
			return
		}

		req.Header.Set("Authorization", "Bearer "+apiToken)

		resp, err := client.Do(req)
		if err != nil {
			write("[LIST ERROR]", err)
			return
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			Result []struct {
				ID   string `json:"id"`
				Mode string `json:"mode"`
				Configuration struct {
					Target string `json:"target"`
					Value  string `json:"value"`
				} `json:"configuration"`
			} `json:"result"`
			ResultInfo struct {
				TotalPages int `json:"total_pages"`
			} `json:"result_info"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			write("[JSON ERROR]", err)
			return
		}

		for _, rule := range result.Result {
			if rule.Mode == "block" && rule.Configuration.Target == "ip" {
				deleteURL := fmt.Sprintf(
					"https://api.cloudflare.com/client/v4/zones/%s/firewall/access_rules/rules/%s",
					zoneID,
					rule.ID,
				)

				delReq, _ := http.NewRequest("DELETE", deleteURL, nil)
				delReq.Header.Set("Authorization", "Bearer "+apiToken)

				delResp, err := client.Do(delReq)
				if err != nil {
					write("[DELETE ERROR]", err)
					continue
				}
				delResp.Body.Close()

				if delResp.StatusCode >= 200 && delResp.StatusCode < 300 {
					write("[CLEARED]", rule.Configuration.Value)
				} else {
					write("[CLEAR FAILED]", rule.Configuration.Value, delResp.StatusCode)
				}
			}
		}

		if page >= result.ResultInfo.TotalPages {
			break
		}
		page++
	}

	write("[DONE] All blocked IPs cleared")
}


/* ---------------- HELPERS ---------------- */

func argument(key string, defaults ...string) string {
	response := ""
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, key+"=") {
			response = strings.SplitN(arg, "=", 2)[1]
			response = strings.Trim(response, "\"")
			return response
		}
	}
	if len(defaults) > 0 {
		return defaults[0]
	}
	return ""
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
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) +
		text +
		strings.Repeat(" ", width-len(text)-padding)
}

func table(name string) {
	write("╔════════════════════════════════════════════════╗")
	write("║" + center(name) + "║")
	write("╚════════════════════════════════════════════════╝")
}

func row(name string) {
	write("┌────────────────────────────────────────────────┐")
	write("│" + center(name) + "│")
	write("└────────────────────────────────────────────────┘")
}


func toInt(number string) int {
	intNumber, err := strconv.Atoi(number)
	if err != nil {
		return 0
	}
	return intNumber
}




