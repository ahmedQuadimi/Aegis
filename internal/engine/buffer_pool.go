package engine

import "sync"

const DefaultBufferSize = 32 * 1024

var bufferPool = sync.Pool {
	New: func() any {
		return make([]byte, DefaultBufferSize)
	},
}

