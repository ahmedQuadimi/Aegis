package cache

import (
	"time"
)

type Storage interface {
	Get(key string) (*CacheItem, bool)
	Set(key string, body []byte, header map[string][]string, ttl time.Duration)
}
