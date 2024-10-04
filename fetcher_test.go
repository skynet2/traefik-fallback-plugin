package traefik_fallback_plugin_test

import (
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
		fc := traefik_fallback_plugin.NewHttpFetcher(
			cl,
			"http://example.com/index.html",
			30*time.Second,
			60*time.Second)

		transport.EXPECT().RoundTrip(gomock.Any()).
			DoAndReturn(func(request *http.Request) (*http.Response, error) {
				assert.EqualValues(t, "http://example.com/index.html", request.URL.String())
				assert.EqualValues(t, http.MethodGet, request.Method)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(nil),
				}, nil
			})

		assert.True(t, fc.CanFetch())

		record, err := fc.Fetch(context.TODO())
		assert.NoError(t, err)
		assert.NotNil(t, record)
	})
}
