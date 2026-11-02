package server

import (
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	rateWindow   = time.Minute
	rateMaxFails = 20
)

type rateLimiter struct {
	mu    sync.Mutex
	fails map[string][]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{fails: make(map[string][]time.Time)}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cut := now.Add(-rateWindow)
	list := rl.fails[ip]
	kept := list[:0]
	for _, t := range list {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	rl.fails[ip] = kept
	return len(kept) < rateMaxFails
}

func (rl *rateLimiter) fail(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.fails[ip] = append(rl.fails[ip], time.Now())
}
