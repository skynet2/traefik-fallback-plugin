package traefik_fallback_plugin

import "context"

//go:generate mockgen -destination interfaces_mocks_test.go -package traefik_fallback_plugin_test -source=interfaces.go

type Fetcher interface {
	Fetch(ctx context.Context) (*CacheRecord, error)
	CanFetch() bool
}
