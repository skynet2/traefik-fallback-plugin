package traefik_fallback_plugin_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	traefik_fallback_plugin "github.com/skynet2/traefik-fallback-plugin"
)

func TestDefaultCache(t *testing.T) {
	c := traefik_fallback_plugin.NewDefaultCache()

	resp, ok := c.Load("test")
	assert.False(t, ok)
	assert.Nil(t, resp)

	ref := &traefik_fallback_plugin.CacheRecord{}

	c.Store("test", ref)

	resp, ok = c.Load("test")
	assert.True(t, ok)
	assert.Equal(t, ref, resp)

	resp, ok = c.Load("xxx")
	assert.False(t, ok)
	assert.Nil(t, resp)
}
