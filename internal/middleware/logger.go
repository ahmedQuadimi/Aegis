package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

func SetupLoggerMiddleware() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &StatusRecorder{ResponseWriter: w, Status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		duration := time.Since(start)
		slog.Info("Request processed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.Status,
			"duration_ms", duration.Milliseconds(),
			"ip", r.RemoteAddr,
			"bytes", recorder.Written,
			"user_agent", r.UserAgent(),
		)
	})
}
