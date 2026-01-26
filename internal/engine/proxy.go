package engine

import (
	"log"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/ahmedQuadimi/Aegis/internal/adapters/config"
	"github.com/ahmedQuadimi/Aegis/internal/lb"
	"github.com/ahmedQuadimi/Aegis/internal/transport"
)

type Engine struct {
	balancer lb.Balancer
	proxy    *httputil.ReverseProxy
}

func NewEngine(balancer lb.Balancer, pool *sync.Pool, route config.RouteConfig, backendMap map[string]*lb.Backend) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Origin", "Aegis-Proxy")
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		targetAddr, _ := r.Context().Value("target").(string)
		log.Printf("Passive Check: Backend %s failed: %v", targetAddr, err)
		if b, ok := backendMap[targetAddr]; ok {
			b.UpdateStatus(false)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Aegis: Service Temporarily Unavailable"))
	}
	return &httputil.ReverseProxy{
		Director:     director,
		BufferPool:   &PoolBuffer{pool: pool},
		ErrorHandler: errorHandler,
		Transport: &transport.RetryTransport{
			RoundTripper:  http.DefaultTransport,
			Balancer:      balancer,
			MaxRetries:    route.Retries,
			MaxRetryBytes: route.RetryBufferSize,
		},
	}
}
