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
}

var mut = sync.Mutex{}
var cachedRecord *CacheRecord

func getFromURL(ctx context.Context, targetURL string, timeout time.Duration) (*CacheRecord, error) {
	if cachedRecord != nil {
		return cachedRecord, nil
	}

	mut.Lock()
	defer mut.Unlock()
	if cachedRecord != nil {
		return cachedRecord, nil
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

	cachedRecord = &CacheRecord{
		Body:        body,
		ContentType: resp.Header.Get("Content-Type"),
	}

	return cachedRecord, nil
}
