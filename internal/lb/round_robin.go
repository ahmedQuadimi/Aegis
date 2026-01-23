package lb

import (
	"sync/atomic"
)

type RoundRobin struct {
	counter  atomic.Uint64
	Backends []*Backend
}

func (robin *RoundRobin) Next() string {
	n := uint64(len(robin.Backends))
	if n == 0 {
		return ""
	}
	for i := uint64(0); i < n; i++ {
		idx := robin.counter.Add(1) - 1
		target := robin.Backends[idx%n]
		if target.IsAlive() {
			return target.Addr
		}
	}
	return ""
}
