package main

import(
	"sync"
	"time"
)

type RateLimiter struct {
    mu      sync.Mutex
    requests map[string][]time.Time
    limit   int
    window  time.Duration
}

func (r *RateLimiter) Allow(userID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-r.window)

    timestamps := r.requests[userID]

    // Window dışındaki request'leri temizle
    i := 0
    for i < len(timestamps) && timestamps[i].Before(cutoff) {
        i++
    }

    timestamps = timestamps[i:]

    // Limit dolmuşsa
    if len(timestamps) >= r.limit {
        r.requests[userID] = timestamps
        return false
    }

    // Yeni request'i ekle
    timestamps = append(timestamps, now)
    r.requests[userID] = timestamps

    return true
}
