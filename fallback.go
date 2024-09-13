package traefik_fallback_plugin

import (
	"context"
	"fmt"
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
	fallbackURL         string
	fallbackCodes       map[int]struct{}
	ctx                 context.Context
	fallbackStatusCode  int
	timeout             time.Duration
	fallbackContentType string
	cacheTTL            time.Duration
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

	statusCode, err := strconv.Atoi(config.FallbackStatusCode)
	if err != nil {
		return nil, fmt.Errorf("invalid fallback status code: %s", config.FallbackStatusCode)
	}

	timeout, err := time.ParseDuration(config.UpstreamTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %s", config.UpstreamTimeout)
	}

	if timeout == 0 {
		timeout = 3 * time.Second
	}

	f := &Fallback{
		fallbackURL:         config.FallbackURL,
		next:                next,
		name:                name,
		fallbackCodes:       statusCodes,
		fallbackStatusCode:  statusCode,
		timeout:             timeout,
		ctx:                 ctx,
		fallbackContentType: config.FallbackContentType,
		cacheTTL:            3 * time.Minute,
	}

	if config.CacheTTL != "" {
		cacheTTL, cacheErr := time.ParseDuration(config.CacheTTL)
		if cacheErr != nil {
			return nil, fmt.Errorf("invalid cacheTTL: %s", config.CacheTTL)
		}

		f.cacheTTL = cacheTTL
	}

	return f, nil
}

func (f *Fallback) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	f.handler().ServeHTTP(writer, request)
}

func (f *Fallback) handler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if f.fallbackURL == "" || len(f.fallbackCodes) == 0 {
			f.next.ServeHTTP(rw, req)
			return
		}

		recorder := httptest.NewRecorder()

		ctx, cancel := context.WithTimeout(f.ctx, f.timeout)
		hasResponse := false

		go func() {
			f.next.ServeHTTP(recorder, req)
			hasResponse = true
			cancel()
		}()

		<-ctx.Done()

		ctx = f.ctx // swap context

		_, ok := f.fallbackCodes[recorder.Code]

		if !hasResponse || ok { // fallback
			fallBackData, err := getFromURL(ctx, f.fallbackURL, f.timeout)
			if err != nil {
				rw.WriteHeader(http.StatusTeapot)
				_, _ = rw.Write([]byte(err.Error()))
				return
			}

			rw.WriteHeader(f.fallbackStatusCode)

			if f.fallbackContentType != "" {
				rw.Header().Set("Content-Type", f.fallbackContentType)
			} else {
				rw.Header().Set("Content-Type", fallBackData.ContentType)
			}

			_, _ = rw.Write(fallBackData.Body)

			return
		}

		for name, values := range recorder.Header() {
			rw.Header()[name] = values
		}

		rw.WriteHeader(recorder.Code)
		_, _ = rw.Write(recorder.Body.Bytes())
	})
}
