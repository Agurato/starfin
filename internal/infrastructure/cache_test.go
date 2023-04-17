package infrastructure_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Agurato/starfin/internal/infrastructure"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	const cachePath = "../../cache"
	cache := infrastructure.NewCache(cachePath)
	os.RemoveAll(cachePath + "/test")

	t.Run("GetCachedPath", func(t *testing.T) {
		path := "/foo/bar"
		abs, err := filepath.Abs(cachePath + path)
		assert.NoError(t, err)
		assert.Equal(t, abs, cache.GetCachedPath(path))
	})

	t.Run("CacheFile", func(t *testing.T) {
		outputFile := "test/image.jpg"
		_, err := cache.CacheFile("https://image.tmdb.org/t/p/w342/zwzWCmH72OSC9NA0ipoqw5Zjya8.jpg", outputFile)
		assert.NoError(t, err)
		info, err := os.Stat(cache.GetCachedPath(outputFile))
		assert.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})
}
