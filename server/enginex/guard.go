package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"
)

type Block[T any] struct {
	Size int64
	List []T
}

func NewBlock[T any](pow int) *Block[T] {
	return &Block[T]{
		Size: (1 << pow) - 1,
		List: make([]T, 1<<pow,1<<pow),
	}
}

func (b *Block[T]) Get(key uint64) *T {
	hash := int64(key) & b.Size
	return &b.List[hash]
}

/* ---------------- RATE LIMIT ---------------- */

type Rate struct {
	Count  int32
	Window int64
}

type Window struct {
	Value int64
}

func (w *Window) Get() int64 {
	return atomic.LoadInt64(&w.Value)
}

func NewWindow(period int64) *Window {
	w := &Window{}
	ticker := time.NewTicker(time.Second)

	go func() {
		for range ticker.C {
			atomic.StoreInt64(&w.Value, time.Now().Unix()/period)
		}
	}()
	return w
}

/* ---------------- UTILS ---------------- */

func randomIP() uint64 {
	return uint64(rand.Uint32())
}

/* ---------------- MAIN ---------------- */

func main() {
	rand.Seed(time.Now().UnixNano())

	const (
		blockPow = 20         // ~4M buckets
		limit    = 5         // max requests
		period   = int64(10)    // seconds
		requests = 5_000_000   // benchmark load
	)

	Status()

	block := NewBlock[Rate](blockPow)
	window := NewWindow(period)

	start := time.Now()

	var allowed uint64
	var blocked uint64

	for i := 0; i < requests; i++ {
		go func(){
			ip := randomIP()
			rate := block.Get(ip)

			nowWindow := window.Get()

			if atomic.LoadInt64(&rate.Window) != nowWindow {
				atomic.StoreInt64(&rate.Window, nowWindow)
				atomic.StoreInt32(&rate.Count, 0)
			}

			if atomic.AddInt32(&rate.Count, 1) <= limit {
				allowed++
			} else {
				blocked++
			}
		}()
	}

	Status()

	time.Sleep(time.Second * 2)

	elapsed := time.Since(start)

	fmt.Println("---- RESULT ----")
	fmt.Printf("Requests: %d\n", requests)
	fmt.Printf("Allowed : %d\n", allowed)
	fmt.Printf("Blocked : %d\n", blocked)
	fmt.Printf("Time    : %v\n", elapsed)
	fmt.Printf("RPS     : %d\n", int64(requests)/elapsed.Milliseconds()*1000)

	Status()
	
	time.Sleep(time.Second * 2)
	
}


func Status(){
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("\n---- MEMORY ----\n")
	fmt.Printf("Alloc      = %d MB\n", m.Alloc/1024/1024)
	fmt.Printf("TotalAlloc = %d MB\n", m.TotalAlloc/1024/1024)
	fmt.Printf("Sys        = %d MB\n", m.Sys/1024/1024)
	fmt.Printf("NumGC      = %d\n", m.NumGC)
}