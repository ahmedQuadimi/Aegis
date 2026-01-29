package lb

import (
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/ahmedQuadimi/Aegis/internal/adapters/config"
)

type Backend struct {
	Addr           string
	Alive          bool
	failureCount   int
	successCount   int
	Config         config.HealthCheckConfig
	mux            sync.RWMutex
	ActiveRequests int64
}

func (b *Backend) Inc() {
	atomic.AddInt64(&b.ActiveRequests, 1)
}

func (b *Backend) Dec() {
	atomic.AddInt64(&b.ActiveRequests, -1)
}

func (b *Backend) GetActiveRequests() int64 {
	return atomic.LoadInt64(&b.ActiveRequests)
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	alive := b.Alive
	b.mux.RUnlock()
	return alive
}

func (b *Backend) UpdateStatus(isHealth bool) {
	b.mux.Lock()
	defer b.mux.Unlock()

	if isHealth {
		b.failureCount = 0
		b.successCount++
		if b.successCount >= b.Config.HealthyThreshold && !b.Alive {
			b.Alive = true
			slog.Info("Backend status changed",
				"backend", b.Addr,
				"status", "HEALTHY",
				"reason", "threshold_met",
			)
		}
	} else {
		b.successCount = 0
		b.failureCount++
		if b.failureCount >= b.Config.UnhealthyThreshold && b.Alive {
			b.Alive = false
			slog.Warn("Backend status changed",
				"backend", b.Addr,
				"status", "UNHEALTHY",
				"reason", "threshold_met",
			)
		}
	}
}

func (b *Backend) GetHost() (string, error) {
	u, err := url.Parse(b.Addr)
	if err != nil {
		slog.Error("Invalid backend address configuration",
			"backend", b.Addr,
			"error", err.Error(),
		)
		return "", err
	}
	return u.Host, nil
}
