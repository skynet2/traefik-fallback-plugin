package traefik_fallback_plugin

import (
	"context"
	"io"
	"net/http"
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

var mapMut sync.Mutex
var cache = make(map[string]*CacheRecord)

func getFromURL(
	ctx context.Context,
	targetURL string,
	timeout time.Duration,
) (*CacheRecord, error) {
	if cachedRecord, ok := cache[targetURL]; ok {
		if !cachedRecord.IsExpired() {
			return cachedRecord, nil
		}
	}

	DefaultMutex.Lock(targetURL)
	defer DefaultMutex.Unlock(targetURL)

	if cachedRecord, ok := cache[targetURL]; ok {
		if !cachedRecord.IsExpired() {
			return cachedRecord, nil
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rec := &CacheRecord{
		Body:        body,
		ContentType: resp.Header.Get("Content-Type"),
		ExpiresAt:   time.Now().Add(time.Minute),
	}

	mapMut.Lock()
	cache[targetURL] = rec
	defer mapMut.Unlock()

	return rec, nil
}
