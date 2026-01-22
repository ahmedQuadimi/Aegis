package engine

import "sync"

const DefaultBufferSize = 32 * 1024

type PoolBuffer struct {
	pool *sync.Pool
}

func (p *PoolBuffer) Get() []byte {
	return p.pool.Get().([]byte)
}

func (p *PoolBuffer) Put(buf []byte) {
	p.pool.Put(buf)
}
