package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ahmedQuadimi/Aegis/internal/adapters/config"
	redis_adapter "github.com/ahmedQuadimi/Aegis/internal/redis"
)

func RedisRateLimiter(next http.Handler, rdb *redis_adapter.Client, cfg config.RouteConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		ip = strings.Trim(ip, "[]")
		key := "ratelimit:" + ip

		limit := cfg.RateLimit

		allowed, err := rdb.AllowRequest(r.Context(), key, limit, time.Second)

		if err != nil {
			slog.Error("Redis error", "err", err)
			next.ServeHTTP(w, r)
			return
		}

		if !allowed {
			slog.Warn("limit exceeded", "ip", ip)
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Too Many Requests (Redis)", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
