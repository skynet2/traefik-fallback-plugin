package traefik_fallback_plugin

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
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

	return &Fallback{
		fallbackURL:         config.FallbackURL,
		next:                next,
		name:                name,
		fallbackCodes:       statusCodes,
		fallbackStatusCode:  statusCode,
		timeout:             timeout,
		ctx:                 ctx,
		fallbackContentType: config.FallbackContentType,
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

		//recorder := httptest.NewRecorder()

		recorder := &Writer{}

		ctx, cancel := context.WithTimeout(f.ctx, f.timeout)
		hasResponse := false

		go func() {
			f.next.ServeHTTP(recorder, req)
			hasResponse = true
			cancel()
		}()

		_ = <-ctx.Done()

		ctx = f.ctx // swap context

		_, ok := f.fallbackCodes[recorder.StatusCode]

		if !hasResponse || ok { // fallback
			fallBackData, err := getFromURL(ctx, f.fallbackURL, f.timeout)
			if err != nil {
				rw.WriteHeader(http.StatusTeapot)
				_, _ = rw.Write([]byte(err.Error()))
				return
			}

			rw.WriteHeader(f.fallbackStatusCode)
			_, _ = rw.Write(fallBackData.Body)

			if f.fallbackContentType != "" {
				rw.Header().Set("Content-Type", f.fallbackContentType)
			} else {
				rw.Header().Set("Content-Type", fallBackData.ContentType)
			}

			return
		}

		rw.WriteHeader(recorder.StatusCode)
		for name, values := range recorder.Header() {
			rw.Header()[name] = values
		}
		//
		//rw.Header().Del("Content-Encoding")
		//rw.Header().Del("Content-Length")
		_, _ = rw.Write(recorder.Buf.Bytes())

		//if recorder.Header().Get("Content-Encoding") == "gzip" {
		//	data, err := gUnzipData(recorder.Body.Bytes())
		//	if err != nil {
		//		rw.WriteHeader(http.StatusTeapot)
		//		_, _ = rw.Write([]byte(err.Error()))
		//		return
		//	}
		//
		//	log.Printf("Body UnGziped: %s", string(data))
		//
		//	_, _ = rw.Write(data)
		//	return
		//} else {
		//	log.Printf("Body: %s", recorder.Body.String())
		//
		//	_, _ = rw.Write(recorder.Body.Bytes())
		//}
	})
}

func gUnzipData(data []byte) (resData []byte, err error) {
	b := bytes.NewBuffer(data)

	var r io.Reader
	r, err = gzip.NewReader(b)
	if err != nil {
		return
	}

	var resB bytes.Buffer
	_, err = resB.ReadFrom(r)
	if err != nil {
		return
	}

	resData = resB.Bytes()

	return
}
