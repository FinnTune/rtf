package utility

import (
	"math"
	"net"
	"net/http"
	"strconv"
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
// per-IP rate with 429 Too Many Requests and a Retry-After header telling a
// well-behaved client how long to wait before trying again.
func (rl *IPRateLimiter) Limit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Reserve()+Cancel() (rather than Allow()) is the standard way to
		// get a wait-time estimate out of x/time/rate without side effects
		// when the request turns out to be rejected: a reservation with
		// delay 0 is exactly what Allow() would have accepted, and
		// cancelling one that isn't used restores the token it provisionally
		// took, leaving the bucket exactly as Allow() would have left it.
		reservation := rl.getVisitor(clientIP(r)).Reserve()
		if !reservation.OK() {
			// Only possible with burst 0, which nothing in this codebase
			// configures — fail closed rather than let it through unbounded.
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		if delay := reservation.Delay(); delay > 0 {
			reservation.Cancel()
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(delay.Seconds()))))
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}
