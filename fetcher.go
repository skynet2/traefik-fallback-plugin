package traefik_fallback_plugin

import (
	"context"
	"io"
	"net/http"
	"time"
)

type HttpFetcher struct {
	targetURL string
	timeout   time.Duration
	cacheTTL  time.Duration
}

func NewHttpFetcher(
	targetURL string,
	cacheTTL time.Duration,
	timeout time.Duration,
) *HttpFetcher {
	return &HttpFetcher{
		targetURL: targetURL,
		cacheTTL:  cacheTTL,
		timeout:   timeout,
	}
}

func (h *HttpFetcher) CanFetch() bool {
	return h.targetURL != ""
}

func (h *HttpFetcher) Fetch(
	ctx context.Context,
) (*CacheRecord, error) {
	if rec, ok := cache.Load(h.targetURL); ok {
		cachedRecord := rec.(*CacheRecord)

		if !cachedRecord.IsExpired() {
			return cachedRecord, nil
		}
	}

	DefaultMutex.Lock(h.targetURL)
	defer DefaultMutex.Unlock(h.targetURL)

	if rec, ok := cache.Load(h.targetURL); ok {
		cachedRecord := rec.(*CacheRecord)

		if !cachedRecord.IsExpired() {
			return cachedRecord, nil
		}
	}

	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.targetURL, nil)
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
		ExpiresAt:   time.Now().Add(h.cacheTTL),
	}

	cache.Store(h.targetURL, rec)

	return rec, nil
}
