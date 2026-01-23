package engine

import (
	"net/http"
	"net/http/httputil"
)

type Discpatcher struct {
	routes map[string]*httputil.ReverseProxy
}

func NewDiscpatcher() *Discpatcher {
	return &Discpatcher{
		routes: make(map[string]*httputil.ReverseProxy),
	}
}

func (d *Discpatcher) AddRoute(host string, proxy *httputil.ReverseProxy) {
	d.routes[host] = proxy
}

func (d *Discpatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if proxy, ok := d.routes[r.Host]; ok {
		proxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Not Found", http.StatusNotFound)
}
