package lb

import (
	"sync/atomic"
)

type RoundRobin struct {
	counter atomic.Uint64
	Servers []string
}

func (robin *RoundRobin) Next() string {
	if len(robin.Servers) == 0 {
		return ""
	}
	return robin.Servers[int(robin.counter.Add(1)-1)%len(robin.Servers)]
}
