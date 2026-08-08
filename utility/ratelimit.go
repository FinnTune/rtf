package utility

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter tracks a token-bucket rate limiter per client IP.
type IPRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	r        rate.Limit
	b        int
}

// NewIPRateLimiter creates a limiter allowing r requests/second per IP, with
// burst b. Idle IPs are forgotten after a few minutes so memory doesn't grow
// unbounded.
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	rl := &IPRateLimiter{
		visitors: make(map[string]*visitor),
		r:        r,
		b:        b,
	}
	go rl.cleanupVisitors()
	return rl
}

func (rl *IPRateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.r, rl.b)
		rl.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *IPRateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Limit wraps an http.HandlerFunc, rejecting requests over the configured
// per-IP rate with 429 Too Many Requests.
func (rl *IPRateLimiter) Limit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.getVisitor(clientIP(r)).Allow() {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}
