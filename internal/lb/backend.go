package lb

import (
	"log"
	"net/url"
	"sync"

	"github.com/ahmedQuadimi/Aegis/internal/adapters/config"
)

type Backend struct {
	Addr         string
	Alive        bool
	failureCount int
	successCount int
	Config       config.HealthCheckConfig
	mux          sync.RWMutex
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
			log.Printf("Backend %s is now HEALTHY", b.Addr)
		}
	} else {
		b.successCount = 0
		b.failureCount++
		if b.failureCount >= b.Config.UnhealthyThreshold && b.Alive {
			b.Alive = false
			log.Printf("Backend %s is now UNHEALTHY", b.Addr)
		}
	}
}

func (b *Backend) GetHost() (string, error) {
	u, err := url.Parse(b.Addr)
	if err != nil {
		log.Fatal("Invalid backend address:", b.Addr)
		return "", err
	}
	return u.Host, nil
}
