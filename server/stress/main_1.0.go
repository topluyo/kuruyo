package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/proxy"
)

//////////////////////////
// CONFIG
//////////////////////////

const (
	MaxWorkers     = 100        // worker pool
	FailThreshold  = 3          // kaç hatadan sonra proxy disable
	RequestTimeout = 10 * time.Second
)

//////////////////////////
// RESPONSE STRUCT
//////////////////////////

type Response struct {
	Proxy      string
	StatusCode int
	Duration   time.Duration
	Preview64  string
	Error      error
}

//////////////////////////
// PROXY CLIENT
//////////////////////////

type ProxyClient struct {
	Addr      string
	Client    *http.Client
	FailCount int32
	Disabled  int32 // atomic bool
}

var (
	proxies []*ProxyClient
	rrIndex uint64
)

//////////////////////////
// LOAD PROXIES FROM FILE
//////////////////////////

func LoadProxies(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var list []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			list = append(list, line)
		}
	}
	return list, scanner.Err()
}

//////////////////////////
// INIT PROXY CLIENTS
//////////////////////////

func InitProxies(proxyAddrs []string) error {

	for _, addr := range proxyAddrs {

		dialer, err := proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
		if err != nil {
			fmt.Println("[SKIP] proxy init failed:", addr)
			continue
		}

		dialContext := func(ctx context.Context, network, target string) (net.Conn, error) {
			return dialer.Dial(network, target)
		}

		transport := &http.Transport{
			DialContext:         dialContext,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:    30 * time.Second,
		}

		client := &http.Client{
			Transport: transport,
			Timeout:   RequestTimeout,
		}

		proxies = append(proxies, &ProxyClient{
			Addr:   addr,
			Client: client,
		})
	}

	if len(proxies) == 0 {
		return fmt.Errorf("no valid proxies loaded")
	}

	fmt.Println("[OK] Loaded proxies:", len(proxies))
	return nil
}

//////////////////////////
// GET NEXT HEALTHY PROXY
//////////////////////////

func nextProxy() *ProxyClient {
	total := len(proxies)

	for i := 0; i < total; i++ {
		idx := int(atomic.AddUint64(&rrIndex, 1) % uint64(total))
		p := proxies[idx]

		if atomic.LoadInt32(&p.Disabled) == 0 {
			return p
		}
	}
	return nil
}

//////////////////////////
// WORKER
//////////////////////////

func worker(id int, jobs <-chan string, results chan<- Response, wg *sync.WaitGroup) {
	defer wg.Done()

	for url := range jobs {

		pc := nextProxy()
		if pc == nil {
			results <- Response{
				Error: fmt.Errorf("no active proxy available"),
			}
			continue
		}

		start := time.Now()

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			results <- Response{Proxy: pc.Addr, Error: err}
			continue
		}

		resp, err := pc.Client.Do(req)
		if err != nil {
			fail := atomic.AddInt32(&pc.FailCount, 1)

			if fail >= FailThreshold {
				atomic.StoreInt32(&pc.Disabled, 1)
				fmt.Println("[DISABLED] proxy:", pc.Addr)
			}

			results <- Response{
				Proxy:    pc.Addr,
				Duration: time.Since(start),
				Error:    err,
			}
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		atomic.StoreInt32(&pc.FailCount, 0)

		preview := string(body)
		if len(preview) > 64 {
			preview = preview[:64]
		}

		results <- Response{
			Proxy:      pc.Addr,
			StatusCode: resp.StatusCode,
			Duration:   time.Since(start),
			Preview64:  preview,
		}
	}
}

//////////////////////////
// TEST FUNCTION
//////////////////////////

func Test(url string, count int) {

	jobs := make(chan string, count)
	results := make(chan Response, count)

	var wg sync.WaitGroup

	for i := 0; i < MaxWorkers; i++ {
		wg.Add(1)
		go worker(i, jobs, results, &wg)
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
			fmt.Printf("[ERR] proxy=%s err=%v\n", r.Proxy, r.Error)
		} else {
			fmt.Printf("[OK] proxy=%s status=%d time=%v\n",
				r.Proxy, r.StatusCode, r.Duration)
		}
	}

	fmt.Println("TOTAL TIME:", time.Since(start))
}

//////////////////////////
// MAIN
//////////////////////////

func main() {

	proxyList, err := LoadProxies("proxies.txt")
	if err != nil {
		panic(err)
	}

	if err := InitProxies(proxyList); err != nil {
		panic(err)
	}

	Test("https://topluyo.com/TEST/RATE", 10000)
}
