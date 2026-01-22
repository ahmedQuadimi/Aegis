package engine

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	lb "github.com/ahmedQuadimi/Aegis/internal/lb"
)

type Engine struct {
	balancer lb.Balancer
	proxy    *httputil.ReverseProxy
}

func NewEngine(balancer lb.Balancer, pool *sync.Pool) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		targetAddress := balancer.Next()
		target, err := url.Parse(targetAddress)
		if err != nil {
			panic(err)
		}

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
	}
	return &httputil.ReverseProxy{
		Director:   director,
		BufferPool: &PoolBuffer{pool: pool},
	}
}
