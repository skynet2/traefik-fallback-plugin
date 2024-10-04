package traefik_fallback_plugin

import (
	"context"
	"net/http"
)

//go:generate mockgen -destination interfaces_mocks_test.go -package traefik_fallback_plugin_test -source=interfaces.go

type Fetcher interface {
	Fetch(ctx context.Context) (*CacheRecord, error)
	CanFetch() bool
}

type Cache interface {
	Load(key string) (*CacheRecord, bool)
	Store(key string, value *CacheRecord)
}

type Transport interface {
	http.RoundTripper
}
