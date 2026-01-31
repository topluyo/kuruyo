package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"log"
	"strconv"
	"golang.org/x/net/proxy"
)

/////////////////////////////
// CONFIG
/////////////////////////////

var (
	TestURL        = "https://example.com"
	TestPerProxy   = 2
	RequestTimeout = 10 * time.Second
	FailLimit      = 3
	MaxFastProxy  = 100
)

/////////////////////////////
// STRUCTS
/////////////////////////////

type ProxyClient struct {
	Addr     string
	Client   *http.Client
	Fail     int32
	Disabled int32
	Latency  time.Duration
}

type Response struct {
	Proxy    string
	Status   int
	Error    error
	Duration time.Duration
}

/////////////////////////////
// GLOBALS
/////////////////////////////

var (
	activeProxies []*ProxyClient
	rrIndex       uint64
)

/////////////////////////////
// LOAD PROXIES
/////////////////////////////

func LoadProxies(file string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var list []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		p := strings.TrimSpace(sc.Text())
		if p != "" {
			list = append(list, p)
		}
	}
	return list, sc.Err()
}

/////////////////////////////
// CREATE CLIENT
/////////////////////////////

func NewProxyClient(addr string) (*ProxyClient, error) {
	dialer, err := proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}

	dialCtx := func(ctx context.Context, network, target string) (net.Conn, error) {
		return dialer.Dial(network, target)
	}

	tr := &http.Transport{
		DialContext:         dialCtx,
		MaxIdleConns:        500,
		MaxIdleConnsPerHost: 500,
		IdleConnTimeout:    30 * time.Second,
	}

	return &ProxyClient{
		Addr: addr,
		Client: &http.Client{
			Transport: tr,
			Timeout:   RequestTimeout,
		},
	}, nil
}

/////////////////////////////
// VALIDATION & LATENCY
/////////////////////////////

func ValidateProxies(proxyList []string) {
	fmt.Println("🔍 Validating proxies...")

	var wg sync.WaitGroup
	mu := sync.Mutex{}

	for _, addr := range proxyList {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			pc, err := NewProxyClient(p)
			if err != nil {
				return
			}

			var totalLatency time.Duration
			success := 0

			for i := 0; i < TestPerProxy; i++ {
				start := time.Now()
				req, _ := http.NewRequest("GET", TestURL, nil)
				resp, err := pc.Client.Do(req)
				if err == nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					success++
					totalLatency += time.Since(start)
				}
			}

			if success == TestPerProxy {
				pc.Latency = totalLatency / time.Duration(TestPerProxy)
				mu.Lock()
				activeProxies = append(activeProxies, pc)
				mu.Unlock()
				fmt.Printf("[OK] %-22s avg %.2fs\n", p, pc.Latency.Seconds())
			} else {
				fmt.Printf("[FAIL] %s\n", p)
			}

		}(addr)
	}

	wg.Wait()
	fmt.Println("✅ Total working proxies:", len(activeProxies))
}

/////////////////////////////
// ROUND ROBIN
/////////////////////////////

func nextProxy() *ProxyClient {
	total := len(activeProxies)
	for i := 0; i < total; i++ {
		idx := int(atomic.AddUint64(&rrIndex, 1) % uint64(total))
		p := activeProxies[idx]
		if atomic.LoadInt32(&p.Disabled) == 0 {
			return p
		}
	}
	return nil
}

/////////////////////////////
// WORKER
/////////////////////////////

func worker(jobs <-chan string, results chan<- Response, wg *sync.WaitGroup) {
	defer wg.Done()

	for url := range jobs {

		pc := nextProxy()
		if pc == nil {
			results <- Response{Error: fmt.Errorf("no active proxy")}
			continue
		}

		start := time.Now()
		req, _ := http.NewRequest("GET", url, nil)
		resp, err := pc.Client.Do(req)
		duration := time.Since(start)

		if err != nil {
			if atomic.AddInt32(&pc.Fail, 1) >= int32(FailLimit) {
				atomic.StoreInt32(&pc.Disabled, 1)
				fmt.Println("[DISABLED]", pc.Addr)
			}
			results <- Response{Proxy: pc.Addr, Error: err, Duration: duration}
			continue
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		atomic.StoreInt32(&pc.Fail, 0)

		results <- Response{
			Proxy:    pc.Addr,
			Status:   resp.StatusCode,
			Duration: duration,
		}
	}
}

/////////////////////////////
// LOAD TEST
/////////////////////////////

func Test(url string, count int) {

	workers := runtime.NumCPU() * 8
	jobs := make(chan string, count)
	results := make(chan Response, count)

	var wg sync.WaitGroup
	var latencies []float64
	var mu sync.Mutex

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker(jobs, results, &wg)
	}

	start := time.Now()

	for i := 0; i < count; i++ {
		jobs <- url
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		if r.Error != nil {
			fmt.Printf("[+] %-22s [ERR] %-40v %6.2fs\n", r.Proxy, r.Error, r.Duration.Seconds())
		} else {
			fmt.Printf("[+] %-22s [%3d] %6.2fs\n", r.Proxy, r.Status, r.Duration.Seconds())
			mu.Lock()
			latencies = append(latencies, r.Duration.Seconds())
			mu.Unlock()
		}
	}

	// P50, P95, P99
	if len(latencies) > 0 {
		sort.Float64s(latencies)
		p := func(q float64) float64 {
			idx := int(float64(len(latencies)-1) * q)
			return latencies[idx]
		}
		fmt.Printf("\n⏱ P50: %.2fs\n", p(0.50))
		fmt.Printf("⏱ P95: %.2fs\n", p(0.95))
		fmt.Printf("⏱ P99: %.2fs\n", p(0.99))
	}

	fmt.Println("⏱ Total time:", time.Since(start))
}

/////////////////////////////
// MAIN
/////////////////////////////

func main() {

	if(argument("r")==""){
		row("STRESS TEST")
		write(" r=50000  request ")
		write(" p=50     private ip ")
		return
	}
	r := int32(toInt(argument("r","10000")))

	MaxFastProxy = toInt(argument("p","50"))
	


	list, err := LoadProxies("proxies.txt")
	if err != nil {
		panic(err)
	}

	ValidateProxies(list)

	if len(activeProxies) == 0 {
		panic("no working proxies")
	}

	// En hızlı 20 proxyyi seç
	sort.Slice(activeProxies, func(i, j int) bool {
		return activeProxies[i].Latency < activeProxies[j].Latency
	})
	if len(activeProxies) > MaxFastProxy {
		activeProxies = activeProxies[:MaxFastProxy]
	}

	fmt.Printf("⚡ Using %d fastest proxies for load test\n", len(activeProxies))

	write(r)

	Test("https://topluyo.com/RATE/TEST", 5000)
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



