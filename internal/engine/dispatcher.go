package engine

import (
	"net"
	"net/http"
	"strings"
	"sync"
)

type Dispatcher struct {
	routes map[string]http.Handler
	mu     sync.RWMutex
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		routes: make(map[string]http.Handler),
	}
}

func (d *Dispatcher) AddRoute(host string, handler http.Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.routes[host] = handler
}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if strings.Contains(host, ":") {
		h, _, err := net.SplitHostPort(host)
		if err == nil {
			host = h
		}
	}
	d.mu.RLock()
	handler, ok := d.routes[host]
	d.mu.RUnlock()

	if ok {
		handler.ServeHTTP(w, r)
		return
	}

	http.Error(w, "Aegis: Route Not Found", http.StatusNotFound)
}
