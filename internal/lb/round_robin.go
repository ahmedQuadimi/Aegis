package lb

import (
	"sync/atomic"
)

type RoundRobin struct {
	counter atomic.Uint64
}

func (robin *RoundRobin) NextServer(servers []string) string {
	if len(servers) == 0 {
		return ""
	}
	val := robin.counter.Add(1)
	return servers[int(val - 1) % len(servers)]
}
