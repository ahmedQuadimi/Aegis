package engine

import (
	"net/http"
	"net/http/httputil"
)

type Dispatcher struct {
	routes map[string]*httputil.ReverseProxy
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		routes: make(map[string]*httputil.ReverseProxy),
	}
}

func (d *Dispatcher) AddRoute(host string, proxy *httputil.ReverseProxy) {
	d.routes[host] = proxy
}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if proxy, ok := d.routes[r.Host]; ok {
		proxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Not Found", http.StatusNotFound)
}
