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

var cache = sync.Map{}
