package cache

import (
	"sync"
	"time"
)

type CacheItem struct {
	Body      []byte
	Header    map[string][]string
	ExpiresAt time.Time
}

type MemoryCache struct {
	items map[string]*CacheItem
	mu    sync.RWMutex
}

func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]*CacheItem),
	}
	go cache.cleanupLoop()
	return cache
}

func (c *MemoryCache) Get(key string) (*CacheItem, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists || time.Now().After(item.ExpiresAt) {
		return nil, false
	}
	return item, true
}

func (c *MemoryCache) Set(key string, value []byte, header map[string][]string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &CacheItem{
		Body:      value,
		Header:    header,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		c.mu.Lock()
		for key, item := range c.items {
			if time.Now().After(item.ExpiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
