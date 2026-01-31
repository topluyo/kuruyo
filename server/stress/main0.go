package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/proxy"
)

// ==================================
// RESPONSE STRUCT
// ==================================
type Response struct {
	Proxy      string
	StatusCode int
	Duration   time.Duration
	Preview64  string
	Error      error
}

// ==================================
// PROXY CLIENT POOL
// ==================================
type ProxyClient struct {
	Addr   string
	Client *http.Client
}

var (
	clients []*ProxyClient
	rrIndex uint64
)

// ==================================
// INIT PROXY CLIENTS
// ==================================
func InitProxies(proxyAddrs []string) error {

	for _, addr := range proxyAddrs {

		dialer, err := proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
		if err != nil {
			return fmt.Errorf("proxy %s init error: %w", addr, err)
		}

		dialContext := func(ctx context.Context, network, target string) (net.Conn, error) {
			return dialer.Dial(network, target)
		}

		transport := &http.Transport{
			DialContext:         dialContext,
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:    30 * time.Second,
		}

		client := &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		}

		clients = append(clients, &ProxyClient{
			Addr:   addr,
			Client: client,
		})
	}

	return nil
}

// ==================================
// ROUND ROBIN CLIENT PICK
// ==================================
func nextClient() *ProxyClient {
	i := atomic.AddUint64(&rrIndex, 1)
	return clients[int(i)%len(clients)]
}

// ==================================
// ASYNC REQUEST
// ==================================
func RequestAsync(targetURL string, out chan<- Response) {

	go func() {

		pc := nextClient()
		start := time.Now()

		req, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			out <- Response{Proxy: pc.Addr, Error: err}
			return
		}

		resp, err := pc.Client.Do(req)
		if err != nil {
			out <- Response{
				Proxy:    pc.Addr,
				Duration: time.Since(start),
				Error:    err,
			}
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			out <- Response{
				Proxy:      pc.Addr,
				StatusCode: resp.StatusCode,
				Duration:   time.Since(start),
				Error:      err,
			}
			return
		}

		preview := string(body)
		if len(preview) > 64 {
			preview = preview[:64]
		}

		out <- Response{
			Proxy:      pc.Addr,
			StatusCode: resp.StatusCode,
			Duration:   time.Since(start),
			Preview64:  preview,
		}
	}()
}

// ==================================
// TEST FUNCTION
// ==================================
func Test(url string, count int) {

	resultChan := make(chan Response, count)
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < count; i++ {
		wg.Add(1)
		RequestAsync(url, resultChan)

		go func() {
			defer wg.Done()
			resp := <-resultChan

			if resp.Error != nil {
				fmt.Printf("[ERR] proxy=%s err=%v\n", resp.Proxy, resp.Error)
				return
			}

			fmt.Printf(
				"[OK] proxy=%s status=%d time=%v preview=\"%s\"\n",
				resp.Proxy,
				resp.StatusCode,
				resp.Duration,
				resp.Preview64,
			)
		}()
	}

	wg.Wait()
	fmt.Println("TOTAL TIME:", time.Since(start))
}

// ==================================
// MAIN
// ==================================
func main() {

	proxies := []string{
		"185.162.228.121:1080",
		"185.162.228.122:1080",
		// istediğin kadar ekle
	}

	if err := InitProxies(proxies); err != nil {
		panic(err)
	}

	Test("https://example.com/notfound", 1000)
}
