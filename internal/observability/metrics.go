package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var RequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "aegis_requests_total",
		Help: "Total number of HTTP requests processed",
	},
	[]string{"status", "method", "host"},
)

var RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "aegis_request_duration_seconds",
	Help:    "Histogram of request processing duration",
	Buckets: prometheus.DefBuckets,
}, []string{"method", "host"})

var ActiveBackend = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "aegis_active_backends",
	Help: "number of healthy backend per host",
}, []string{"host"})
