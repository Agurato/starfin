package cache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Agurato/starfin/internal/cache"
	"github.com/stretchr/testify/assert"
)

const (
	cachePath = "../../cache"
)

func TestMain(m *testing.M) {
	os.Setenv("CACHE_PATH", cachePath)
	cache.InitCache()
	os.RemoveAll(cachePath + "/test")
	result := m.Run()
	os.Exit(result)
}

func TestGetCachedPath(t *testing.T) {
	path := "/foo/bar"
	abs, err := filepath.Abs(cachePath + path)
	assert.NoError(t, err)
	assert.Equal(t, abs, cache.GetCachedPath(path))
}

func TestCacheFile(t *testing.T) {
	outputFile := "test/image.jpg"
	err := cache.CacheFile("https://image.tmdb.org/t/p/w342/zwzWCmH72OSC9NA0ipoqw5Zjya8.jpg", outputFile)
	assert.NoError(t, err)
	info, err := os.Stat(cache.GetCachedPath(outputFile))
	assert.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}
