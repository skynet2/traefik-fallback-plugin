package traefik_fallback_plugin_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	traefik_fallback_plugin "github.com/skynet2/traefik-fallback-plugin"
)

func TestFetcher(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		transport := NewMockTransport(gomock.NewController(t))

		cl := &http.Client{
			Transport: transport,
		}
		cache := NewMockCache(gomock.NewController(t))

		fc := traefik_fallback_plugin.NewHttpFetcher(
			cl,
			cache,
			"http://example.com/index.html",
			30*time.Second,
			60*time.Second)

		cache.EXPECT().Load("http://example.com/index.html").
			Return(nil, false)

		cache.EXPECT().Load("http://example.com/index.html").
			Return(nil, false)

		cache.EXPECT().Store("http://example.com/index.html", gomock.Any()).
			DoAndReturn(func(s string, record *traefik_fallback_plugin.CacheRecord) {
				assert.EqualValues(t, "http://example.com/index.html", s)
				assert.NotNil(t, record)
				assert.WithinDuration(t, record.ExpiresAt, time.Now().UTC(), 30*time.Second)
			})

		transport.EXPECT().RoundTrip(gomock.Any()).
			DoAndReturn(func(request *http.Request) (*http.Response, error) {
				assert.EqualValues(t, "http://example.com/index.html", request.URL.String())
				assert.EqualValues(t, http.MethodGet, request.Method)

				return &http.Response{
					StatusCode:    http.StatusOK,
					Body:          io.NopCloser(bytes.NewBuffer([]byte("test"))),
					ContentLength: 1,
				}, nil
			})

		assert.True(t, fc.CanFetch())

		record, err := fc.Fetch(context.TODO())
		assert.NoError(t, err)
		assert.NotNil(t, record)
		assert.EqualValues(t, "test", string(record.Body))
	})
}

func TestFetcherCache(t *testing.T) {
	t.Run("success from cache", func(t *testing.T) {
		cache := NewMockCache(gomock.NewController(t))

		rec := &traefik_fallback_plugin.CacheRecord{
			ExpiresAt: time.Now().Add(30 * time.Second),
		}

		fc := traefik_fallback_plugin.NewHttpFetcher(
			nil,
			cache,
			"http://example.com/index.html",
			30*time.Second,
			60*time.Second)

		cache.EXPECT().Load("http://example.com/index.html").
			Return(rec, true)

		resp, err := fc.Fetch(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, rec, resp)
	})

	t.Run("first expired", func(t *testing.T) {
		cache := NewMockCache(gomock.NewController(t))

		rec := &traefik_fallback_plugin.CacheRecord{
			ExpiresAt: time.Now().Add(-30 * time.Second),
		}

		rec2 := &traefik_fallback_plugin.CacheRecord{
			ExpiresAt: time.Now().Add(30 * time.Second),
		}

		fc := traefik_fallback_plugin.NewHttpFetcher(
			nil,
			cache,
			"http://example.com/index.html",
			30*time.Second,
			60*time.Second)

		cache.EXPECT().Load("http://example.com/index.html").
			Return(rec, true)

		cache.EXPECT().Load("http://example.com/index.html").
			Return(rec2, true)

		resp, err := fc.Fetch(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, rec2, resp)
	})
}
