package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ahmedQuadimi/Aegis/internal/observability"
)

type statusRecorder struct {
	http.ResponseWriter
	StatusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.StatusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, StatusCode: http.StatusOK}
		next.ServeHTTP(rec, r)
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rec.StatusCode)
		host := r.Host

		observability.RequestsTotal.WithLabelValues(status, r.Method, host).Inc()
		observability.RequestDuration.WithLabelValues(r.Method, host).Observe(duration)
	})
}
