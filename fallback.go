package traefik_fallback_plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"
)

// Config the plugin configuration.
type Config struct {
	FallbackOnStatusCodes []string `json:"fallbackOnStatusCodes,omitempty"`
	FallbackURL           string   `json:"fallbackURL,omitempty"`
	ExpectedStatusCode    string   `json:"expectedStatusCode,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// Fallback plugin.
type Fallback struct {
	next          http.Handler
	name          string
	fallbackURL   string
	fallbackCodes map[int]struct{}
	ctx           context.Context
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	statusCodes := map[int]struct{}{}
	for _, code := range config.FallbackOnStatusCodes {
		parsedCode, err := strconv.Atoi(code)
		if err != nil {
			return nil, fmt.Errorf("invalid status code: %s", code)
		}

		statusCodes[parsedCode] = struct{}{}
	}

	return &Fallback{
		fallbackURL:   config.FallbackURL,
		next:          next,
		name:          name,
		fallbackCodes: statusCodes,
		ctx:           ctx,
	}, nil
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

		ctx, cancel := context.WithTimeout(f.ctx, 3*time.Second)
		hasResponse := false

		go func() {
			f.next.ServeHTTP(recorder, req)
			hasResponse = true
			cancel()
		}()

		_ = <-ctx.Done()

		_, ok := f.fallbackCodes[recorder.Code]

		if !hasResponse || ok {
			rw.WriteHeader(200) // todo
			//_, _ = rw.Write(recorder.Body.Bytes())
			_, _ = rw.Write([]byte("hello"))
		} else {
			rw.WriteHeader(recorder.Code)
			_, _ = rw.Write(recorder.Body.Bytes())
		}

		targetHeaders := rw.Header()

		for name, values := range recorder.Header() {
			targetHeaders[name] = values
		}
	})
}
