package lb

type LeastConnections struct {
	Backends []*Backend
}

func (lc *LeastConnections) Next() string {
	var best *Backend
	var minReq int64 = -1
	var active int64
	for _, backend := range lc.Backends {
		if !backend.Alive {
			continue
		}
		active = backend.GetActiveRequests()
		if minReq == -1 || active < minReq {
			best = backend
			minReq = active
		}
	}
	return best.Addr
}
