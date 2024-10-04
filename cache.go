package traefik_fallback_plugin

import (
	"sync"
	"time"
)

type CacheRecord struct {
	Body        []byte
	ContentType string
	ExpiresAt   time.Time
}

func (c *CacheRecord) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

type DefaultCache struct {
	cache *sync.Map
}

func NewDefaultCache() *DefaultCache {
	return &DefaultCache{
		cache: &sync.Map{},
	}
}

func (c *DefaultCache) Load(key string) (*CacheRecord, bool) {
	rec, ok := c.cache.Load(key)
	if !ok {
		return nil, false
	}

	converted, ok := rec.(*CacheRecord)
	if !ok {
		return nil, false
	}

	return converted, ok
}

func (c *DefaultCache) Store(key string, value *CacheRecord) {
	c.cache.Store(key, value)
}
