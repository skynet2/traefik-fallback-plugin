package traefik_fallback_plugin

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"
)

// Config the plugin configuration.
type Config struct {
	FallbackOnStatusCodes string `json:"fallbackOnStatusCodes,omitempty"`
	FallbackURL           string `json:"fallbackURL,omitempty"`
	FallbackStatusCode    string `json:"fallbackStatusCode"`
	FallbackContentType   string `json:"fallbackContentType,omitempty"`
	UpstreamTimeout       string `json:"upstreamTimeout,omitempty"`
	CacheTTL              string `json:"cacheTTL,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// Fallback plugin.
type Fallback struct {
	next                http.Handler
	name                string
	fallbackCodes       map[int]struct{}
	ctx                 context.Context
	fallbackStatusCode  int
	timeout             time.Duration
	fallbackContentType string
	fetcher             Fetcher
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	statusCodes := map[int]struct{}{}
	for _, code := range strings.Split(config.FallbackOnStatusCodes, ",") {
		parsedCode, err := strconv.Atoi(strings.TrimSpace(code))
		if err != nil {
			return nil, fmt.Errorf("invalid status code: %s", code)
		}

		statusCodes[parsedCode] = struct{}{}
	}

	f := &Fallback{
		next:                next,
		name:                name,
		fallbackCodes:       statusCodes,
		fallbackStatusCode:  http.StatusOK,
		timeout:             3 * time.Second,
		ctx:                 ctx,
		fallbackContentType: config.FallbackContentType,
	}

	if config.FallbackStatusCode != "" {
		statusCode, statusCodeErr := strconv.Atoi(config.FallbackStatusCode)
		if statusCodeErr != nil {
			return nil, fmt.Errorf("invalid fallback status code: %s", config.FallbackStatusCode)
		}

		f.fallbackStatusCode = statusCode
	}

	if config.UpstreamTimeout != "" {
		timeout, timeoutErr := time.ParseDuration(config.UpstreamTimeout)
		if timeoutErr != nil {
			return nil, fmt.Errorf("invalid timeout: %s", config.UpstreamTimeout)
		}

		f.timeout = timeout
	}

	cacheTTL := 1 * time.Minute
	if config.CacheTTL != "" {
		parsedTTL, cacheErr := time.ParseDuration(config.CacheTTL)
		if cacheErr != nil {
			return nil, fmt.Errorf("invalid cacheTTL: %s", config.CacheTTL)
		}

		cacheTTL = parsedTTL
	}

	cache := NewDefaultCache()

	f.fetcher = NewHttpFetcher(
		http.DefaultClient,
		cache,
		config.FallbackURL,
		cacheTTL,
		f.timeout,
	)

	return f, nil
}

func (f *Fallback) SetFetcher(fetcher Fetcher) {
	f.fetcher = fetcher
}

func (f *Fallback) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	f.handler().ServeHTTP(writer, request)
}

func (f *Fallback) handler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if !f.fetcher.CanFetch() || len(f.fallbackCodes) == 0 {
			f.next.ServeHTTP(rw, req)
			return
		}

		recorder := httptest.NewRecorder()

		ctx, cancel := context.WithTimeout(f.ctx, f.timeout)
		hasResponse := false

		go func() {
			defer cancel()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic: %+v", r)
				}
			}()

			req = req.WithContext(ctx)
			f.next.ServeHTTP(recorder, req)
			hasResponse = true
		}()

		<-ctx.Done()

		ctx = f.ctx // swap context

		_, ok := f.fallbackCodes[recorder.Code]

		if !hasResponse || ok { // fallback
			fallBackData, err := f.fetcher.Fetch(ctx)
			if err != nil {
				rw.WriteHeader(http.StatusTeapot)
				_, _ = rw.Write([]byte(err.Error()))
				return
			}

			rw.WriteHeader(f.fallbackStatusCode)

			if f.fallbackContentType != "" {
				rw.Header().Set("Content-Type", f.fallbackContentType)
			} else if fallBackData.ContentType != "" {
				rw.Header().Set("Content-Type", fallBackData.ContentType)
			}

			if fallBackData.Body != nil {
				_, _ = rw.Write(fallBackData.Body)
			}

			return
		}

		for name, values := range recorder.Header() {
			rw.Header()[name] = values
		}

		rw.WriteHeader(recorder.Code)

		if recorder.Body != nil {
			_, _ = rw.Write(recorder.Body.Bytes())
		}
	})
}
