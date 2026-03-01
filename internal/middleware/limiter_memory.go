package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ahmedQuadimi/Aegis/internal/adapters/config"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	limiters map[string]*visitor
	mu       sync.RWMutex
	r        rate.Limit
	b        int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewIPRateLimiter(r rate.Limit, b int, cfg config.RouteConfig) *IPRateLimiter {
	limiter := &IPRateLimiter{
		limiters: make(map[string]*visitor),
		r:        r,
		b:        b,
	}

	go limiter.cleanupVisitors(cfg)
	return limiter
}

func (i *IPRateLimiter) cleanupVisitors(cfg config.RouteConfig) {
	for {
		time.Sleep(time.Minute)

		i.mu.Lock()
		for ip, v := range i.limiters {
			if time.Since(v.lastSeen) > (time.Duration(cfg.CleanupDuration) * time.Millisecond) {
				delete(i.limiters, ip)
			}
		}
		i.mu.Unlock()
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	v, exists := i.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(i.r, i.b)
		i.limiters[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (i *IPRateLimiter) RateLimitMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		limiter := i.GetLimiter(ip)
		if limiter.Allow() {
			next.ServeHTTP(w, r)
		} else {
			slog.Warn("Rate limit exceeded", "ip", ip)
			http.Error(w, "Too Many Requests Error 429", http.StatusTooManyRequests)
		}
	}
}
