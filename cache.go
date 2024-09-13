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

var cache = sync.Map{}

func getFromURL(
	ctx context.Context,
	targetURL string,
	timeout time.Duration,
) (*CacheRecord, error) {
	if rec, ok := cache.Load(targetURL); ok {
		cachedRecord := rec.(*CacheRecord)

		if !cachedRecord.IsExpired() {
			return cachedRecord, nil
		}
	}

	DefaultMutex.Lock(targetURL)
	defer DefaultMutex.Unlock(targetURL)

	if rec, ok := cache.Load(targetURL); ok {
		cachedRecord := rec.(*CacheRecord)

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

	cache.Store(targetURL, rec)

	return rec, nil
}
