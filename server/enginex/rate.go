package main

import (
	"sync"
	"time"
)

type Rate struct {
	Request int
	Period  int64 // saniye
	Wait    int   // saniye

	
  // shards 
	Blocked sync.Map
}

func (r *Rate) currentWindow() int64 {
	return atomic.LoadInt64(&nowNano) / r.periodNano
}



type RateItem struct {
	Count  int
	Window int
}




var nowNano int64

func Init() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			atomic.StoreInt64(&nowNano, time.Now().UnixNano())
			<-ticker.C
		}
	}()
}

func NewRate(shardSize int, rate Rate) *Rate {
	return &Rate{
		data:       NewShard[RateItem](shardSize),
		rate:       rate,
		periodNano: int64(rate.Period) * int64(time.Second),
	}
}

