package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type PerformanceReport struct {
	Name            string
	TotalRequests   int
	AllowedRequests int64
	BlockedRequests int64
  IPCount         int
	Duration        time.Duration
	RPS             float64
	AvgLatency      time.Duration
}


func RateLimiterChecker() {



	loaded := Limiter.Load()

	limits := make([]*Limit, 0)

  limits  = append(limits, loaded.limits["hard_block"])

  /*
  for _, l := range loaded.limits {
		limits = append(limits, l)
	}
  */
 

	tests := []struct {
		name        string
		reqPerIP    int
		ipCount     int
		concurrency int
		ipPrefix    string
	}{
    {
			name:        "single aggressive traffic",
			reqPerIP:    1000000,
			ipCount:     10,
			concurrency: 100,
			ipPrefix:    "19.168.1",
		},
		{
			name:        "aggressive traffic",
			reqPerIP:    1000,
			ipCount:     10000,
			concurrency: 100,
			ipPrefix:    "192.168.1",
		},
		{
			name:        "normal traffic",
			reqPerIP:    4,
			ipCount:     2500000,
			concurrency: 100,
			ipPrefix:    "10.20.30",
		},
	}


	for _, tc := range tests {

		report := runRateLimiterTest(
			tc.name,
			tc.reqPerIP,
			tc.ipCount,
			tc.concurrency,
			limits,
			tc.ipPrefix,
		)


		fmt.Println("--------------------------------")
		fmt.Println("Rate Limiter Performance Report")
		fmt.Println("--------------------------------")

		fmt.Printf("Scenario        : %s\n", report.Name)
		fmt.Printf("Total Requests  : %d\n", report.TotalRequests)
    fmt.Printf("IP Count        : %d\n", report.IPCount)
		fmt.Printf("Allowed         : %d\n", report.AllowedRequests)
		fmt.Printf("Blocked         : %d\n", report.BlockedRequests)
		fmt.Printf("Duration        : %v\n", report.Duration)
		fmt.Printf("Requests/sec    : %.2f\n", report.RPS)
		fmt.Printf("Avg Latency     : %v\n", report.AvgLatency)

		fmt.Println("--------------------------------")
	}
}



func runRateLimiterTest(
	name string,
	reqPerIP int,
	ipCount int,
	concurrency int,
	limits []*Limit,
	ipPrefix string,

) PerformanceReport {


	totalRequests := ipCount * reqPerIP


	requests := make([]string, 0, totalRequests)


	for i := 0; i < ipCount; i++ {

		ip := fmt.Sprintf(
			"%s.%d",
			ipPrefix,
			i,
		)

		for j := 0; j < reqPerIP; j++ {
			requests = append(
				requests,
				ip,
			)
		}
	}


	// gerçek trafik dağılımına daha yakın olsun
	rand.Shuffle(
		len(requests),
		func(i, j int) {
			requests[i], requests[j] = requests[j], requests[i]
		},
	)


	requestCh := make(
		chan string,
		totalRequests,
	)


	for _, ip := range requests {
		requestCh <- ip
	}

	close(requestCh)



	var (
		allowed       int64
		blocked       int64
		totalLatency  int64
	)


	start := time.Now()


	var wg sync.WaitGroup


	worker := func() {

		defer wg.Done()


		for ip := range requestCh {

			reqStart := time.Now()


			ok := CheckRateLimiter(
				ip,
				limits,
			)


			atomic.AddInt64(
				&totalLatency,
				int64(time.Since(reqStart)),
			)


			if ok {
				atomic.AddInt64(
					&allowed,
					1,
				)

			} else {

				atomic.AddInt64(
					&blocked,
					1,
				)
			}
		}
	}



	for i := 0; i < concurrency; i++ {

		wg.Add(1)

		go worker()
	}



	wg.Wait()


	duration := time.Since(start)



	avgLatency := time.Duration(0)

	if totalRequests > 0 {

		avgLatency = time.Duration(
			totalLatency / int64(totalRequests),
		)
	}



	return PerformanceReport{
		Name: name,
		TotalRequests: totalRequests,
    IPCount: ipCount,
		AllowedRequests: allowed,
		BlockedRequests: blocked,
		Duration: duration,
		RPS: float64(totalRequests) / duration.Seconds(),
		AvgLatency: avgLatency,
	}
}
