package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"time"

	"github.com/ahmedQuadimi/Aegis/internal/cache"
)

type ResponseInterceptor struct {
	http.ResponseWriter
	StatusCode int
	Body       *bytes.Buffer
}

func (r *ResponseInterceptor) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ResponseInterceptor) Write(b []byte) (int, error) {
	r.Body.Write(b)
	return r.ResponseWriter.Write(b)
}

func NewCacheMiddleware(storage *cache.MemoryCache, ttl int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			cacheKey := r.URL.String()
			if cachedItem, found := storage.Get(cacheKey); found {
				slog.Info("Cache HIT", "path", r.URL.Path)
				for key, values := range cachedItem.Header {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
				w.Header().Set("X-Cache", "HIT")
				w.WriteHeader(http.StatusOK)
				w.Write(cachedItem.Body)
				return
			}
			slog.Info("Cache MISS", "path", r.URL.Path)
			interceptor := &ResponseInterceptor{
				ResponseWriter: w,
				Body:           bytes.NewBuffer(nil),
				StatusCode:     http.StatusOK,
			}
			interceptor.Header().Set("X-Cache", "MISS")
			next.ServeHTTP(interceptor, r)
			if interceptor.StatusCode == http.StatusOK {
				storage.Set(cacheKey, interceptor.Body.Bytes(), interceptor.Header(), time.Duration(ttl)*time.Second)
			}
		})
	}
}
