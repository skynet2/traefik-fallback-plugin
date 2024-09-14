package traefik_fallback_plugin_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	traefik_fallback_plugin "github.com/skynet2/traefik-fallback-plugin"
)

func TestNewFallbackInvalidStatusCode(t *testing.T) {
	ctx := context.Background()
	_, err := traefik_fallback_plugin.New(ctx, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), &traefik_fallback_plugin.Config{
		FallbackOnStatusCodes: "invalid",
	}, "test")

	assert.Error(t, err)
}

func TestNewFallbackInvalidFallbackStatusCode(t *testing.T) {
	ctx := context.Background()
	_, err := traefik_fallback_plugin.New(ctx, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), &traefik_fallback_plugin.Config{
		FallbackOnStatusCodes: "404",
		FallbackStatusCode:    "invalid",
	}, "test")

	assert.Error(t, err)
}

func TestNewFallbackInvalidUpstreamTimeout(t *testing.T) {
	ctx := context.Background()
	_, err := traefik_fallback_plugin.New(ctx, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), &traefik_fallback_plugin.Config{
		UpstreamTimeout: "invalid",
	}, "test")

	assert.Error(t, err)
}

func TestNewFallbackInvalidCacheTTL(t *testing.T) {
	ctx := context.Background()
	_, err := traefik_fallback_plugin.New(ctx, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), &traefik_fallback_plugin.Config{
		CacheTTL: "invalid",
	}, "test")

	assert.Error(t, err)
}

func TestFallbackServeHTTPWithoutFallback(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	fallback, err := traefik_fallback_plugin.New(
		context.Background(),
		handler,
		&traefik_fallback_plugin.Config{
			FallbackOnStatusCodes: "200,201",
			FallbackStatusCode:    "409",
		}, "test",
	)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	fallback.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestFallbackServeHTTPWithFallback(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	fallback, _ := traefik_fallback_plugin.New(context.Background(), handler, &traefik_fallback_plugin.Config{
		FallbackOnStatusCodes: "500",
		FallbackURL:           "http://example.com",
		FallbackStatusCode:    "200",
		FallbackContentType:   "application/xx",
	}, "test")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	fetcher := NewMockFetcher(gomock.NewController(t))
	fetcher.EXPECT().CanFetch().Return(true)
	fetcher.EXPECT().Fetch(gomock.Any()).Return(nil, errors.New("unexpected err"))

	fallback.(*traefik_fallback_plugin.Fallback).SetFetcher(fetcher)
	fallback.ServeHTTP(rec, req)

	assert.Equal(t, 418, rec.Code)
	assert.Equal(t, "unexpected err", rec.Body.String())
	assert.Equal(t, "", rec.Header().Get("Content-Type"))
}

func TestFallbackServeHTTPWithFallbackSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	fallback, _ := traefik_fallback_plugin.New(context.Background(), handler, &traefik_fallback_plugin.Config{
		FallbackOnStatusCodes: "500",
		FallbackURL:           "http://example.com",
		FallbackStatusCode:    "200",
		FallbackContentType:   "application/xx",
	}, "test")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	fetcher := NewMockFetcher(gomock.NewController(t))
	fetcher.EXPECT().CanFetch().Return(true)
	fetcher.EXPECT().Fetch(gomock.Any()).Return(&traefik_fallback_plugin.CacheRecord{
		Body:        []byte("content"),
		ContentType: "ignored",
	}, nil)

	fallback.(*traefik_fallback_plugin.Fallback).SetFetcher(fetcher)
	fallback.ServeHTTP(rec, req)

	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "content", rec.Body.String())
	assert.Equal(t, "application/xx", rec.Header().Get("Content-Type"))
}
