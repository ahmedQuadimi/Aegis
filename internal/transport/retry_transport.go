package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ahmedQuadimi/Aegis/internal/lb"
)

type RetryTransport struct {
	http.RoundTripper
	MaxRetries    int
	MaxRetryBytes int64
	Balancer      lb.Balancer
	BackendMap    map[string]*lb.Backend
}

func (rt *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return rt.runRetryLoop(req, nil)
	}

	if req.ContentLength == -1 || req.ContentLength > rt.MaxRetryBytes {
		slog.Warn("Payload too large, skipping retry logic",
			"content_length", req.ContentLength,
			"max_limit", rt.MaxRetryBytes,
			"action", "streaming_single_shot",
		)
		return rt.runSingleShot(req)
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()

	return rt.runRetryLoop(req, bodyBytes)
}

func (rt *RetryTransport) runSingleShot(req *http.Request) (*http.Response, error) {
	targetAddr := rt.Balancer.Next()
	if targetAddr == "" {
		return nil, fmt.Errorf("no healthy backends")
	}

	rt.setTarget(req, targetAddr)

	return rt.RoundTripper.RoundTrip(req)
}

func (rt *RetryTransport) runRetryLoop(req *http.Request, bodyBytes []byte) (*http.Response, error) {
	var lastErr error

	for i := 0; i <= rt.MaxRetries; i++ {
		targetAddr := rt.Balancer.Next()
		if targetAddr == "" {
			return nil, fmt.Errorf("no healthy backends")
		}
		rt.setTarget(req, targetAddr)
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}

		resp, err := rt.RoundTripper.RoundTrip(req)
		if err == nil {
			return resp, nil
		}

		if b, ok := rt.BackendMap[targetAddr]; ok {
			b.UpdateStatus(false)
		}

		slog.Warn("Retry attempt failed",
			"attempt", i+1,
			"max_retries", rt.MaxRetries,
			"backend", targetAddr,
			"error", err.Error(),
		)
		lastErr = err
	}

	return nil, lastErr
}

func (rt *RetryTransport) setTarget(req *http.Request, addr string) {
	targetURL, _ := url.Parse(addr)
	req.URL.Scheme = targetURL.Scheme
	req.URL.Host = targetURL.Host

	ctx := context.WithValue(req.Context(), "target", addr)
	*req = *req.WithContext(ctx)
}
